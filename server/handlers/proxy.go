package handlers

import (
	"context"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/pkg/response"
	"github.com/Seann-Moser/go-serve/server/endpoints"
)

type proxy struct {
	redirectURL *url.URL
	logger      *zap.Logger
	respManager *response.Response
	timeout     time.Duration
}

func NewProxyHandler(ep *endpoints.Endpoint, logger *zap.Logger) (*proxy, error) {
	respManger := response.NewResponse(false)
	redirectURL, err := url.Parse(ep.Redirect)
	if err != nil {
		return nil, err
	}
	return &proxy{
		redirectURL: redirectURL,
		logger:      logger,
		respManager: respManger,
		timeout:     time.Duration(ep.Timeout) * time.Millisecond,
	}, nil
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var ctx context.Context
	var cancel context.CancelFunc
	if p.timeout > 0 {
		ctx, cancel = context.WithTimeout(r.Context(), p.timeout)
	} else {
		cancel = func() {}
		ctx = r.Context()
	}
	defer func() {
		cancel()
	}()
	u := url.URL{
		Opaque:      "",
		Scheme:      p.redirectURL.Scheme,
		User:        r.URL.User,
		Host:        p.redirectURL.Host,
		Path:        r.URL.Path,
		RawPath:     "",
		ForceQuery:  false,
		RawQuery:    r.URL.RawQuery,
		Fragment:    r.URL.Fragment,
		RawFragment: r.URL.RawFragment,
	}
	p.logger.Info("redirecting to proxy endpoint", zap.String("endpoint", u.String()), zap.String("path", r.URL.Path))
	req, err := http.NewRequestWithContext(ctx, r.Method, u.String(), r.Body)
	if err != nil {
		p.respManager.Error(r, w, err, http.StatusInternalServerError, "failed creating proxy request")
		return
	}
	for _, c := range r.Cookies() {
		req.AddCookie(c)
	}
	req.Header = r.Header
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		p.respManager.Error(r, w, err, http.StatusInternalServerError, "failed sending proxy request")
		return
	}
	p.respManager.Raw(r, w, resp)
}

func NewProxy(ctx context.Context, ep *endpoints.Endpoint, trimPath string) (*endpoints.Endpoint, error) {
	respManger := response.NewResponse(false)
	redirectURL, err := url.Parse(ep.Redirect)
	if err != nil {
		return nil, err
	}
	ctxLogger.Debug(ctx, "redirect url", zap.String("url", ep.Redirect), zap.String("subdomain", ep.SubDomain), zap.Strings("methods", ep.Methods), zap.String("path", ep.URLPath))

	return &endpoints.Endpoint{
		SubDomain:       ep.SubDomain,
		URLPath:         ep.URLPath,
		Methods:         ep.Methods,
		PermissionLevel: endpoints.All,

		HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			var ctx context.Context
			var cancel context.CancelFunc
			if ep.Timeout > 0 {
				ctx, cancel = context.WithTimeout(r.Context(), time.Duration(ep.Timeout)*time.Millisecond)
			} else {
				cancel = func() {}
				ctx = r.Context()
			}

			defer func() {
				cancel()
			}()
			u := url.URL{
				Opaque:      "",
				Scheme:      redirectURL.Scheme,
				User:        r.URL.User,
				Host:        redirectURL.Host,
				Path:        strings.TrimPrefix(r.URL.Path, trimPath),
				RawPath:     "",
				ForceQuery:  false,
				RawQuery:    r.URL.RawQuery,
				Fragment:    r.URL.Fragment,
				RawFragment: r.URL.RawFragment,
			}
			ctxLogger.Info(ctx, "redirecting to proxy endpoint", zap.String("endpoint", u.String()), zap.String("path", r.URL.Path))
			req, err := http.NewRequestWithContext(ctx, r.Method, u.String(), r.Body)
			if err != nil {
				respManger.Error(r, w, err, http.StatusInternalServerError, "failed creating proxy request")
				return
			}
			for _, c := range r.Cookies() {
				req.AddCookie(c)
			}
			req.Header = r.Header
			resp, err := (&http.Client{}).Do(req)
			if err != nil {
				respManger.Error(r, w, err, http.StatusInternalServerError, "failed sending proxy request")
				return
			}
			ctxLogger.Debug(ctx, "finished redirecting data", zap.Int("status_code", resp.StatusCode))
			respManger.Raw(r, w, resp)
		},
	}, nil

}
