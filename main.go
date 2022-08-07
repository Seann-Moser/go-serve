package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/server"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	"github.com/Seann-Moser/go-serve/server/handlers"
	"github.com/Seann-Moser/go-serve/server/middleware"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	s := server.NewServer(context.Background(), "8888", "mnlib.com", logger)
	if err := s.AddEndpoints(handlers.HealthCheck); err != nil {
		log.Fatal(err)
	}
	if err := s.AddEndpoints(
		&endpoints.Endpoint{
			SubDomain: "books",
			URL: &url.URL{
				Path: "/{path}",
			},
			Methods:         []string{http.MethodGet, http.MethodPost},
			PermissionLevel: 0,
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]string{
					"message": "mnlib endpoint",
				})
			},
			Handler: nil,
		}); err != nil {
		log.Fatal(err)
	}
	if err := s.AddEndpoints(
		&endpoints.Endpoint{
			SubDomain: "auth",
			URL: &url.URL{
				Path: "/{path}",
			},
			Methods:         []string{http.MethodGet, http.MethodPost},
			PermissionLevel: 0,
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]string{
					"message": "auth endpoint",
				})
			},
			Handler: nil,
		}); err != nil {
		log.Fatal(err)
	}
	h, err := handlers.NewProxy("proxy", "https://www.google.com", logger)
	s.AddEndpoints(h)

	s.AddMiddleware(middleware.NewMetrics(true, logger).Middleware, middleware.NewCorsMiddleware().Cors)
	if err := s.StartServer(); err != nil {
		log.Fatal(err)
	}
}
