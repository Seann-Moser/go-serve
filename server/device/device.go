package device

import (
	"crypto/sha1"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type Device struct {
	ID          string `db:"id" json:"id" qc:"primary;join,where::="`
	Name        string `db:"name" json:"name" qc:"primary;update"`
	IPv4        string `db:"ip_v4" json:"ip_v4" qc:"primary"`
	IPv6        string `db:"ip_v6" json:"ip_v6" qc:"primary"`
	UserAgent   string `db:"user_agent" json:"user_agent" qc:"data_type::text;primary"`
	Active      bool   `db:"active" json:"active" qc:"default::true;update;where::="`
	UpdatedDate string `db:"updated_date" json:"updated_date" qc:"skip;data_type::TIMESTAMP;default::NOW() ON UPDATE CURRENT_TIMESTAMP"`
	CreatedDate string `db:"created_date" json:"created_date" qc:"skip;data_type::TIMESTAMP;default::NOW()"`
}

func GetDeviceFromRequest(r *http.Request) *Device {
	device := &Device{}
	device.Name = strings.ToUpper(uuid.New().String())
	device.loadIP(r)
	device.UserAgent = r.UserAgent()
	device.Name = r.UserAgent()
	return device
}

func (d *Device) loadIP(r *http.Request) {
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
func (d *Device) GenerateDeviceKey(salt string) string {
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("%s-%s-%s-%s-%v", d.ID, d.UserAgent, d.IPv4, d.IPv6, salt)))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}
