package cookies

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
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

type AuthSignature struct {
	ID        string
	Key       string
	DeviceID  string
	Expires   time.Time
	MaxAge    int
	Signature string
}

func (c *AuthSignature) Valid() bool {
	if len(c.Key) == 0 || len(c.ID) == 0 || c.Expires.Unix() == 0 || len(c.Signature) == 0 || len(c.DeviceID) == 0 {
		return false
	}
	return true
}
func (c *AuthSignature) ContainsFields() bool {
	return len(c.Key) > 0 || len(c.ID) > 0 || c.Expires.Unix() > 0 || len(c.Signature) > 0 || len(c.DeviceID) > 0
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
