package server

import (
	"context"
	"errors"
	"fmt"
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

type Server struct {
	ServingPort     string `yaml:"serving_port" json:"serving_port" env:"SERVING_PORT"`
	ctx             context.Context
	router          *mux.Router
	logger          *zap.Logger
	EndpointManager *endpoint_manager.Manager
	Response        *response.Response
	Request         *request.Request
	PathPrefix      string
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
	return fs
}

func New(ctx context.Context, logger *zap.Logger) *Server {
	return NewServer(ctx,
		viper.GetString(serverPortFlag),
		viper.GetString(serverPrefixFlag),
		viper.GetInt64(serverMaxReceivedBytesFlag),
		viper.GetBool(serverShowErrFlag), logger)
}

func NewServer(ctx context.Context, servingPort string, pathPrefix string, mb int64, showErr bool, logger *zap.Logger) *Server {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	notifyContext, cancel := context.WithCancel(ctx)
	go func() {
		osCall := <-c
		logger.Info(fmt.Sprintf("system call:%+v", osCall))
		cancel()
	}()

	router := mux.NewRouter()
	if !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}

	return &Server{
		ServingPort:     servingPort,
		ctx:             notifyContext,
		router:          router,
		logger:          logger,
		EndpointManager: endpoint_manager.NewManager(router),
		Response:        response.NewResponse(showErr),
		Request:         request.NewRequest(mb),
		PathPrefix:      pathPrefix,
	}
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

type NotFound struct {
	logger *zap.Logger
	resp   *response.Response
}

func NewNF() *NotFound {
	return &NotFound{
		resp: response.NewResponse(false),
	}
}

func (n *NotFound) ServeHTTP(writer http.ResponseWriter, r *http.Request) {
	n.resp.Error(r.Context(), writer, nil, http.StatusNotFound, fmt.Sprintf(
		"%d: path not found: %s", http.StatusNotFound, r.URL.String()))
	n.logger.Error("failed to find routing path", zap.String("url", r.URL.String()))
}

var _ http.Handler = &NotFound{}

func (s *Server) NotFoundHandler(nf http.Handler) {
	s.router.NotFoundHandler = nf
}

func (s *Server) StartServer() error {
	server := &http.Server{
		Addr:    ":" + s.ServingPort,
		Handler: s.router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("failed creating server", zap.Error(err))
		}
	}()

	s.logger.Info("staring server", zap.String("address", server.Addr), zap.String("port", s.ServingPort), zap.String("prefix", s.PathPrefix))
	<-s.ctx.Done()
	s.logger.Info("server stopped")
	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctxShutDown); err != nil {
		s.logger.Error("server Shutdown Failed", zap.Error(err))
		return err
	}
	s.logger.Info("server exited properly")
	return nil
}
