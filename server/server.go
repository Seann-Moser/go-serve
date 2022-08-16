package server

import (
	"context"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/request"
	"github.com/Seann-Moser/go-serve/pkg/response"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/server/endpoint_manager"
	"github.com/Seann-Moser/go-serve/server/endpoints"
)

type Server struct {
	Endpoint        string `yaml:"endpoint" json:"endpoint" env:"ENDPOINT"`
	ServingPort     string `yaml:"serving_port" json:"serving_port" env:"SERVING_PORT"`
	ctx             context.Context
	router          *mux.Router
	logger          *zap.Logger
	EndpointManager *endpoint_manager.Manager
	host            string
	subrouters      map[string]*endpoint_manager.Manager
	Response        *response.Response
	Request         *request.Request
}

func NewServer(ctx context.Context, servingPort string, host string, mb int64, showErr bool, logger *zap.Logger) *Server {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	notifyContext, cancel := context.WithCancel(ctx)
	go func() {
		osCall := <-c
		logger.Info(fmt.Sprintf("system call:%+v", osCall))
		cancel()
	}()
	router := mux.NewRouter()
	if host != "" {
		router = router.Host(fmt.Sprintf("{subdomain:[a-z]+}.%s", host)).Subrouter()
	}
	return &Server{
		Endpoint:        "",
		host:            host,
		ServingPort:     servingPort,
		ctx:             notifyContext,
		router:          router,
		logger:          logger,
		EndpointManager: endpoint_manager.NewManager(ctx, router, logger),
		subrouters:      map[string]*endpoint_manager.Manager{},
		Response:        response.NewResponse(showErr, logger),
		Request:         request.NewRequest(mb, logger),
	}
}

func (s *Server) AddEndpoints(endpoint ...*endpoints.Endpoint) error {
	for _, e := range endpoint {
		if len(e.SubDomain) > 0 && len(s.host) > 0 {
			sub := fmt.Sprintf("%s.%s", e.SubDomain, s.host)
			if s.host == "" {
				sub = e.SubDomain
			}
			subRouter, found := s.subrouters[sub]
			if !found {
				sr := s.router.Host(sub).Subrouter()
				subRouter = endpoint_manager.NewManager(s.ctx, sr, s.logger)
				subRouter.SetExtraFunc(s.EndpointManager.ExtraAddEndpointProcess)
				s.subrouters[e.SubDomain] = subRouter
			}
			err := subRouter.AddEndpoint(e)
			if err != nil {
				return fmt.Errorf("failed adding endpoint: %w", err)
			}
		} else {
			err := s.EndpointManager.AddEndpoint(e)
			if err != nil {
				return fmt.Errorf("failed adding endpoint: %w", err)
			}
		}
	}
	return nil
}

func (s *Server) AddMiddleware(middlewareFunc ...mux.MiddlewareFunc) {
	s.router.Use(middlewareFunc...)
}
func (s *Server) AddMiddlewareWithSubdomain(subdomain string, middlewareFunc ...mux.MiddlewareFunc) error {
	if len(subdomain) == 0 {
		return fmt.Errorf("no subdomain provided")
	}
	subRouter, found := s.subrouters[subdomain]
	if !found {
		return fmt.Errorf("subdomain does not exist")
	}
	subRouter.Router.Use(middlewareFunc...)
	return nil
}

func (s *Server) StartServer() error {
	server := &http.Server{
		Addr:    ":" + s.ServingPort,
		Handler: s.router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("failed creating server", zap.Error(err))
		}
	}()
	s.logger.Info(fmt.Sprintf("server started on port: %s", s.ServingPort))
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
