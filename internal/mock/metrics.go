package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type MetricsService struct {
	GetSeriesFn        func(context.Context, *ps.GetMetricSeriesRequest) (*ps.MetricSeries, error)
	GetSeriesFnInvoked bool

	GetInstantFn        func(context.Context, *ps.GetInstantMetricsRequest) (*ps.InstantMetrics, error)
	GetInstantFnInvoked bool
}

func (s *MetricsService) GetSeries(ctx context.Context, req *ps.GetMetricSeriesRequest) (*ps.MetricSeries, error) {
	s.GetSeriesFnInvoked = true
	return s.GetSeriesFn(ctx, req)
}

func (s *MetricsService) GetInstant(ctx context.Context, req *ps.GetInstantMetricsRequest) (*ps.InstantMetrics, error) {
	s.GetInstantFnInvoked = true
	return s.GetInstantFn(ctx, req)
}
