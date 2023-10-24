package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Seann-Moser/go-serve/server/middle"

	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/server"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	"github.com/Seann-Moser/go-serve/server/handlers"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	s := server.NewServer(context.Background(), "8888", "/test", 0, false, logger)
	if err := s.AddEndpoints(handlers.HealthCheck); err != nil {
		log.Fatal(err)
	}
	if err := s.AddEndpoints(
		&endpoints.Endpoint{
			SubDomain:       "books",
			URLPath:         "/{path}",
			Methods:         []string{http.MethodGet, http.MethodPost},
			PermissionLevel: 0,
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]string{
					"message": "mnlib endpoint",
				})
			},
			Handler: nil,
		}); err != nil {
		log.Fatal(err)
	}
	if err := s.AddEndpoints(
		&endpoints.Endpoint{
			SubDomain:       "auth",
			URLPath:         "/{path}",
			Methods:         []string{http.MethodGet, http.MethodPost},
			PermissionLevel: 0,
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]string{
					"message": "auth endpoint",
				})
			},
			Handler: nil,
		}); err != nil {
		log.Fatal(err)
	}
	h, _ := handlers.NewProxy(&endpoints.Endpoint{
		SubDomain: "proxy",
		Redirect:  "https://www.google.com",
		URLPath:   "search/search",
	}, 10*time.Second, "/test/search/search", logger)
	_ = s.AddEndpoints(h)
	if err := s.StartServer(); err != nil {
		logger.Fatal("failed creating cors", zap.Error(err))
	}
	cors, _ := middle.NewCorsMiddleware([]string{}, []string{}, []string{}, false, logger)
	if err := s.StartServer(); err != nil {
		logger.Fatal("failed creating cors", zap.Error(err))
	}
	s.AddMiddleware(cors.Cors)
	//s.AddMiddleware(middle.NewMetrics(true, logger).Middleware, middle.NewCorsMiddleware().Cors)
	if err := s.StartServer(); err != nil {
		log.Fatal(err)
	}
}
