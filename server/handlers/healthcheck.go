package handlers

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/pkg/response"
	"github.com/Seann-Moser/go-serve/server/endpoints"
)

var HealthCheck = &endpoints.Endpoint{
	URL: &url.URL{
		Path: "/health_check",
	},
	Methods:         []string{http.MethodGet, http.MethodPost},
	PermissionLevel: endpoints.All,
	HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	},
	Handler: nil,
}

func NewProxy(subdomain, redirect string, logger *zap.Logger) (*endpoints.Endpoint, error) {
	respManger := response.NewResponse(logger)
	redirectURL, err := url.Parse(redirect)
	if err != nil {
		return nil, err
	}
	logger.Info("redirect url", zap.String("url", redirect))
	return &endpoints.Endpoint{
		SubDomain: subdomain,
		URL: &url.URL{
			Path: "/{path}",
		},
		Methods:         []string{http.MethodGet, http.MethodPost},
		PermissionLevel: endpoints.All,
		HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer func() {
				cancel()
			}()
			u := url.URL{
				Opaque:      "",
				Scheme:      redirectURL.Scheme,
				User:        r.URL.User,
				Host:        redirectURL.Host,
				Path:        r.URL.Path,
				RawPath:     "",
				ForceQuery:  false,
				RawQuery:    r.URL.RawQuery,
				Fragment:    r.URL.Fragment,
				RawFragment: r.URL.RawFragment,
			}
			logger.Info("redirecting to proxy endpoint", zap.String("endpoint", u.String()))
			req, err := http.NewRequestWithContext(ctx, r.Method, u.String(), r.Body)
			if err != nil {
				respManger.Error(w, err, http.StatusInternalServerError, "failed creating proxy request")
				return
			}
			req.Header = r.Header
			resp, err := (&http.Client{}).Do(req)
			if err != nil {
				respManger.Error(w, err, http.StatusInternalServerError, "failed sending proxy request")
				return
			}
			respManger.Raw(w, resp)
		},
	}, nil

}
