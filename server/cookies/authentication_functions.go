package cookies

import "net/http"

type AuthFunctions interface {
	HasAccessToEndpoint(id string, string, path string, r *http.Request) (bool, error)
	ValidDevice(id string, deviceId string, path string, r *http.Request) (bool, error)
	CanSkipValidation(r *http.Request) bool
}
