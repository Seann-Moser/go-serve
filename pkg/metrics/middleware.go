package metrics

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"net/http"
	"strings"
	"time"
)

var httpMiddlewareMeter = otel.Meter("endpoint-metrics")

var (
	httpTotalRequests metric.Int64UpDownCounter = nil
	httpLatency       metric.Float64Histogram   = nil
)

func (m *Metrics) Middleware() func(next http.Handler) http.Handler {
	_ = m.createMeasures()
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if strings.EqualFold(r.URL.Path, "/healthcheck") || strings.EqualFold(r.URL.Path, "/metrics") {
				next.ServeHTTP(w, r)
				return
			}
			entry := m.newAuditLog(r)

			ww := NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				m.write(r.Context(), entry, ww.Status(), ww.BytesWritten(), ww.Header(), time.Since(t1), nil)
			}()

			next.ServeHTTP(ww, WithLogEntry(r, m.write))
		}
		return http.HandlerFunc(fn)
	}
}
func (m *Metrics) newAuditLog(r *http.Request) *AuditLog {
	entry := &AuditLog{
		Service: m.Name,
		Path:    getRawPath(r),
		Method:  r.Method,
		Version: m.Version,
	}
	return entry
}

func (m *Metrics) write(ctx context.Context, entry *AuditLog, status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	entry.StatusCode = int64(status)
	entry.Latency = elapsed.Milliseconds()
	httpTotalRequests.Add(ctx, 1,
		metric.WithAttributes(
			semconv.HTTPStatusCode(status),
			semconv.HTTPMethod(entry.Method),
			semconv.HTTPRoute(entry.Path),
			semconv.HTTPResponseContentLength(bytes),
		),
	)
	httpLatency.Record(ctx, float64(entry.Latency), metric.WithAttributes(
		semconv.HTTPStatusCode(status),
		semconv.HTTPMethod(entry.Method),
		semconv.HTTPRoute(entry.Path),
		semconv.HTTPResponseContentLength(bytes),
	))
}

func (m *Metrics) createMeasures() error {
	var err error
	httpTotalRequests, err = httpMiddlewareMeter.Int64UpDownCounter(
		"server.request.counter",
		metric.WithDescription("Number of finished API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return err
	}

	httpLatency, err = httpMiddlewareMeter.Float64Histogram(
		"server.latency",
		metric.WithUnit("ms"),
		metric.WithDescription("Measures the duration of inbound HTTP requests."),
	)
	if err != nil {
		return err
	}

	return nil
}

func getRawPath(r *http.Request) string {
	rawPath := r.URL.Path
	muxVars := mux.Vars(r)
	for k, v := range muxVars {
		rawPath = strings.ReplaceAll(rawPath, v, fmt.Sprintf("{%s}", k))
	}
	return rawPath
}
