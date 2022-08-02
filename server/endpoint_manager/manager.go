package endpoint_manager

import (
	"context"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/server/endpoints"
)

type AddEndpoints func(manager Manager) error

type Manager struct {
	ctx                     context.Context
	Router                  *mux.Router
	logger                  *zap.Logger
	extraAddEndpointProcess func(endpoint *endpoints.Endpoint) error
}

func NewManager(ctx context.Context, router *mux.Router, logger *zap.Logger) *Manager {
	return &Manager{
		ctx:                     ctx,
		Router:                  router,
		logger:                  logger,
		extraAddEndpointProcess: nil,
	}
}
func (m *Manager) SetExtraFunc(v func(endpoint *endpoints.Endpoint) error) {
	m.extraAddEndpointProcess = v
}

func (m *Manager) AddEndpoint(endpoint *endpoints.Endpoint) error {
	if endpoint.HandlerFunc != nil {
		m.logger.Debug("adding handler func",
			zap.String("path", endpoint.URL.Path),
			zap.Strings("methods", endpoint.Methods))
		m.Router.HandleFunc(endpoint.URL.Path, endpoint.HandlerFunc).Methods(endpoint.Methods...)
	} else if endpoint.Handler != nil {
		m.logger.Debug("adding handler",
			zap.String("path", endpoint.URL.Path),
			zap.Strings("methods", endpoint.Methods))
		m.Router.Handle(endpoint.URL.Path, endpoint.Handler).Methods(endpoint.Methods...)
	}
	if m.extraAddEndpointProcess == nil {
		return nil
	}
	return m.extraAddEndpointProcess(endpoint)
}
