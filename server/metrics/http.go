package metrics

import (
	"context"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"net/http"
	"time"
)

func (m *Metrics) NewHttpClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &ochttp.Transport{
			Base: http.DefaultTransport,
		},
	}
}

func (m *Metrics) RegisterHTTPClient(ctx context.Context, customTags ...tag.Key) *Metrics {
	ochttp.ClientRoundtripLatencyDistribution.TagKeys = append(
		ochttp.ClientRoundtripLatencyDistribution.TagKeys,
		ochttp.KeyClientHost,
		ochttp.Method,
		m.VersionTag(),
	)

	ochttp.ClientRoundtripLatencyDistribution.TagKeys = append(
		ochttp.ClientRoundtripLatencyDistribution.TagKeys,
		customTags...,
	)
	err := view.Register(ochttp.ClientRoundtripLatencyDistribution)
	if err != nil {
		ctxLogger.Error(ctx, "failed registering http client view")
	}
	return m
}

func (m *Metrics) RegisterHTTPServer(ctx context.Context, customTags ...tag.Key) *Metrics {
	ochttp.ServerLatencyView.TagKeys = append(
		ochttp.ServerLatencyView.TagKeys,
		ochttp.StatusCode,
		ochttp.Method,
		ochttp.KeyServerRoute,
		m.VersionTag(),
	)

	ochttp.ServerLatencyView.TagKeys = append(
		ochttp.ServerLatencyView.TagKeys,
		customTags...,
	)
	err := view.Register(ochttp.ServerLatencyView)
	if err != nil {
		ctxLogger.Error(ctx, "failed registering http server view")
	}
	return m
}
