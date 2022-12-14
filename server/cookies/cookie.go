package cookies

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/pkg/response"
	"github.com/Seann-Moser/go-serve/server/device"
)

type Cookies struct {
	DefaultExpiresDuration time.Duration
	Salt                   string
	VerifySignature        bool
	Response               *response.Response
	Logger                 *zap.Logger
	authFunctions          AuthFunctions
}

func New(salt string, verifySignature bool, defaultExpires time.Duration, showError bool, authFunctions AuthFunctions, Logger *zap.Logger) *Cookies {
	return &Cookies{
		DefaultExpiresDuration: defaultExpires,
		Salt:                   salt,
		VerifySignature:        verifySignature,
		Response:               response.NewResponse(showError, Logger),
		Logger:                 Logger,
		authFunctions:          authFunctions,
	}
}

func (c *Cookies) DeviceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.SetDeviceID(w, r)
		next.ServeHTTP(w, r)
		return
	})
}

func (c *Cookies) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//canSkip := c.authFunctions.CanSkipValidation(r)

		auth := AuthFromCookies(r)
		if c.VerifySignature && auth.ContainsFields() {
			authSignature := c.GetAuthSignature(auth.ID, auth.Key, &auth.Expires, r)
			if auth.Signature != authSignature.Signature {
				c.RemoveCookies(w, r)
				c.Logger.Warn("invalid signature", zap.String("current", auth.Signature), zap.String("expected", authSignature.Signature))
				c.Response.Error(w, nil, http.StatusUnauthorized, "invalid signature")
				return
			}
		}
		path := r.URL.Path
		for _, v := range mux.Vars(r) {
			path = strings.ReplaceAll(path, v, "%")
		}
		if access, err := c.authFunctions.HasAccessToEndpoint(auth.ID, auth.Key, path, r); !access || err != nil {
			c.RemoveCookies(w, r)
			c.Response.Error(w, nil, http.StatusUnauthorized, "unauthorized access to endpoint")
			return
		}

		if access, err := c.authFunctions.ValidDevice(auth.ID, auth.DeviceID, path, r); !access || err != nil {
			c.RemoveCookies(w, r)
			c.Response.Error(w, nil, http.StatusUnauthorized, "invalid device")
			return
		}

		next.ServeHTTP(w, r)
	})
}
func (c *Cookies) GetAuthSignature(id, key string, expires *time.Time, r *http.Request) *AuthSignature {
	var auth *AuthSignature
	if r == nil {
		auth = &AuthSignature{
			ID:      id,
			Key:     key,
			Expires: time.Now().Add(c.DefaultExpiresDuration),
		}
	} else {
		auth = &AuthSignature{
			ID:       id,
			Key:      key,
			Expires:  time.Now().Add(c.DefaultExpiresDuration),
			DeviceID: device.GetDeviceFromRequest(r).GenerateDeviceKey(c.Salt),
		}
	}
	if expires != nil {
		auth.Expires = *expires
	}
	auth.computeSignature(c.Salt)
	return auth
}
func (c *Cookies) SetAuthCookies(w http.ResponseWriter, r *http.Request, id string, key string, path string) error {
	auth := c.GetAuthSignature(id, key, nil, r)

	var cookies []*http.Cookie
	cookies = append(cookies, getCookie(auth, CookieDeviceId, auth.DeviceID, path))
	cookies = append(cookies, getCookie(auth, CookieID, auth.ID, path))
	cookies = append(cookies, getCookie(auth, CookieKey, auth.Key, path))
	cookies = append(cookies, getCookie(auth, CookieSignature, auth.Signature, path))
	cookies = append(cookies, getCookie(auth, CookieTimestamp, strconv.Itoa(int(time.Now().Unix())), path))
	cookies = append(cookies, getCookie(auth, CookieExpires, strconv.Itoa(int(auth.Expires.Unix())), path))
	cookies = append(cookies, getCookie(auth, CookieMaxAge, strconv.Itoa(int(auth.MaxAge)), path))
	for _, cookie := range cookies {
		r.AddCookie(cookie)
		http.SetCookie(w, cookie)
	}
	return nil
}

func (c *Cookies) RemoveCookies(w http.ResponseWriter, r *http.Request) {
	for _, c := range r.Cookies() {
		c.MaxAge = -1
		http.SetCookie(w, c)
	}
}

func (c *Cookies) SetDeviceID(w http.ResponseWriter, r *http.Request) {
	device := device.GetDeviceFromRequest(r)
	key := device.GenerateDeviceKey("")
	cookie := http.Cookie{
		Name:    CookieDeviceId,
		Value:   key,
		Expires: time.Now().Add(c.DefaultExpiresDuration),
		MaxAge:  int(c.DefaultExpiresDuration.Seconds()),
	}
	r.AddCookie(&cookie)
	http.SetCookie(w, &cookie)
}
