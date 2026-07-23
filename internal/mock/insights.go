package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type QueryInsightsService struct {
	ListQueriesFn        func(context.Context, *ps.ListQueryInsightsRequest, ...ps.ListOption) ([]*ps.QueryInsight, error)
	ListQueriesFnInvoked bool

	ListErrorsFn        func(context.Context, *ps.ListQueryInsightsErrorsRequest, ...ps.ListOption) ([]*ps.QueryInsightError, error)
	ListErrorsFnInvoked bool

	ListAnomaliesFn        func(context.Context, *ps.ListAnomaliesRequest, ...ps.ListOption) ([]*ps.Anomaly, error)
	ListAnomaliesFnInvoked bool
}

func (s *QueryInsightsService) ListQueries(ctx context.Context, req *ps.ListQueryInsightsRequest, opts ...ps.ListOption) ([]*ps.QueryInsight, error) {
	s.ListQueriesFnInvoked = true
	return s.ListQueriesFn(ctx, req, opts...)
}

func (s *QueryInsightsService) ListErrors(ctx context.Context, req *ps.ListQueryInsightsErrorsRequest, opts ...ps.ListOption) ([]*ps.QueryInsightError, error) {
	s.ListErrorsFnInvoked = true
	return s.ListErrorsFn(ctx, req, opts...)
}

func (s *QueryInsightsService) ListAnomalies(ctx context.Context, req *ps.ListAnomaliesRequest, opts ...ps.ListOption) ([]*ps.Anomaly, error) {
	s.ListAnomaliesFnInvoked = true
	return s.ListAnomaliesFn(ctx, req, opts...)
}

type SchemaRecommendationService struct {
	ListFn        func(context.Context, *ps.ListSchemaRecommendationsRequest, ...ps.ListOption) ([]*ps.SchemaRecommendation, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetSchemaRecommendationRequest) (*ps.SchemaRecommendation, error)
	GetFnInvoked bool

	DismissFn        func(context.Context, *ps.DismissSchemaRecommendationRequest) (*ps.SchemaRecommendation, error)
	DismissFnInvoked bool
}

func (s *SchemaRecommendationService) List(ctx context.Context, req *ps.ListSchemaRecommendationsRequest, opts ...ps.ListOption) ([]*ps.SchemaRecommendation, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req, opts...)
}

func (s *SchemaRecommendationService) Get(ctx context.Context, req *ps.GetSchemaRecommendationRequest) (*ps.SchemaRecommendation, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *SchemaRecommendationService) Dismiss(ctx context.Context, req *ps.DismissSchemaRecommendationRequest) (*ps.SchemaRecommendation, error) {
	s.DismissFnInvoked = true
	return s.DismissFn(ctx, req)
}
