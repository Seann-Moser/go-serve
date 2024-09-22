package generator

import (
	"github.com/Seann-Moser/go-serve/pkg/generator/generators"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestClientGenerate tests the Generate function of the Client
func TestClientGenerate(t *testing.T) {
	// Setup

	mockGen := generators.GoClientGenerator{}
	client := New()

	// Expectations

	// Execute
	err := client.Generate([]generators.Generator{mockGen}, GetEndpoints()...)

	// Assert
	assert.NoError(t, err)

}

// TestNuxtClientGenerateError tests the Generate function of the Client when an error occurs
func TestNuxtClientGenerateError(t *testing.T) {
	// Setup
	mockGen := generators.NuxtPluginGenerator{}
	client := New()

	// Expectations

	// Execute
	err := client.Generate([]generators.Generator{mockGen}, GetEndpoints()...)
	assert.NoError(t, err)
	// Assert
}

// TestClientGenerateError tests the Generate function of the Client when an error occurs
func TestClientGenerateError(t *testing.T) {
	// Setup
	mockGen := generators.GoClientGenerator{}
	client := New()

	// Expectations

	// Execute
	err := client.Generate([]generators.Generator{mockGen}, GetEndpoints()...)
	assert.NoError(t, err)
	// Assert
}

func GetEndpoints() []*endpoints.Endpoint {
	//c := Client{}
	e := []*endpoints.Endpoint{
		{
			SubDomain: "test",
			URLPath:   "/account/{account_id}/user/{user_id}",
			Methods:   []string{http.MethodPost},
			//HandlerFunc: c.HandlerFuncs,
			ResponseTypeMap: map[string]interface{}{
				"POST": generators.ClientFunc{},
			},
		},
		{
			SubDomain: "test",
			URLPath:   "/account/{account_id}/user/{user_id}",
			Methods:   []string{http.MethodGet},
			//HandlerFunc: HandlerFuncs,
		},
		{
			SubDomain: "test",
			URLPath:   "/account/{account_id}/user/{user_id}/settings",
			Methods:   []string{http.MethodGet, http.MethodDelete},
			Headers:   []string{"header", "test"},
			//HandlerFunc: c.HandlerFuncs,
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
	//e[1] = e[1].SetResponseType(RequestData{}, http.MethodGet)
	//e[2] = e[2].SetRequestType([]ResponseData{}, http.MethodGet)
	e[3] = e[3].SetRequestType(map[string]string{}, http.MethodGet)
	e[3] = e[3].SetResponseType(map[string]interface{}{}, http.MethodGet)
	e[3] = e[3].SetResponseType("", http.MethodGet)

	return e
}
