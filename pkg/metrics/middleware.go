package metrics

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"net/http"
	"strings"
	"time"
)

var httpMiddlewareMeter = otel.Meter("endpoint-metrics")

var (
	httpTotalRequests metric.Int64UpDownCounter            = nil
	httpLatency       metric.Float64Histogram              = nil
	histograms        map[string]metric.Float64Histogram   = map[string]metric.Float64Histogram{}
	upDownCounters    map[string]metric.Int64UpDownCounter = map[string]metric.Int64UpDownCounter{}
	counters          map[string]metric.Int64Counter       = map[string]metric.Int64Counter{}

	meters map[string]metric.Meter = map[string]metric.Meter{}
)

func Measure[T int64 | float64](ctx context.Context, name string, v T, attributes ...attribute.KeyValue) error {
	if !Find(name) {
		return fmt.Errorf("measure %s not found", name)
	}

	switch any(v).(type) {
	case int64:
		if m, found := counters[name]; found {
			m.Add(ctx, int64(v),
				metric.WithAttributes(
					attributes...,
				),
			)
			return nil
		}
		if m, found := upDownCounters[name]; found {
			m.Add(ctx, int64(v),
				metric.WithAttributes(
					attributes...,
				),
			)
			return nil
		}
	case float64:
		if m, found := histograms[name]; found {
			m.Record(ctx, float64(v),
				metric.WithAttributes(
					attributes...,
				),
			)
			return nil
		}
	default:
		return fmt.Errorf("unsupported type %T", v)
	}

	return nil
}

func Find(name string) bool {
	if _, found := counters[name]; found {
		return true
	}
	if _, found := histograms[name]; found {
		return true
	}
	if _, found := upDownCounters[name]; found {
		return true
	}
	return false
}

func RegisterHistogram(name string, meter string, options ...metric.Float64HistogramOption) error {
	//m, found := meters[meter]
	//if !found {
	//	meters[meter] = otel.Meter(meter)
	//	m = meters[meter]
	//}
	histogram, err := httpMiddlewareMeter.Float64Histogram(
		name,
		options...,
	)
	if err != nil {
		return err
	}
	histograms[name] = histogram
	return nil
}

func RegisterCounter(name string, meter string, options ...metric.Int64CounterOption) error {
	//m, found := meters[meter]
	//if !found {
	//	meters[meter] = otel.Meter(meter)
	//	m = meters[meter]
	//}
	counter, err := httpMiddlewareMeter.Int64Counter(
		name,
		options...,
	)
	if err != nil {
		return err
	}
	counters[name] = counter
	return nil
}

func RegisterUpDownCounter(name string, meter string, options ...metric.Int64UpDownCounterOption) error {
	//m, found := meters[meter]
	//if !found {
	//	meters[meter] = otel.Meter(meter)
	//	m = meters[meter]
	//}
	counter, err := httpMiddlewareMeter.Int64UpDownCounter(
		name,
		options...,
	)
	if err != nil {
		return err
	}
	upDownCounters[name] = counter
	return nil
}

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
			semconv.HTTPResponseStatusCode(status),
			semconv.HTTPRequestMethodOriginal(entry.Method),
			semconv.HTTPRoute(entry.Path),
		),
	)
	httpLatency.Record(ctx, float64(entry.Latency), metric.WithAttributes(
		semconv.HTTPResponseStatusCode(status),
		semconv.HTTPRequestMethodOriginal(entry.Method),
		semconv.HTTPRoute(entry.Path),
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
