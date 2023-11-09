package endpoint_manager

import (
	"context"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/Seann-Moser/go-serve/server/handlers"
	"github.com/Seann-Moser/go-serve/server/metrics"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/server/endpoints"
)

type AddEndpoints func(manager Manager) error

type EndpointHandler interface {
	GetEndpoints() []*endpoints.Endpoint
}

type Manager struct {
	Router                  *mux.Router
	ExtraAddEndpointProcess func(ctx context.Context, endpoint *endpoints.Endpoint) error
}

func NewManager(router *mux.Router) *Manager {
	return &Manager{
		Router:                  router,
		ExtraAddEndpointProcess: nil,
	}
}
func (m *Manager) SetExtraFunc(v func(ctx context.Context, endpoint *endpoints.Endpoint) error) {
	m.ExtraAddEndpointProcess = v
}

func (m *Manager) AddRawEndpoints(ctx context.Context, endpoints ...*endpoints.Endpoint) error {
	for _, endpoint := range endpoints {
		err := m.AddEndpoint(ctx, endpoint)
		if err != nil {
			return fmt.Errorf("failed adding endpoint %s: %w", endpoint.URLPath, err)
		}
	}
	return nil
}

func (m *Manager) AddEndpoints(ctx context.Context, handlers []EndpointHandler) error {
	for _, h := range handlers {
		for _, endpoint := range h.GetEndpoints() {
			err := m.AddEndpoint(ctx, endpoint)
			if err != nil {
				return fmt.Errorf("failed adding endpoint %s: %w", endpoint.URLPath, err)
			}
		}
	}
	return nil
}

func (m *Manager) AddEndpoint(ctx context.Context, endpoint *endpoints.Endpoint) error {
	if endpoint.Methods == nil || len(endpoint.Methods) == 0 {
		endpoint.Methods = []string{http.MethodPost, http.MethodGet, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions}
	}
	if len(endpoint.Redirect) > 0 && endpoint.HandlerFunc == nil && endpoint.Handler == nil {
		ep, err := handlers.NewProxy(ctx, endpoint, "")
		if err != nil {
			return err
		}
		endpoint = ep
	}
	if endpoint.HandlerFunc != nil {
		ctxLogger.Debug(ctx, "adding handler func",
			zap.String("path", endpoint.URLPath),
			zap.Strings("methods", endpoint.Methods))
		m.Router.HandleFunc(endpoint.URLPath, endpoint.HandlerFunc).Methods(endpoint.Methods...)
	} else if endpoint.Handler != nil {
		ctxLogger.Debug(ctx, "adding handler",
			zap.String("path", endpoint.URLPath),
			zap.Strings("methods", endpoint.Methods))
		m.Router.Handle(endpoint.URLPath, endpoint.Handler).Methods(endpoint.Methods...)
	} else {
		ctxLogger.Error(ctx, "failed to add handler for", zap.String("path", endpoint.URLPath), zap.Strings("methods", endpoint.Methods))
	}
	if m.ExtraAddEndpointProcess == nil {
		return nil
	}
	return m.ExtraAddEndpointProcess(ctx, endpoint)
}

func (m *Manager) AddDefaultMetrics() {
	m.Router.Use(metrics.PrometheusTotalRequestsMiddleware)
	metrics.AddMetricsEndpoint(m.Router)
	metrics.RegisterDefaultMetrics()
}
