package cookies

import (
	"net/http"
	"strconv"
	"time"
)

func AuthFromCookies(r *http.Request) *AuthSignature {
	auth := &AuthSignature{}
	auth.ID = getCookieValue(CookieID, r)
	auth.Key = getCookieValue(CookieKey, r)
	auth.Signature = getCookieValue(CookieSignature, r)
	auth.DeviceID = getCookieValue(CookieDeviceId, r)
	expires, _ := strconv.Atoi(getCookieValue(CookieExpires, r))

	auth.Expires = time.Unix(int64(expires), 0)
	maxAge, _ := strconv.Atoi(getCookieValue(CookieMaxAge, r))

	auth.MaxAge = maxAge
	return auth
}

func getCookieValue(key string, r *http.Request) string {
	cookie, err := r.Cookie(key)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func getCookie(auth *AuthSignature, key, value, path string) *http.Cookie {
	if len(path) == 0 {
		return &http.Cookie{
			Name:    key,
			Value:   value,
			Expires: auth.Expires,
			MaxAge:  auth.MaxAge,
		}
	}
	return &http.Cookie{
		Name:    key,
		Value:   value,
		Expires: auth.Expires,
		Path:    path,
		MaxAge:  auth.MaxAge,
	}
}
