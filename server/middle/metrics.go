package middle

import (
	"net/http"

	"go.uber.org/zap"
)

type Metrics struct {
	logger        *zap.Logger
	LogDeviceInfo bool
}

func NewMetrics(logDevice bool, logger *zap.Logger) *Metrics {
	return &Metrics{
		logger:        logger,
		LogDeviceInfo: logDevice,
	}
}
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.LogDeviceInfo {
			device := LoadDeviceDetails(r)
			m.logger.Debug("device hit endpoint",
				zap.String("endpoint", r.URL.String()),
				zap.String("ipv4", device.IPv4),
				zap.String("ipv6", device.IPv6),
				zap.String("user-agent", device.UserAgent),
				zap.String("device-hash", device.GenerateDeviceKey("")),
			)
		} else {
			m.logger.Debug("hit endpoint",
				zap.String("endpoint", r.URL.String()))
		}

		next.ServeHTTP(w, r)
		return
	})
}
