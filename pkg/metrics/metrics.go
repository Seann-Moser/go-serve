package metrics

import (
	"context"
	"errors"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.uber.org/zap"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"
)

type Metrics struct {
	Name           string
	Namespace      string
	Enabled        bool
	Version        string
	router         *http.ServeMux
	port           int
	zipkinEnpdoint string
}

func MetricFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("metrics", pflag.ExitOnError)
	fs.String("metrics-namespace", "", "")
	fs.String("metrics-version", "dev", "")
	fs.String("metrics-name", "dev", "")
	fs.String("metrics-zipkin-endpoint", "", "")

	fs.Bool("metrics-enabled", false, "")
	fs.Int("metrics-port", 8081, "")
	return fs
}

func New() *Metrics {
	return &Metrics{
		Namespace:      viper.GetString("metrics-namespace"),
		Version:        viper.GetString("metrics-version"),
		Enabled:        viper.GetBool("metrics-enabled"),
		port:           viper.GetInt("metrics-port"),
		Name:           viper.GetString("metrics-name"),
		zipkinEnpdoint: viper.GetString("metrics-zipkin-endpoint"),
	}
}

func (m *Metrics) StartServer(ctx context.Context) error {
	m.router = http.NewServeMux()
	otelShutdown, err := setupOTelSDK(ctx, m.zipkinEnpdoint, m.Name, m.Version)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()
	m.router.HandleFunc("/debug/pprof/", pprof.Index)
	m.router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	m.router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	m.router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	m.router.HandleFunc("/debug/pprof/trace", pprof.Trace)

	m.router.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(m.port),
		Handler: m.router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			ctxLogger.Error(ctx, "failed creating metrics server", zap.Error(err))
		}
	}()

	ctxLogger.Info(ctx, "staring metrics server", zap.String("address", server.Addr), zap.Int("port", m.port))
	<-ctx.Done()
	ctxLogger.Info(ctx, "server stopped")
	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctxShutDown); err != nil {
		ctxLogger.Error(ctx, "server Shutdown Failed", zap.Error(err))
		return err
	}
	ctxLogger.Info(ctx, "server exited properly")

	return nil
}

// setupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func setupOTelSDK(ctx context.Context, zipkinEndpoint string, name, version string) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error
	r, err := newResource(name, version)
	if err != nil {
		return nil, err
	}
	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)
	if zipkinEndpoint != "" {
		// Set up trace provider.
		tracerProvider, err := newTraceProvider(zipkinEndpoint, r)
		if err != nil {
			return nil, err
		}
		shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
		otel.SetTracerProvider(tracerProvider)
	}

	// Set up meter provider.
	meterProvider, err := newMeterProvider(r)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTraceProvider(zipkinEndpoint string, res *resource.Resource) (*trace.TracerProvider, error) {
	exp, err := zipkin.New(zipkinEndpoint)
	if err != nil {
		return nil, err
	}
	traceProvider := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(exp,
			// Default is 5s. Set to 1s for demonstrative purposes.
			trace.WithBatchTimeout(time.Second)),
	)
	return traceProvider, nil
}

func newResource(name, version string) (*resource.Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(name),
			semconv.ServiceVersion(version),
		))
}

func newMeterProvider(res *resource.Resource) (*metric.MeterProvider, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}
	provider := metric.NewMeterProvider(metric.WithResource(res), metric.WithReader(exporter))
	return provider, nil
}
