package middle

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/pkg/response"
)

const (
	CookieID        = "id"
	CookieKey       = "key"
	CookieSignature = "signature"
	CookieTimestamp = "timestamp"
	CookieExpires   = "expires"
	CookieDeviceId  = "device_id"
	CookieMaxAge    = "max_age"
)

type AuthFunctions interface {
	HasAccessToEndpoint(id string, string, path string) (bool, error)
	ValidDevice(id string, deviceId string, path string) (bool, error)
}
type Cookies struct {
	DefaultExpiresDuration time.Duration
	Salt                   string
	VerifySignature        bool
	Response               *response.Response
	Logger                 *zap.Logger
	authFunctions          AuthFunctions
}

func NewCookies(salt string, verifySignature bool, defaultExpires time.Duration, showError bool, authFunctions AuthFunctions, Logger *zap.Logger) *Cookies {
	return &Cookies{
		DefaultExpiresDuration: defaultExpires,
		Salt:                   salt,
		VerifySignature:        verifySignature,
		Response:               response.NewResponse(showError, Logger),
		Logger:                 Logger,
		authFunctions:          authFunctions,
	}
}

func (c *Cookies) CookiesDeviceID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.SetDeviceID(w, r)
		next.ServeHTTP(w, r)
		return
	})
}

func (c *Cookies) CookiesAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := AuthFromCookies(r)
		if c.VerifySignature {
			if !auth.Valid() {
				c.RemoveCookies(w, r)
				c.Response.Error(w, nil, http.StatusUnauthorized, "missing cookies")
				return
			}
			authSignature := &AuthSignature{
				ID:  auth.ID,
				Key: auth.Key,
			}
			auth.computeSignature(r, c.DefaultExpiresDuration, c.Salt)
			if auth.Signature != authSignature.Signature {
				c.RemoveCookies(w, r)
				c.Response.Error(w, nil, http.StatusUnauthorized, "invalid signature")
				return
			}
		}
		path := r.URL.Path
		for _, v := range mux.Vars(r) {
			path = strings.ReplaceAll(path, v, "%")
		}
		if access, err := c.authFunctions.HasAccessToEndpoint(auth.ID, auth.Key, path); !access || err != nil {
			c.RemoveCookies(w, r)
			c.Response.Error(w, nil, http.StatusUnauthorized, "unauthorized access to endpoint")
			return
		}

		if access, err := c.authFunctions.ValidDevice(auth.ID, auth.DeviceID, path); !access || err != nil {
			c.RemoveCookies(w, r)
			c.Response.Error(w, nil, http.StatusUnauthorized, "invalid device")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (c *Cookies) SetAuthCookies(w http.ResponseWriter, r *http.Request, id string, key string) error {
	auth := &AuthSignature{
		ID:  id,
		Key: key,
	}
	auth.computeSignature(r, c.DefaultExpiresDuration, c.Salt)

	var cookies []*http.Cookie
	cookies = append(cookies, getCookie(auth, CookieDeviceId, auth.DeviceID))
	cookies = append(cookies, getCookie(auth, CookieID, auth.ID))
	cookies = append(cookies, getCookie(auth, CookieKey, auth.Key))
	cookies = append(cookies, getCookie(auth, CookieSignature, auth.Signature))
	cookies = append(cookies, getCookie(auth, CookieTimestamp, strconv.Itoa(int(time.Now().Unix()))))
	cookies = append(cookies, getCookie(auth, CookieExpires, strconv.Itoa(int(auth.Expires.Unix()))))
	cookies = append(cookies, getCookie(auth, CookieMaxAge, strconv.Itoa(int(auth.MaxAge))))
	for _, c := range cookies {
		r.AddCookie(c)
		http.SetCookie(w, c)
	}
	return nil
}
func getCookie(auth *AuthSignature, key, value string) *http.Cookie {
	return &http.Cookie{
		Name:    key,
		Value:   value,
		Expires: auth.Expires,
		MaxAge:  auth.MaxAge,
	}
}

type AuthSignature struct {
	ID        string
	Key       string
	DeviceID  string
	Expires   time.Time
	MaxAge    int
	Signature string
}

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
func (c *AuthSignature) Valid() bool {
	if len(c.Key) == 0 || len(c.ID) == 0 || c.Expires.Unix() == 0 || len(c.Signature) == 0 || len(c.DeviceID) == 0 {
		return false
	}
	return true
}
func (c *AuthSignature) computeSignature(r *http.Request, defaultExpires time.Duration, salt string) {
	device := LoadDeviceDetails(r)
	if c.DeviceID == "" {
		c.DeviceID = device.GenerateDeviceKey(salt)
	}
	if c.Expires.Unix() == 0 {
		c.Expires = time.Now().Add(defaultExpires)
	}

	c.MaxAge = int(defaultExpires.Seconds())

	signatureRaw := fmt.Sprintf("%s-%s-%s-%d-%d-%s", c.ID, c.Key, c.DeviceID, c.MaxAge, c.Expires.Unix(), salt)
	hasher := sha256.New()
	hasher.Write([]byte(signatureRaw))
	c.Signature = base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

func (c *Cookies) RemoveCookies(w http.ResponseWriter, r *http.Request) {
	for _, c := range r.Cookies() {
		c.MaxAge = -1
		http.SetCookie(w, c)
	}
}

func (c *Cookies) SetDeviceID(w http.ResponseWriter, r *http.Request) {
	device := LoadDeviceDetails(r)
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
