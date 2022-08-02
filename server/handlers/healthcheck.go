package handlers

import (
	"net/http"
	"net/url"

	"github.com/Seann-Moser/go-serve/server/endpoints"
)

var HealthCheck = endpoints.Endpoint{
	SubdomainProxy: "",
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
