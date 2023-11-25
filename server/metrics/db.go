package metrics

import (
	"context"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/opencensus-integrations/ocsql"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

func (m *Metrics) RegisterSQL(ctx context.Context, customTags ...tag.Key) *Metrics {
	ocsql.SQLClientLatencyView.TagKeys = []tag.Key{
		ocsql.GoSQLInstance,
		ocsql.GoSQLMethod,
		ocsql.GoSQLStatus,
		m.VersionTag(),
	}

	ocsql.SQLClientLatencyView.TagKeys = append(
		ocsql.SQLClientLatencyView.TagKeys,
		customTags...,
	)
	err := view.Register(ocsql.SQLClientLatencyView)
	if err != nil {
		ctxLogger.Error(ctx, "failed registering sql client view")
	}
	return m
}
