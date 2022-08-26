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

func NewProxy(ep *endpoints.Endpoint, logger *zap.Logger) (*endpoints.Endpoint, error) {
	respManger := response.NewResponse(false, logger)
	redirectURL, err := url.Parse(ep.Redirect)
	if err != nil {
		return nil, err
	}
	logger.Info("redirect url", zap.String("url", ep.Redirect), zap.String("subdomain", ep.SubDomain), zap.Strings("methods", ep.Methods), zap.String("path", ep.URLPath))
	return &endpoints.Endpoint{
		SubDomain:       ep.SubDomain,
		URLPath:         ep.URLPath,
		Methods:         ep.Methods,
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
			logger.Info("redirecting to proxy endpoint", zap.String("endpoint", u.String()), zap.String("path", r.URL.Path))
			req, err := http.NewRequestWithContext(ctx, r.Method, u.String(), r.Body)
			if err != nil {
				respManger.Error(w, err, http.StatusInternalServerError, "failed creating proxy request")
				return
			}
			for _, c := range r.Cookies() {
				req.AddCookie(c)
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
