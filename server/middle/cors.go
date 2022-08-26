package middle

import (
	"net/http"
	"strconv"
	"strings"
)

type CorsMiddleware struct {
	AllowedOrigins     []string
	AllowedMethods     []string
	AllowedHeaders     []string
	AllowedCredentials bool
}

func NewCorsMiddleware(origin, methods, headers []string, creds bool) *CorsMiddleware {
	return &CorsMiddleware{
		AllowedOrigins:     origin,
		AllowedMethods:     methods,
		AllowedHeaders:     headers,
		AllowedCredentials: creds,
	}
}
func (c *CorsMiddleware) Cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", c.getOrigins())
		w.Header().Set("Access-Control-Allow-Methods", c.getMethods())
		w.Header().Set("Access-Control-Allow-Headers", c.getHeaders())
		w.Header().Set("Access-Control-Allow-Credentials", strconv.FormatBool(c.AllowedCredentials))
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Next
		next.ServeHTTP(w, r)
		return
	})
}

func (c *CorsMiddleware) getOrigins() string {
	return getCorsData(c.AllowedOrigins)
}
func (c *CorsMiddleware) getMethods() string {
	return getCorsData(c.AllowedMethods)
}
func (c *CorsMiddleware) getHeaders() string {
	return getCorsData(c.AllowedHeaders)
}

func getCorsData(list []string) string {
	if list == nil {
		return "*"
	}
	if len(list) == 0 {
		return "*"
	}
	return strings.Join(list, ", ")
}
