package response

import (
	"encoding/json"
	"github.com/Seann-Moser/go-serve/pkg/pagination"
	"net/http"
	"reflect"
	"strings"
)

type Response struct {
	showError bool
}

type BaseResponseGeneric[T any] struct {
	Message string                 `json:"message"`
	Data    T                      `json:"data,omitempty"`
	Page    *pagination.Pagination `json:"page,omitempty"`
}

type BaseResponse struct {
	Message  string                 `json:"message"`
	Data     interface{}            `json:"data,omitempty"`
	Page     *pagination.Pagination `json:"page,omitempty"`
	skipWrap bool
	array    bool
}

func (b BaseResponse) Encode(r *http.Request, w http.ResponseWriter) error {
	if r.Header.Get("Accept") == "application/json-array" {
		b.array = true
		b.skipWrap = true
	}
	if strings.ToLower(r.URL.Query().Get("wrap")) == "false" {
		b.skipWrap = true
	}
	if strings.ToLower(r.URL.Query().Get("array")) == "true" {
		b.array = true
		b.skipWrap = true
	}
	if !b.skipWrap {
		return json.NewEncoder(w).Encode(b)
	}

	if b.array {
		if b.Data == nil {
			return json.NewEncoder(w).Encode([]interface{}{})
		}
		if isArray(b.Data) {
			return json.NewEncoder(w).Encode(b.Data)
		} else {
			return json.NewEncoder(w).Encode([]interface{}{b.Data})
		}
	}
	if b.Data == nil {
		return json.NewEncoder(w).Encode(struct{}{})

	}
	return json.NewEncoder(w).Encode(b.Data)
}

// isArray checks if the input is an array.
func isArray(i interface{}) bool {
	t := reflect.TypeOf(i)
	if t == nil {
		return false
	}
	kind := t.Kind()
	switch kind {
	case reflect.Slice:
		return true
	case reflect.Array:
		return true
	default:
		return false
	}
}
