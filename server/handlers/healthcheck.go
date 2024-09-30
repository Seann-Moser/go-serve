package handlers

import (
	"context"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"net/http"
	"strings"
	"time"

	"github.com/Seann-Moser/go-serve/server/endpoints"
)

var HealthCheck = &endpoints.Endpoint{
	URLPath:         "/healthcheck",
	PermissionLevel: endpoints.All,
	HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	},
	Handler: nil,
}

var AdvancedHealthCheck = &endpoints.Endpoint{
	URLPath:         "/healthcheck",
	PermissionLevel: endpoints.All,
	HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	},
	Handler: nil,
}

type Ping func(ctx context.Context) bool

func NewAdvancedHealthCheck(timeout time.Duration, pings map[string]Ping) *endpoints.Endpoint {
	return &endpoints.Endpoint{
		URLPath:         "/healthcheck",
		PermissionLevel: endpoints.All,
		HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			var missing []string
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			for k, v := range pings {
				if !v(ctx) {
					missing = append(missing, fmt.Sprintf("failed to ping:%s", k))
				}
			}
			if len(missing) > 0 {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(strings.Join(missing, ", ")))
				ctxLogger.Error(r.Context(), strings.Join(missing, ", "))
				return
			}
			w.WriteHeader(http.StatusOK)
		},
		Handler: nil,
	}
}
