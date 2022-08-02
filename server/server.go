package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/server/endpoint_manager"
)

type Server struct {
	Endpoint        string `yaml:"endpoint" json:"endpoint" env:"ENDPOINT"`
	ServingPort     string `yaml:"serving_port" json:"serving_port" env:"SERVING_PORT"`
	ctx             context.Context
	router          *mux.Router
	logger          *zap.Logger
	EndpointManager *endpoint_manager.Manager
}

func NewServer(ctx context.Context, servingPort string, logger *zap.Logger) *Server {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	notifyContext, cancel := context.WithCancel(ctx)
	go func() {
		osCall := <-c
		logger.Info(fmt.Sprintf("system call:%+v", osCall))
		cancel()
	}()
	router := mux.NewRouter()
	return &Server{
		Endpoint:        "",
		ServingPort:     servingPort,
		ctx:             notifyContext,
		router:          router,
		logger:          logger,
		EndpointManager: endpoint_manager.NewManager(ctx, router, logger),
	}
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
