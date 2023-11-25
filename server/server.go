package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/Seann-Moser/go-serve/server/metrics"
	"golang.org/x/sync/errgroup"
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

type Server struct {
	ServingPort     string `yaml:"serving_port" json:"serving_port" env:"SERVING_PORT"`
	ctx             context.Context
	router          *mux.Router
	EndpointManager *endpoint_manager.Manager
	Response        *response.Response
	Request         *request.Request
	PathPrefix      string

	MetricsServer *metrics.Metrics
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
	fs.AddFlagSet(metrics.MetricFlags())
	return fs
}

func New(ctx context.Context) *Server {
	return NewServer(ctx,
		viper.GetString(serverPortFlag),
		viper.GetString(serverPrefixFlag),
		viper.GetInt64(serverMaxReceivedBytesFlag),
		viper.GetBool(serverShowErrFlag))
}

func NewServer(ctx context.Context, servingPort string, pathPrefix string, mb int64, showErr bool) *Server {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	notifyContext, cancel := context.WithCancel(ctx)
	go func() {
		osCall := <-c
		ctxLogger.Info(ctx, fmt.Sprintf("system call:%+v", osCall))
		cancel()
	}()

	router := mux.NewRouter()
	if !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}
	m := metrics.New(map[string]string{"ver": VERSION})
	if VERSION != "dev" {
		m.Version = VERSION
	}

	return &Server{
		ServingPort:     servingPort,
		ctx:             notifyContext,
		router:          router,
		EndpointManager: endpoint_manager.NewManager(router),
		Response:        response.NewResponse(showErr),
		Request:         request.NewRequest(mb),
		PathPrefix:      pathPrefix,
		MetricsServer:   m,
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

func (s *Server) Start() error {
	eg, errCtx := errgroup.WithContext(s.GetContext())
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

func (s *Server) StartServer(ctx context.Context) error {
	server := &http.Server{
		Addr:    ":" + s.ServingPort,
		Handler: s.router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			ctxLogger.Error(ctx, "failed creating server", zap.Error(err))
		}
	}()

	ctxLogger.Info(ctx, "staring server", zap.String("address", server.Addr), zap.String("port", s.ServingPort), zap.String("prefix", s.PathPrefix))
	<-ctx.Done()
	ctxLogger.Info(ctx, "server stopped")
	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctxShutDown); err != nil {
		ctxLogger.Error(ctx, "server Shutdown Failed", zap.Error(err))
		return err
	}
	ctxLogger.Info(ctx, "server exited properly")
	return nil
}
