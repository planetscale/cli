package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type QueryInsightsService struct {
	ListQueriesFn        func(context.Context, *ps.ListQueryInsightsRequest, ...ps.ListOption) ([]*ps.QueryInsight, error)
	ListQueriesFnInvoked bool

	ListQuerySamplesFn        func(context.Context, *ps.ListQuerySamplesRequest, ...ps.ListOption) ([]*ps.QuerySample, error)
	ListQuerySamplesFnInvoked bool

	ListErrorsFn        func(context.Context, *ps.ListQueryInsightsErrorsRequest, ...ps.ListOption) ([]*ps.QueryInsightError, error)
	ListErrorsFnInvoked bool

	ListErrorQueriesFn        func(context.Context, *ps.ListErrorQueriesRequest, ...ps.ListOption) ([]*ps.QuerySample, error)
	ListErrorQueriesFnInvoked bool

	ListAnomaliesFn        func(context.Context, *ps.ListAnomaliesRequest, ...ps.ListOption) ([]*ps.Anomaly, error)
	ListAnomaliesFnInvoked bool

	GetAnomalyFn        func(context.Context, *ps.GetAnomalyRequest) (*ps.Anomaly, error)
	GetAnomalyFnInvoked bool

	ListTagsFn        func(context.Context, *ps.ListQueryTagsRequest, ...ps.ListOption) ([]*ps.QueryTag, error)
	ListTagsFnInvoked bool

	GetTagFn        func(context.Context, *ps.GetQueryTagRequest, ...ps.ListOption) (*ps.QueryTag, error)
	GetTagFnInvoked bool

	ListTagSummariesFn        func(context.Context, *ps.ListTagSummariesRequest, ...ps.ListOption) ([]*ps.TagSummary, error)
	ListTagSummariesFnInvoked bool
}

func (s *QueryInsightsService) ListQueries(ctx context.Context, req *ps.ListQueryInsightsRequest, opts ...ps.ListOption) ([]*ps.QueryInsight, error) {
	s.ListQueriesFnInvoked = true
	return s.ListQueriesFn(ctx, req, opts...)
}

func (s *QueryInsightsService) ListErrorQueries(ctx context.Context, req *ps.ListErrorQueriesRequest, opts ...ps.ListOption) ([]*ps.QuerySample, error) {
	s.ListErrorQueriesFnInvoked = true
	return s.ListErrorQueriesFn(ctx, req, opts...)
}

func (s *QueryInsightsService) GetAnomaly(ctx context.Context, req *ps.GetAnomalyRequest) (*ps.Anomaly, error) {
	s.GetAnomalyFnInvoked = true
	return s.GetAnomalyFn(ctx, req)
}

func (s *QueryInsightsService) ListQuerySamples(ctx context.Context, req *ps.ListQuerySamplesRequest, opts ...ps.ListOption) ([]*ps.QuerySample, error) {
	s.ListQuerySamplesFnInvoked = true
	return s.ListQuerySamplesFn(ctx, req, opts...)
}

func (s *QueryInsightsService) ListErrors(ctx context.Context, req *ps.ListQueryInsightsErrorsRequest, opts ...ps.ListOption) ([]*ps.QueryInsightError, error) {
	s.ListErrorsFnInvoked = true
	return s.ListErrorsFn(ctx, req, opts...)
}

func (s *QueryInsightsService) ListAnomalies(ctx context.Context, req *ps.ListAnomaliesRequest, opts ...ps.ListOption) ([]*ps.Anomaly, error) {
	s.ListAnomaliesFnInvoked = true
	return s.ListAnomaliesFn(ctx, req, opts...)
}

func (s *QueryInsightsService) ListTags(ctx context.Context, req *ps.ListQueryTagsRequest, opts ...ps.ListOption) ([]*ps.QueryTag, error) {
	s.ListTagsFnInvoked = true
	return s.ListTagsFn(ctx, req, opts...)
}

func (s *QueryInsightsService) GetTag(ctx context.Context, req *ps.GetQueryTagRequest, opts ...ps.ListOption) (*ps.QueryTag, error) {
	s.GetTagFnInvoked = true
	return s.GetTagFn(ctx, req, opts...)
}

func (s *QueryInsightsService) ListTagSummaries(ctx context.Context, req *ps.ListTagSummariesRequest, opts ...ps.ListOption) ([]*ps.TagSummary, error) {
	s.ListTagSummariesFnInvoked = true
	return s.ListTagSummariesFn(ctx, req, opts...)
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
