package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test BaseResponse.Encode
func TestBaseResponse_Encode_JSONArrayHeader(t *testing.T) {
	// Create a request with Accept header as application/json-array
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json-array")
	w := httptest.NewRecorder()

	// Response struct
	response := BaseResponse{
		Message: "Success",
		Data:    []int{1, 2, 3},
	}

	err := response.Encode(req, w)
	assert.NoError(t, err)

	// Check if the response matches the expected output
	expectedResponse, _ := json.Marshal(response.Data)
	assert.JSONEq(t, string(expectedResponse), w.Body.String())
}

func TestBaseResponse_Encode_QueryWrapFalse(t *testing.T) {
	// Create a request with wrap=false query param
	req := httptest.NewRequest(http.MethodGet, "/?wrap=false", nil)
	w := httptest.NewRecorder()

	response := BaseResponse{
		Message: "Success",
		Data:    "Hello",
	}

	err := response.Encode(req, w)
	assert.NoError(t, err)

	expectedResponse, _ := json.Marshal(response.Data)
	assert.JSONEq(t, string(expectedResponse), w.Body.String())
}

func TestBaseResponse_Encode_QueryArrayTrue(t *testing.T) {
	// Create a request with array=true query param
	req := httptest.NewRequest(http.MethodGet, "/?array=true", nil)
	w := httptest.NewRecorder()

	response := BaseResponse{
		Message: "Success",
		Data:    "Item",
	}

	err := response.Encode(req, w)
	assert.NoError(t, err)

	expectedResponse, _ := json.Marshal([]interface{}{"Item"})
	assert.JSONEq(t, string(expectedResponse), w.Body.String())
}

func TestBaseResponse_Encode_Wrapped(t *testing.T) {
	// Create a request without any wrap or array settings
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	response := BaseResponse{
		Message: "Success",
		Data:    "Item",
	}

	err := response.Encode(req, w)
	assert.NoError(t, err)

	expectedResponse, _ := json.Marshal(response)
	assert.JSONEq(t, string(expectedResponse), w.Body.String())
}

func TestBaseResponse_Encode_SingleArrayItem(t *testing.T) {
	// Create a request with array=true and data being an array of length 1
	req := httptest.NewRequest(http.MethodGet, "/?array=true", nil)
	w := httptest.NewRecorder()

	response := BaseResponse{
		Message: "Success",
		Data:    []string{"Item1"},
	}

	err := response.Encode(req, w)
	assert.NoError(t, err)

	expectedResponse, _ := json.Marshal(response.Data)
	assert.JSONEq(t, string(expectedResponse), w.Body.String())
}

func TestIsArray(t *testing.T) {
	// Test for array type
	assert.True(t, isArray([3]int{1, 2, 3}))
	assert.True(t, isArray([]int{1, 2, 3})) // This is a slice, not an array
	assert.False(t, isArray("not an array"))
	assert.False(t, isArray(123))
	assert.False(t, isArray(nil))
}
