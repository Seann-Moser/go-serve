package endpoints

import (
	"net/http"
)

type Permission int

const (
	All Permission = iota
	SignedIn
	Admin
)

type EndpointHandler func(w http.ResponseWriter, r *http.Request)

type Endpoint struct {
	SubDomain       string          `json:"sub_domain" db:"sub_domain" q_config:"primary,where:="`
	Redirect        string          `json:"redirect" db:"redirect" q_config:"join,update"`
	URLPath         string          `json:"url_path" yaml:"url_path" db:"url_path" q_config:"primary,data_type:varchar(512),delete,join,update,where:="`
	PermissionLevel Permission      `json:"permission_level" yaml:"permission_level" db:"permission_level" q_config:"join,update,where:<="`
	Methods         []string        `json:"methods" yaml:"methods" db:"-"`
	HandlerFunc     EndpointHandler `db:"-" json:"-"`
	Handler         http.Handler    `db:"-" json:"-"`
}
