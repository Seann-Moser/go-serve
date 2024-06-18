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
	_, err = GenerateBaseJSClient(true, []string{"Api-Key"}, GetEndpoints()...)
	if err != nil {
		t.Error(err)
	}

}
func TestGenerateComments(t *testing.T) {
	GenerateComments(&ApiDoc{}, map[string]string{}, GetEndpoints()...)
}

func GetEndpoints() []*endpoints.Endpoint {
	c := Client{}
	e := []*endpoints.Endpoint{
		{
			SubDomain:   "test",
			URLPath:     "/account/{account_id}/user/{user_id}",
			Methods:     []string{http.MethodPost},
			HandlerFunc: c.HandlerFuncs,
			ResponseTypeMap: map[string]interface{}{
				"POST": RequestData{},
			},
		},
		{
			SubDomain:   "test",
			URLPath:     "/account/{account_id}/user/{user_id}",
			Methods:     []string{http.MethodGet},
			HandlerFunc: HandlerFuncs,
		},
		{
			SubDomain:   "test",
			URLPath:     "/account/{account_id}/user/{user_id}/settings",
			Methods:     []string{http.MethodGet, http.MethodDelete},
			Headers:     []string{"header", "test"},
			HandlerFunc: c.HandlerFuncs,
		},
		{
			SubDomain:   "test",
			URLPath:     "/account/{account_id}/user/{user_id}/settings/query",
			Methods:     []string{http.MethodGet},
			QueryParams: []string{"q", "query", "token_id"},
			Async:       true,
		},
		{
			SubDomain:   "test",
			URLPath:     "/account/{account_id}/user/{user_id}/settings/query/2",
			Methods:     []string{http.MethodGet},
			QueryParams: []string{"q", "query", "token_id"},
			Async:       true,
		},
	}
	e[1] = e[1].SetResponseType(RequestData{}, http.MethodGet)
	e[2] = e[2].SetRequestType([]ResponseData{}, http.MethodGet)
	e[3] = e[3].SetRequestType(map[string]string{}, http.MethodGet)
	e[3] = e[3].SetResponseType(map[string]interface{}{}, http.MethodGet)
	e[3] = e[3].SetResponseType("", http.MethodGet)

	return e
}
