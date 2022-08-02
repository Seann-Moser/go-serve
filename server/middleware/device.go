package middleware

import (
	"crypto/sha1"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type UserDevices struct {
	ID        string `db:"id" json:"id" table:"primary" where:"="`
	Name      string `db:"name" json:"name" table:"primary" can_update:"true"`
	IPv4      string `db:"ip_v4" json:"ip_v4" table:"primary"`
	IPv6      string `db:"ip_v6" json:"ip_v6"`
	UserAgent string `db:"user_agent" json:"user_agent" data_type:"text"`
}

func LoadDeviceDetails(r *http.Request) *UserDevices {
	device := &UserDevices{}
	device.Name = strings.ToUpper(uuid.New().String())
	device.loadIP(r)
	device.UserAgent = r.UserAgent()
	device.Name = r.UserAgent()
	return device
}
func (d *UserDevices) loadIP(r *http.Request) {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	for _, ip := range strings.Split(IPAddress, ",") {
		if strings.Count(ip, ":") == 1 {
			ip = ip[:strings.LastIndex(ip, ":")]
		}
		if net.ParseIP(ip).To4() != nil {
			if d.IPv4 != "" {
				d.IPv4 = fmt.Sprintf("%s,%s", d.IPv4, ip)
			} else {
				d.IPv4 = ip
			}

		} else if net.ParseIP(ip).To16() != nil {
			d.IPv6 = fmt.Sprintf("%s,%s", d.IPv6, ip)
		}

	}
}
func (d *UserDevices) GenerateDeviceKey(t string) string {
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("%s-%s-%s-%s-%v", d.ID, d.UserAgent, d.IPv4, d.IPv6, t)))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}
