package planetscale

import (
	"context"
	"net/http"
	"path"
)

// QueryTag is a tag key observed on branch queries (sqlcommenter / system).
// ID is the dimension key used by the API (e.g. "Sapp", "Busername").
// Name is the friendly key users see in the app (e.g. "app", "username").
// Source is "sql" or "system".
type QueryTag struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Source     string          `json:"source"`
	QueryCount int64           `json:"query_count"`
	Values     []QueryTagValue `json:"values"`
}

// QueryTagValue is one observed value for a tag key.
type QueryTagValue struct {
	Name       string `json:"name"`
	QueryCount int64  `json:"query_count"`
	Kind       string `json:"kind"`
}

// TagSummary is query statistics grouped by one or more tag dimensions.
type TagSummary struct {
	Dimensions             map[string]string `json:"dimensions"`
	QueryCount             int64             `json:"query_count"`
	ErrorCount             int64             `json:"error_count"`
	Tables                 []string          `json:"tables"`
	SumRowsRead            int64             `json:"sum_rows_read"`
	SumRowsReturned        int64             `json:"sum_rows_returned"`
	SumRowsAffected        int64             `json:"sum_rows_affected"`
	RowsReadPerReturned    float64           `json:"rows_read_per_returned"`
	SumTotalDurationMillis float64           `json:"sum_total_duration_millis"`
	SumTotalDurationPct    float64           `json:"sum_total_duration_percent"`
	SumCPUDurationMillis   float64           `json:"sum_cpu_duration_millis"`
	SumIODurationMillis    float64           `json:"sum_io_duration_millis"`
	LastRunAt              string            `json:"last_run_at"`
	TimePerQuery           float64           `json:"time_per_query"`
	P50Latency             float64           `json:"p50_latency"`
	P99Latency             float64           `json:"p99_latency"`
	MaxLatency             float64           `json:"max_latency"`
}

// ListQueryTagsRequest lists tag keys for a branch.
type ListQueryTagsRequest struct {
	Organization string
	Database     string
	Branch       string
}

// GetQueryTagRequest retrieves one tag key and its values.
// Tag is the dimension id (e.g. "Sapp"), not the display name.
type GetQueryTagRequest struct {
	Organization string
	Database     string
	Branch       string
	Tag          string
}

// ListTagSummariesRequest lists query stats grouped by tag dimensions.
// Tags are dimension ids from the tags list endpoint (e.g. "Sapp").
type ListTagSummariesRequest struct {
	Organization string
	Database     string
	Branch       string
	Tags         []string
}

type queryTagsResponse struct {
	Data []*QueryTag `json:"data"`
}

type tagSummariesResponse struct {
	Data []*TagSummary `json:"data"`
}

// WithTags sets the tags[] query parameters (dimension ids for summaries).
func WithTags(tags []string) ListOption {
	return func(opt *ListOptions) error {
		for _, tag := range tags {
			if tag != "" {
				opt.URLValues.Add("tags[]", tag)
			}
		}
		return nil
	}
}

// WithFingerprint sets the fingerprint query parameter.
func WithFingerprint(fingerprint string) ListOption {
	return func(opt *ListOptions) error {
		if fingerprint != "" {
			opt.URLValues.Set("fingerprint", fingerprint)
		}
		return nil
	}
}

// WithKeyspace sets the keyspace query parameter.
func WithKeyspace(keyspace string) ListOption {
	return func(opt *ListOptions) error {
		if keyspace != "" {
			opt.URLValues.Set("keyspace", keyspace)
		}
		return nil
	}
}

func (s *queryInsightsService) ListTags(ctx context.Context, request *ListQueryTagsRequest, opts ...ListOption) ([]*QueryTag, error) {
	listOpts := defaultListOptions(opts...)

	req, err := s.client.newRequest(http.MethodGet, insightsTagsAPIPath(request.Organization, request.Database, request.Branch), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	resp := &queryTagsResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (s *queryInsightsService) GetTag(ctx context.Context, request *GetQueryTagRequest, opts ...ListOption) (*QueryTag, error) {
	listOpts := defaultListOptions(opts...)

	req, err := s.client.newRequest(http.MethodGet, insightsTagAPIPath(request.Organization, request.Database, request.Branch, request.Tag), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	tag := &QueryTag{}
	if err := s.client.do(ctx, req, &tag); err != nil {
		return nil, err
	}

	return tag, nil
}

func (s *queryInsightsService) ListTagSummaries(ctx context.Context, request *ListTagSummariesRequest, opts ...ListOption) ([]*TagSummary, error) {
	opts = append([]ListOption{WithTags(request.Tags)}, opts...)
	listOpts := defaultListOptions(opts...)

	req, err := s.client.newRequest(http.MethodGet, insightsTagSummariesAPIPath(request.Organization, request.Database, request.Branch), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	resp := &tagSummariesResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func insightsTagsAPIPath(org, db, branch string) string {
	return path.Join(insightsAPIPath(org, db, branch), "tags")
}

func insightsTagAPIPath(org, db, branch, tag string) string {
	return path.Join(insightsTagsAPIPath(org, db, branch), tag)
}

func insightsTagSummariesAPIPath(org, db, branch string) string {
	return path.Join(insightsTagsAPIPath(org, db, branch), "summaries")
}
