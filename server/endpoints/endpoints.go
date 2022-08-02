package endpoints

import (
	"net/http"
	"net/url"
)

type Permission int

const (
	All Permission = iota
	SignedIn
	Admin
)

type EndpointHandler func(w http.ResponseWriter, r *http.Request)

type Endpoint struct {
	SubDomain       string
	URL             *url.URL   `json:"url" yaml:"url"`
	Methods         []string   `json:"methods" yaml:"methods"`
	PermissionLevel Permission `json:"permission_level" yaml:"permission_level"`
	HandlerFunc     EndpointHandler
	Handler         http.Handler
}
