package mock

import (
	"context"
	"encoding/json"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type MetricsService struct {
	GetSeriesFn        func(context.Context, *ps.GetMetricSeriesRequest) (*ps.MetricSeries, error)
	GetSeriesFnInvoked bool

	GetInstantFn        func(context.Context, *ps.GetInstantMetricsRequest) (*ps.InstantMetrics, error)
	GetInstantFnInvoked bool

	GetQuerySeriesFn        func(context.Context, *ps.GetQueryMetricSeriesRequest) (*ps.MetricSeries, error)
	GetQuerySeriesFnInvoked bool

	GetTablesFn        func(context.Context, *ps.GetBranchMetricsRequest) (json.RawMessage, error)
	GetTablesFnInvoked bool

	GetKeyspaceTablesFn        func(context.Context, *ps.GetBranchMetricsRequest) (json.RawMessage, error)
	GetKeyspaceTablesFnInvoked bool

	GetTabletSeriesFn        func(context.Context, *ps.GetTabletMetricSeriesRequest) (*ps.MetricSeries, error)
	GetTabletSeriesFnInvoked bool

	GetInstantTabletsFn        func(context.Context, *ps.GetInstantTabletMetricsRequest) (*ps.InstantMetrics, error)
	GetInstantTabletsFnInvoked bool

	GetTagSeriesFn        func(context.Context, *ps.GetTagMetricSeriesRequest) (*ps.MetricSeries, error)
	GetTagSeriesFnInvoked bool
}

func (s *MetricsService) GetSeries(ctx context.Context, req *ps.GetMetricSeriesRequest) (*ps.MetricSeries, error) {
	s.GetSeriesFnInvoked = true
	return s.GetSeriesFn(ctx, req)
}

func (s *MetricsService) GetInstant(ctx context.Context, req *ps.GetInstantMetricsRequest) (*ps.InstantMetrics, error) {
	s.GetInstantFnInvoked = true
	return s.GetInstantFn(ctx, req)
}

func (s *MetricsService) GetQuerySeries(ctx context.Context, req *ps.GetQueryMetricSeriesRequest) (*ps.MetricSeries, error) {
	s.GetQuerySeriesFnInvoked = true
	return s.GetQuerySeriesFn(ctx, req)
}

func (s *MetricsService) GetTables(ctx context.Context, req *ps.GetBranchMetricsRequest) (json.RawMessage, error) {
	s.GetTablesFnInvoked = true
	return s.GetTablesFn(ctx, req)
}

func (s *MetricsService) GetKeyspaceTables(ctx context.Context, req *ps.GetBranchMetricsRequest) (json.RawMessage, error) {
	s.GetKeyspaceTablesFnInvoked = true
	return s.GetKeyspaceTablesFn(ctx, req)
}

func (s *MetricsService) GetTabletSeries(ctx context.Context, req *ps.GetTabletMetricSeriesRequest) (*ps.MetricSeries, error) {
	s.GetTabletSeriesFnInvoked = true
	return s.GetTabletSeriesFn(ctx, req)
}

func (s *MetricsService) GetInstantTablets(ctx context.Context, req *ps.GetInstantTabletMetricsRequest) (*ps.InstantMetrics, error) {
	s.GetInstantTabletsFnInvoked = true
	return s.GetInstantTabletsFn(ctx, req)
}

func (s *MetricsService) GetTagSeries(ctx context.Context, req *ps.GetTagMetricSeriesRequest) (*ps.MetricSeries, error) {
	s.GetTagSeriesFnInvoked = true
	return s.GetTagSeriesFn(ctx, req)
}
