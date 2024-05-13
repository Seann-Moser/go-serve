package endpoints

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Seann-Moser/QueryHelper"
	"github.com/gorilla/mux"
)

type Permission int

const (
	All Permission = iota
	SignedIn
	Admin
	SuperAdmin = Permission(int(^uint(0) >> 1))
)

type Endpoint struct {
	SubDomain       string           `json:"sub_domain" db:"sub_domain" qc:"primary;where::="`
	Redirect        string           `json:"redirect" db:"redirect" qc:"join;update"`
	URLPath         string           `json:"url_path" yaml:"url_path" db:"url_path" qc:"primary;data_type::varchar(512);delete;join;update;where::="`
	PermissionLevel Permission       `json:"permission_level" yaml:"permission_level" db:"permission_level" qc:"join;update;where::<="`
	Role            string           `json:"role" db:"role" qc:"primary;join;update;default::default"`
	Roles           []string         `json:"roles" db:"-"`
	Method          string           `json:"-" db:"method" qc:"primary;update"`
	Methods         []string         `json:"methods" yaml:"methods" db:"-"`
	HandlerFunc     http.HandlerFunc `db:"-" json:"-"`
	Handler         http.Handler     `db:"-" json:"-"`
	Timeout         int              `json:"timeout" db:"timeout" qc:"update;default::10"`

	Description          string                     `json:"description" db:"-"`
	ParamDescriptions    map[string]string          `json:"param_descriptions" db:"-"`
	ResponseDescriptions map[string]HTTPDescription `json:"response_descriptions" db:"-"`

	ResponseFailures map[string]HTTPDescription `json:"response_failures" db:"-"`

	Async           bool                   `json:"-" db:"-"`
	RequestTypeMap  map[string]interface{} `json:"-" db:"-"`
	ResponseTypeMap map[string]interface{} `json:"-" db:"-"`
	Headers         []string               `json:"-" db:"-"`
	QueryParams     []string               `json:"-" db:"-"`
	SkipGenerate    bool                   `json:"-" db:"-"`
}
type HTTPDescription struct {
	Description string
	StatusCode  int
}

func (e *Endpoint) SetResponseType(i interface{}, methods ...string) *Endpoint {
	if e.ResponseTypeMap == nil {
		e.ResponseTypeMap = map[string]interface{}{}
	}
	if len(methods) == 0 {
		for _, m := range e.Methods {
			e.ResponseTypeMap[strings.ToUpper(m)] = i
		}
	} else {
		for _, m := range e.Methods {
			e.ResponseTypeMap[strings.ToUpper(m)] = i
		}
	}
	return e
}

func (e *Endpoint) SetRequestType(i interface{}, method string) *Endpoint {
	if e.RequestTypeMap == nil {
		e.RequestTypeMap = map[string]interface{}{}
	}
	if method == "" {
		for _, m := range e.Methods {
			e.RequestTypeMap[m] = i
		}
	} else {
		e.RequestTypeMap[strings.ToUpper(method)] = i
	}
	return e
}

func NewEndpoint(prefix string, urlPath string, role string, HandlerFunc http.HandlerFunc, methods ...string) *Endpoint {
	path, _ := url.JoinPath(prefix, urlPath)
	return &Endpoint{
		SubDomain:       "",
		Redirect:        "",
		URLPath:         path,
		PermissionLevel: 0,
		Role:            role,
		Method:          strings.Join(methods, ","),
		Methods:         methods,
		HandlerFunc:     HandlerFunc,
		Handler:         nil,
		Timeout:         0,
	}
}

func LoadEndpoints(ctx context.Context, defaultEndpoints ...*Endpoint) ([]*Endpoint, error) {
	var endpoints []*Endpoint
	var err error
	endpointTable, err := QueryHelper.GetTableCtx[Endpoint](ctx)
	if err == nil {
		endpoints, err = QueryHelper.QueryTable[Endpoint](endpointTable).Run(ctx, nil)
		if err != nil {
			return nil, err
		}
	}
	duplicateMap := map[string]bool{}
	var output []*Endpoint
	for _, e := range endpoints {
		key := e.UniqueID()
		if _, found := duplicateMap[key]; !found {
			output = append(output, e)
		}
	}
	for _, e := range defaultEndpoints {
		key := e.UniqueID()
		if _, found := duplicateMap[key]; !found {
			_ = e.Save(ctx)
			output = append(output, e)
		}
	}
	return output, nil
}

func (e *Endpoint) UniqueID() string {
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("%s-%s-%s-%s", e.URLPath, e.Role, e.Method, e.SubDomain)))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)

}
func (e *Endpoint) Save(ctx context.Context) error {
	endpointTable, err := QueryHelper.GetTableCtx[Endpoint](ctx)
	if err != nil {
		return err
	}

	_, err = endpointTable.Insert(ctx, nil, *e)
	if err != nil {
		return err
	}
	return nil
}

func (e *Endpoint) Match(r *http.Request) bool {
	rawPath := r.URL.Path
	muxVars := mux.Vars(r)
	for k, v := range muxVars {
		rawPath = strings.ReplaceAll(rawPath, v, fmt.Sprintf("{%s}", k))
	}
	return strings.EqualFold(e.URLPath, rawPath)
}

func (e *Endpoint) SetMethods(methods ...string) {
	e.Methods = methods
	e.Method = strings.Join(methods, ",")
}

func (e *Endpoint) GetMethods() []string {
	if len(e.Methods) == 0 && len(e.Method) > 0 {
		e.Methods = strings.Split(strings.ToUpper(e.Method), ",")
	} else if len(e.Methods) > 0 {
		e.Method = strings.Join(e.Methods, ",")
	}
	return e.Methods
}
