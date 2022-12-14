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
	HasAccessToEndpoint(id string, string, path string, r *http.Request) (bool, error)
	ValidDevice(id string, deviceId string, path string, r *http.Request) (bool, error)
	CanSkipValidation(r *http.Request) bool
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
			DeviceID: LoadDeviceDetails(r).GenerateDeviceKey(c.Salt),
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
func (c *AuthSignature) ContainsFields() bool {
	return len(c.Key) > 0 || len(c.ID) > 0 || c.Expires.Unix() > 0 || len(c.Signature) > 0
}

func (c *AuthSignature) computeSignature(salt string) {
	c.Signature = c.GetSignature(salt)
}

func (c *AuthSignature) GetSignature(salt string) string {
	c.MaxAge = int(c.Expires.Unix())
	signatureRaw := fmt.Sprintf("%s-%s-%s-%d-%d-%s", c.ID, c.Key, c.DeviceID, c.MaxAge, c.Expires.Unix(), salt)
	hasher := sha256.New()
	hasher.Write([]byte(signatureRaw))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
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
