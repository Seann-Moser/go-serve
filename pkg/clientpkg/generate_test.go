package clientpkg

import (
	"net/http"
	"testing"

	"github.com/Seann-Moser/go-serve/server/endpoints"
)

func TestGenerate(t *testing.T) {
	_, err := GenerateBaseClient(true, []string{"Api-Key"}, GetEndpoints()...)
	if err != nil {
		t.Error(err)
	}
}
func TestGenerateComments(t *testing.T) {
	GenerateComments(&ApiDoc{}, GetEndpoints()...)
}

func GetEndpoints() []*endpoints.Endpoint {
	e := []*endpoints.Endpoint{
		{
			SubDomain:   "test",
			URLPath:     "/account/{account_id}/user/{user_id}",
			Methods:     []string{http.MethodPost},
			HandlerFunc: HandlerFunc,
			ResponseTypeMap: map[string]interface{}{
				"POST": RequestData{},
			},
		},
		{
			SubDomain:   "test",
			URLPath:     "/account/{account_id}/user/{user_id}",
			Methods:     []string{http.MethodGet},
			HandlerFunc: HandlerFunc,
		},
		{
			SubDomain:   "test",
			URLPath:     "/account/{account_id}/user/{user_id}/settings",
			Methods:     []string{http.MethodGet},
			Headers:     []string{"header", "test"},
			HandlerFunc: HandlerFunc,
		},
		{
			SubDomain:   "test",
			URLPath:     "/account/{account_id}/user/{user_id}/settings/query",
			Methods:     []string{http.MethodGet},
			QueryParams: []string{"q", "query", "token_id"},
			Async:       true,
		},
	}
	e[1] = e[1].SetResponseType(RequestData{}, http.MethodGet)
	e[2] = e[2].SetRequestType([]ResponseData{}, http.MethodGet)
	e[3] = e[3].SetRequestType(map[string]string{}, http.MethodGet)
	return e
}
