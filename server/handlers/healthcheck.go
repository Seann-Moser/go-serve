package handlers

import (
	"context"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"net/http"
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
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			eg, ctx := errgroup.WithContext(ctx)
			for k, v := range pings {
				eg.Go(func() error {
					if !v(ctx) {
						return fmt.Errorf("failed to ping:%s", k)
					}
					return nil
				})
				//if !v(ctx) {
				//	missing = append(missing, fmt.Sprintf("failed to ping:%s", k))
				//}
			}

			if err := eg.Wait(); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(err.Error()))
				ctxLogger.Error(r.Context(), "failed to ping all dependencies", zap.Error(err))
				return
			}
			w.WriteHeader(http.StatusOK)
		},
		Handler: nil,
	}
}

// RobotTxt defines the robots.txt endpoint
var RobotTxt = &endpoints.Endpoint{
	URLPath:         "/robots.txt", // Corrected the path to "robots.txt"
	PermissionLevel: endpoints.All,
	HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Set content type to text/plain
		w.Header().Set("Content-Type", "text/plain")

		// Serve the robots.txt content
		// This example disallows all robots from all paths
		robotsTxtContent := `
User-agent: *
Disallow: /
`
		_, _ = w.Write([]byte(robotsTxtContent))
	},
	Handler: nil,
}
