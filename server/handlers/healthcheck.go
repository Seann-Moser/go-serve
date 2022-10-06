package handlers

import (
	"net/http"

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
