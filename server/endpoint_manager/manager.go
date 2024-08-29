package endpoint_manager

import (
	"context"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/Seann-Moser/go-serve/server/handlers"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/server/endpoints"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
	if endpoint == nil {
		return nil
	}
	if len(endpoint.Methods) == 0 {
		endpoint.Methods = []string{http.MethodPost, http.MethodGet, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions}
	}
	var hasOption bool
	for _, m := range endpoint.Methods {
		if m == http.MethodOptions {
			hasOption = true
		}
	}
	if !hasOption {
		endpoint.Methods = append(endpoint.Methods, http.MethodOptions)
	}
	handleFunc := func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) *mux.Route {
		// Configure the "http.route" for the HTTP instrumentation.
		handler := otelhttp.WithRouteTag(pattern, otelhttp.NewHandler(http.HandlerFunc(handlerFunc), pattern))
		return m.Router.Handle(pattern, handler)
	}
	handle := func(pattern string, handlerFunc http.Handler) *mux.Route {
		// Configure the "http.route" for the HTTP instrumentation.\
		handler := otelhttp.WithRouteTag(pattern, otelhttp.NewHandler(handlerFunc, pattern))
		return m.Router.Handle(pattern, handler)
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
		handleFunc(endpoint.URLPath, endpoint.HandlerFunc).Methods(endpoint.Methods...)
	} else if endpoint.Handler != nil {
		ctxLogger.Debug(ctx, "adding handler",
			zap.String("path", endpoint.URLPath),
			zap.Strings("methods", endpoint.Methods))
		handle(endpoint.URLPath, endpoint.Handler).Methods(endpoint.Methods...)
	} else {
		ctxLogger.Error(ctx, "failed to add handler for", zap.String("path", endpoint.URLPath), zap.Strings("methods", endpoint.Methods))
	}
	if m.ExtraAddEndpointProcess == nil {
		return nil
	}
	return m.ExtraAddEndpointProcess(ctx, endpoint)
}
