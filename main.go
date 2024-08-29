package main

import (
	"context"
	"encoding/json"
	"github.com/Seann-Moser/go-serve/server/middle"
	"log"
	"net/http"

	"github.com/Seann-Moser/go-serve/server"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	"github.com/Seann-Moser/go-serve/server/handlers"
)

// @title 
// @version 
// @description 

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @schemes http https
// @host 
// @BasePath /
// @query.collection.format multi


func main() {
	s := server.NewServer(context.Background(), "8888", "/test", 0, false)
	if err := s.AddEndpoints(context.Background(), handlers.HealthCheck); err != nil {
		log.Fatal(err)
	}
	if err := s.AddEndpoints(context.Background(),
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
	if err := s.AddEndpoints(context.Background(),
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
	h, _ := handlers.NewProxy(context.Background(), &endpoints.Endpoint{
		SubDomain: "proxy",
		Redirect:  "https://www.google.com",
		URLPath:   "search/search",
	}, "/test/search/search")
	_ = s.AddEndpoints(context.Background(), h)
	if err := s.StartServer(context.Background()); err != nil {
		log.Fatal(err)
	}
	cors, _ := middle.NewCorsMiddleware([]string{}, []string{}, []string{}, false)
	if err := s.StartServer(context.Background()); err != nil {
		log.Fatal(err)
	}
	s.AddMiddleware(cors.Cors)
	//s.AddMiddleware(middle.NewMetrics(true, logger).Middleware, middle.NewCorsMiddleware().Cors)
	if err := s.StartServer(context.Background()); err != nil {
		log.Fatal(err)
	}
}