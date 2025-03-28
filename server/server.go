package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/Seann-Moser/go-serve/pkg/metrics"
	"github.com/Seann-Moser/go-serve/server/middle"
	"golang.org/x/sync/errgroup"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/Seann-Moser/go-serve/pkg/request"
	"github.com/Seann-Moser/go-serve/pkg/response"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/server/endpoint_manager"
	"github.com/Seann-Moser/go-serve/server/endpoints"
)

var VERSION = "dev"
var NAME = "dev"

type Server struct {
	ServingPort      string `yaml:"serving_port" json:"serving_port" env:"SERVING_PORT"`
	ctx              context.Context
	serverCtx        context.Context
	router           *mux.Router
	EndpointManager  *endpoint_manager.Manager
	Response         *response.Response
	Request          *request.Request
	PathPrefix       string
	shutdownDuration time.Duration
	MetricsServer    *metrics.Metrics
	requestTracker   *middle.RequestTracker
	shutdown         func()
	server           *http.Server
}

const (
	serverPortFlag             = "server-port"
	serverPrefixFlag           = "server-path-prefix"
	serverMaxReceivedBytesFlag = "server-max-bytes"
	serverShowErrFlag          = "server-show-err"
)

func Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("server", pflag.ExitOnError)
	fs.String(serverPortFlag, "8080", "")
	fs.String(serverPrefixFlag, "", "")
	fs.Int64(serverMaxReceivedBytesFlag, int64(20*1024*1024), "")
	fs.Bool(serverShowErrFlag, false, "")
	fs.Duration("shutdown-duration", 15*time.Second, "duration to wait before shutting down the server")
	fs.AddFlagSet(metrics.MetricFlags())
	return fs
}

func New(ctx context.Context) *Server {
	return NewServer(ctx,
		viper.GetString(serverPortFlag),
		viper.GetString(serverPrefixFlag),
		viper.GetInt64(serverMaxReceivedBytesFlag),
		viper.GetBool(serverShowErrFlag),
		viper.GetDuration("shutdown-duration"))
}

func NewServer(ctx context.Context, servingPort string, pathPrefix string, mb int64, showErr bool, shutdownDuration time.Duration) *Server {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	notifyContext, cancel := context.WithCancel(ctx)
	go func() {
		osCall := <-c
		println("starting shutdown")
		ctxLogger.Info(ctx, fmt.Sprintf("system call:%+v", osCall))
		cancel()
	}()

	router := mux.NewRouter()
	if !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}

	m := metrics.New()
	if VERSION != "dev" {
		m.Version = VERSION
	}
	if NAME != "dev" {
		m.Name = NAME
	}
	router.Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.EqualFold(r.URL.Path, "/healthcheck") {
				w.WriteHeader(http.StatusOK)
				return
			}
			handler.ServeHTTP(w, r)
		})
	})
	requestTracker := middle.NewRequestTracker()
	router.Use()
	if m.Enabled {
		router.Use(requestTracker.TrackMiddleware)
	}
	serverCtx, cancelRoute := context.WithCancel(ctx)
	return &Server{
		ServingPort:      servingPort,
		serverCtx:        notifyContext,
		ctx:              serverCtx,
		router:           router,
		EndpointManager:  endpoint_manager.NewManager(router),
		Response:         response.NewResponse(showErr),
		Request:          request.NewRequest(mb),
		PathPrefix:       pathPrefix,
		MetricsServer:    m,
		shutdownDuration: shutdownDuration,
		shutdown:         cancelRoute,
		requestTracker:   requestTracker,
	}
}

func (s *Server) GetResponseManager() *response.Response {
	return s.Response
}

func (s *Server) GetContext() context.Context {
	return s.ctx
}

func (s *Server) AddEndpoints(ctx context.Context, endpoint ...*endpoints.Endpoint) error {
	for _, e := range endpoint {
		if s.PathPrefix > "" {
			var err error
			e.URLPath, err = url.JoinPath(s.PathPrefix, e.URLPath)
			if err != nil {
				return err
			}
			e.URLPath, err = url.PathUnescape(e.URLPath)
			if err != nil {
				return err
			}
		}
		err := s.EndpointManager.AddEndpoint(ctx, e)
		if err != nil {
			return fmt.Errorf("failed adding endpoint: %w", err)
		}
	}
	return nil
}

func (s *Server) AddMiddleware(middlewareFunc ...mux.MiddlewareFunc) {
	s.router.Use(middlewareFunc...)
}

func (s *Server) Start(ctx context.Context) error {
	eg, errCtx := errgroup.WithContext(ctx)
	if s.MetricsServer.Enabled {
		eg.Go(func() error {
			err := s.MetricsServer.StartServer(errCtx)
			if err != nil {
				return err
			}
			return nil
		})
		eg.Go(func() error {
			err := s.StartServer(errCtx)
			if err != nil {
				return err
			}
			return nil
		})
		return eg.Wait()
	}
	return s.StartServer(errCtx)
}
func (s *Server) ConfigureServer(ctx context.Context) *http.Server {
	s.server = &http.Server{
		Addr: ":" + s.ServingPort,

		Handler: s.router,
		BaseContext: func(_ net.Listener) context.Context {
			return ctxLogger.ConfigureCtx(ctxLogger.GetLogger(ctx), ctx)
		},
	}
	return s.server

}

func (s *Server) StartServer(ctx context.Context) error {
	var server *http.Server

	if s.server != nil {
		server = s.server
	} else {
		server = &http.Server{
			Addr:    ":" + s.ServingPort,
			Handler: s.router,
			BaseContext: func(_ net.Listener) context.Context {
				return ctxLogger.ConfigureCtx(ctxLogger.GetLogger(ctx), ctx)
			},
		}
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			ctxLogger.Error(ctx, "failed creating server", zap.Error(err))
		}
	}()
	ctxLogger.Info(ctx, "staring server", zap.String("address", server.Addr), zap.String("port", s.ServingPort), zap.String("prefix", s.PathPrefix))
	<-s.serverCtx.Done()
	ctxLogger.Info(ctx, "server shutting down")
	ctxShutDown, cancel := context.WithTimeout(context.Background(), s.shutdownDuration)
	defer func() {
		cancel()
	}()
	if err := server.Shutdown(ctxShutDown); err != nil {
		ctxLogger.Error(ctx, "server Shutdown Failed", zap.Error(err))
		return err
	}
	<-time.NewTicker(s.shutdownDuration).C

	select {
	case <-ctxShutDown.Done():
		s.shutdown()
	case <-s.requestTracker.Done(s.ctx):
		s.shutdown()
	}

	ctxLogger.Info(ctx, "server exited properly")
	return nil

}
