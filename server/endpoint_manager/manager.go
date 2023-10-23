package endpoint_manager

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Seann-Moser/go-serve/server/metrics"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/server/endpoints"
)

type AddEndpoints func(manager Manager) error

type EndpointHandler interface {
	GetEndpoints() []*endpoints.Endpoint
}

type Manager struct {
	ctx                     context.Context
	Router                  *mux.Router
	logger                  *zap.Logger
	ExtraAddEndpointProcess func(endpoint *endpoints.Endpoint) error
}

func NewManager(ctx context.Context, router *mux.Router, logger *zap.Logger) *Manager {
	return &Manager{
		ctx:                     ctx,
		Router:                  router,
		logger:                  logger,
		ExtraAddEndpointProcess: nil,
	}
}
func (m *Manager) SetExtraFunc(v func(endpoint *endpoints.Endpoint) error) {
	m.ExtraAddEndpointProcess = v
}

func (m *Manager) AddEndpoints(handlers []EndpointHandler) error {
	for _, h := range handlers {
		for _, endpoint := range h.GetEndpoints() {
			err := m.AddEndpoint(endpoint)
			if err != nil {
				return fmt.Errorf("failed adding endpoint %s: %w", endpoint.URLPath, err)
			}
		}
	}
	return nil
}

func (m *Manager) AddEndpoint(endpoint *endpoints.Endpoint) error {
	if endpoint.Methods == nil || len(endpoint.Methods) == 0 {
		endpoint.Methods = []string{http.MethodPost, http.MethodGet, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions}
	}
	if endpoint.HandlerFunc != nil {
		m.logger.Debug("adding handler func",
			zap.String("path", endpoint.URLPath),
			zap.Strings("methods", endpoint.Methods))
		m.Router.HandleFunc(endpoint.URLPath, endpoint.HandlerFunc).Methods(endpoint.Methods...)
	} else if endpoint.Handler != nil {
		m.logger.Debug("adding handler",
			zap.String("path", endpoint.URLPath),
			zap.Strings("methods", endpoint.Methods))
		m.Router.Handle(endpoint.URLPath, endpoint.Handler).Methods(endpoint.Methods...)
	} else {
		m.logger.Error("failed to add handler for", zap.String("path", endpoint.URLPath), zap.Strings("methods", endpoint.Methods))
	}
	if m.ExtraAddEndpointProcess == nil {
		return nil
	}
	return m.ExtraAddEndpointProcess(endpoint)
}

func (m *Manager) AddDefaultMetrics() {
	m.Router.Use(metrics.PrometheusTotalRequestsMiddleware)
	metrics.AddMetricsEndpoint(m.Router)
	metrics.RegisterDefaultMetrics()
}
