package planetscale

import (
	"context"
	"net/http"
	"path"
	"time"
)

var _ QueryInsightsService = &queryInsightsService{}

// QueryInsightsService is an interface for communicating with the PlanetScale
// Query Insights API: aggregated query statistics, query errors, and detected
// anomalies for a database branch.
type QueryInsightsService interface {
	ListQueries(context.Context, *ListQueryInsightsRequest, ...ListOption) ([]*QueryInsight, error)
	ListErrors(context.Context, *ListQueryInsightsErrorsRequest, ...ListOption) ([]*QueryInsightError, error)
	ListAnomalies(context.Context, *ListAnomaliesRequest, ...ListOption) ([]*Anomaly, error)
}

// QueryInsight is an aggregated statistics record for a normalized query
// pattern on a branch.
type QueryInsight struct {
	ID                     string       `json:"id"`
	Fingerprint            string       `json:"fingerprint"`
	NormalizedSQL          string       `json:"normalized_sql"`
	StatementType          string       `json:"statement_type"`
	Keyspace               string       `json:"keyspace"`
	Tables                 []string     `json:"tables"`
	IndexUsages            []IndexUsage `json:"index_usages"`
	QueryCount             int64        `json:"query_count"`
	ErrorCount             int64        `json:"error_count"`
	LastRunAt              time.Time    `json:"last_run_at"`
	TimePerQuery           float64      `json:"time_per_query"`
	P50Latency             float64      `json:"p50_latency"`
	P99Latency             float64      `json:"p99_latency"`
	MaxLatency             float64      `json:"max_latency"`
	SumRowsRead            int64        `json:"sum_rows_read"`
	SumRowsReturned        int64        `json:"sum_rows_returned"`
	SumRowsAffected        int64        `json:"sum_rows_affected"`
	RowsReadPerReturned    float64      `json:"rows_read_per_returned"`
	SumTotalDurationMillis float64      `json:"sum_total_duration_millis"`
	SumTotalDurationPct    float64      `json:"sum_total_duration_percent"`
	SumCPUDurationMillis   float64      `json:"sum_cpu_duration_millis"`
	SumIODurationMillis    float64      `json:"sum_io_duration_millis"`
	BlockCacheHitRatio     float64      `json:"block_cache_hit_ratio"`
	Multishard             bool         `json:"multishard"`
}

// IndexUsage records how often an index served a query pattern.
type IndexUsage struct {
	Name    string  `json:"name"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

// QueryInsightError is an aggregated error record for a query pattern on a
// branch.
type QueryInsightError struct {
	ID                  string    `json:"id"`
	ErrorFingerprint    string    `json:"error_fingerprint"`
	StartedAt           time.Time `json:"started_at"`
	TotalDurationMillis float64   `json:"total_duration_millis"`
	TimePerQuery        float64   `json:"time_per_query"`
	ErrorCount          int64     `json:"error_count"`
	ErrorMessage        string    `json:"error_message"`
}

// Anomaly is a detected resource-usage anomaly on a branch's primary.
type Anomaly struct {
	ID                 string    `json:"id"`
	PeriodStart        time.Time `json:"period_start"`
	PeriodEnd          time.Time `json:"period_end"`
	MinutesInViolation int64     `json:"minutes_in_violation"`
	Active             bool      `json:"active"`
	Duration           float64   `json:"duration"`
	MetricsStart       time.Time `json:"metrics_start"`
	MetricsEnd         time.Time `json:"metrics_end"`
}

// ListQueryInsightsRequest is the request for listing query statistics for a branch.
type ListQueryInsightsRequest struct {
	Organization string
	Database     string
	Branch       string
}

// ListQueryInsightsErrorsRequest is the request for listing query errors for a branch.
type ListQueryInsightsErrorsRequest struct {
	Organization string
	Database     string
	Branch       string
}

// ListAnomaliesRequest is the request for listing anomalies for a branch.
type ListAnomaliesRequest struct {
	Organization string
	Database     string
	Branch       string
}

// WithSort returns a ListOption that sets the "sort" and "dir" URL parameters.
func WithSort(sort, dir string) ListOption {
	return func(opt *ListOptions) error {
		if sort != "" {
			opt.URLValues.Set("sort", sort)
		}
		if dir != "" {
			opt.URLValues.Set("dir", dir)
		}
		return nil
	}
}

// WithPeriod returns a ListOption that sets the "period" URL parameter
// (e.g. "1h", "24h").
func WithPeriod(period string) ListOption {
	return func(opt *ListOptions) error {
		if period != "" {
			opt.URLValues.Set("period", period)
		}
		return nil
	}
}

type queryInsightsService struct {
	client *Client
}

type queryInsightsResponse struct {
	Insights []*QueryInsight `json:"data"`
}

type queryInsightErrorsResponse struct {
	Errors []*QueryInsightError `json:"data"`
}

type anomaliesResponse struct {
	Anomalies []*Anomaly `json:"data"`
}

func (s *queryInsightsService) ListQueries(ctx context.Context, request *ListQueryInsightsRequest, opts ...ListOption) ([]*QueryInsight, error) {
	listOpts := defaultListOptions(opts...)

	req, err := s.client.newRequest(http.MethodGet, insightsAPIPath(request.Organization, request.Database, request.Branch), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	resp := &queryInsightsResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Insights, nil
}

func (s *queryInsightsService) ListErrors(ctx context.Context, request *ListQueryInsightsErrorsRequest, opts ...ListOption) ([]*QueryInsightError, error) {
	listOpts := defaultListOptions(opts...)

	req, err := s.client.newRequest(http.MethodGet, path.Join(insightsAPIPath(request.Organization, request.Database, request.Branch), "errors"), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	resp := &queryInsightErrorsResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Errors, nil
}

func (s *queryInsightsService) ListAnomalies(ctx context.Context, request *ListAnomaliesRequest, opts ...ListOption) ([]*Anomaly, error) {
	listOpts := defaultListOptions(opts...)

	req, err := s.client.newRequest(http.MethodGet, path.Join(insightsAPIPath(request.Organization, request.Database, request.Branch), "anomalies"), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	resp := &anomaliesResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Anomalies, nil
}

func insightsAPIPath(org, db, branch string) string {
	return path.Join("v1/organizations", org, "databases", db, "branches", branch, "insights")
}
