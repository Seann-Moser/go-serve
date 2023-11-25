package metrics

import (
	"context"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/orijtech/gomemcache/memcache"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

func (m *Metrics) RegisterMemCache(ctx context.Context, customTags ...tag.Key) *Metrics {
	memcache.KeyLatencyView.TagKeys = []tag.Key{
		memcache.KeyMethod,
		memcache.KeyStatus,
		m.VersionTag(),
	}

	memcache.KeyLatencyView.TagKeys = append(
		memcache.KeyLatencyView.TagKeys,
		customTags...,
	)

	err := view.Register(memcache.KeyLatencyView)
	if err != nil {
		ctxLogger.Error(ctx, "failed registering memcache view")
	}
	return m
}
