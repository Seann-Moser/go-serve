package metrics

import (
	"context"
	ocPromethus "contrib.go.opencensus.io/exporter/prometheus"
	"errors"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/runmetrics"
	"go.opencensus.io/tag"
	"go.uber.org/zap"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"
)

type Metrics struct {
	Namespace   string
	Enabled     bool
	OnError     func(err error)
	ConstLabels map[string]string
	Version     string
	router      *http.ServeMux
	port        int
}

func MetricFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("metrics", pflag.ExitOnError)
	fs.String("metrics-namespace", "", "")
	fs.String("metrics-version", "dev", "")
	fs.Bool("metrics-enabled", false, "")
	fs.Int("metrics-port", 8081, "")
	return fs
}

func New(constLabels map[string]string) *Metrics {
	return &Metrics{
		Namespace:   viper.GetString("metrics-namespace"),
		Version:     viper.GetString("metrics-version"),
		Enabled:     viper.GetBool("metrics-enabled"),
		port:        viper.GetInt("metrics-port"),
		OnError:     DefaultError,
		ConstLabels: constLabels,
	}
}

func DefaultError(err error) {
	ctxLogger.GetLogger(context.Background()).Error("metrics failed", zap.Error(err))
}

func (m *Metrics) VersionTag() tag.Key {
	key, err := tag.NewKey(m.Version)
	if err != nil {
		return tag.Key{}
	}
	return key
}

func (m *Metrics) RegisterRunMetrics(ctx context.Context) *Metrics {
	err := runmetrics.Enable(runmetrics.RunMetricOptions{
		EnableCPU:    true,
		EnableMemory: true,
	})
	if err != nil {
		ctxLogger.Error(ctx, "failed registering run metrics")
	}

	return m
}

func (m *Metrics) AddConstLabel(key, value string) {
	if m.ConstLabels == nil {
		m.ConstLabels = map[string]string{}
	}
	m.ConstLabels[key] = value
}

func (m *Metrics) StartServer(ctx context.Context) error {
	m.router = http.NewServeMux()

	m.router.HandleFunc("/debug/pprof/", pprof.Index)
	m.router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	m.router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	m.router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	m.router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	exporter, err := ocPromethus.NewExporter(ocPromethus.Options{
		Namespace:   m.Namespace,
		OnError:     m.OnError,
		ConstLabels: m.ConstLabels,
	})
	if err != nil {
		return err
	}
	m.router.Handle("/metrics", exporter)

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
