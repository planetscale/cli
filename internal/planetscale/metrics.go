package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

// MetricsService provides access to branch time-series and instant metrics.
type MetricsService interface {
	GetSeries(context.Context, *GetMetricSeriesRequest) (*MetricSeries, error)
	GetInstant(context.Context, *GetInstantMetricsRequest) (*InstantMetrics, error)
	GetQuerySeries(context.Context, *GetQueryMetricSeriesRequest) (*MetricSeries, error)
	GetTables(context.Context, *GetBranchMetricsRequest) (json.RawMessage, error)
	GetKeyspaceTables(context.Context, *GetBranchMetricsRequest) (json.RawMessage, error)
	GetTabletSeries(context.Context, *GetTabletMetricSeriesRequest) (*MetricSeries, error)
	GetInstantTablets(context.Context, *GetInstantTabletMetricsRequest) (*InstantMetrics, error)
	GetTagSeries(context.Context, *GetTagMetricSeriesRequest) (*MetricSeries, error)
}

type metricsService struct {
	client *Client
}

var _ MetricsService = &metricsService{}

// MetricSeries is a collection of sampled time series over a common range.
type MetricSeries struct {
	Type      string        `json:"type"`
	StartDate time.Time     `json:"start_date"`
	EndDate   time.Time     `json:"end_date"`
	Interval  int           `json:"interval"`
	Series    []*TimeSeries `json:"series"`
}

// TimeSeries contains samples for one metric and set of dimensions.
type TimeSeries struct {
	Type   string            `json:"type"`
	Metric string            `json:"metric"`
	Label  string            `json:"label"`
	Labels map[string]string `json:"labels"`
	Points [][]float64       `json:"points"`
}

// InstantMetrics contains current metric values grouped by their dimensions.
type InstantMetrics struct {
	Type    string           `json:"type"`
	Branch  map[string]any   `json:"branch"`
	Metrics []*InstantMetric `json:"metrics"`
}

// InstantMetric contains the current values for one metric.
type InstantMetric struct {
	Metric string           `json:"metric"`
	Label  string           `json:"label"`
	Values []map[string]any `json:"values"`
}

// GetMetricSeriesRequest describes a branch time-series metrics query.
type GetMetricSeriesRequest struct {
	Organization string
	Database     string
	Branch       string
	Metrics      []string
	Period       string
	From         string
	To           string
	Steps        int
	TabletType   string
	Keyspace     string
	Shard        string
	Role         string
	Container    string
	Pod          string
	Pods         []string
	QueryIDs     []string
	Fingerprint  string
	BudgetID     string
	RuleID       string
	Search       string
}

// GetInstantMetricsRequest describes a branch instant metrics query.
type GetInstantMetricsRequest struct {
	Organization string
	Database     string
	Branch       string
	Metrics      []string
	Role         string
	Shard        string
	Container    string
	Pod          string
}

type GetBranchMetricsRequest struct {
	Organization string
	Database     string
	Branch       string
}

type GetQueryMetricSeriesRequest struct {
	Organization string
	Database     string
	Branch       string
	Metrics      []string
	QueryIDs     []string
	Fingerprint  string
	Keyspace     string
	Period       string
	From         string
	To           string
	Steps        int
	TabletType   string
	BudgetID     string
	RuleID       string
	Search       string
}

type GetTabletMetricSeriesRequest struct {
	Organization string
	Database     string
	Branch       string
	Metrics      []string
	Period       string
	From         string
	To           string
	Steps        int
	Keyspace     string
	Shard        string
	Pod          string
	Workflow     string
}

type GetInstantTabletMetricsRequest struct {
	Organization string
	Database     string
	Branch       string
	Metrics      []string
	Keyspace     string
	Shard        string
}

type GetTagMetricSeriesRequest struct {
	Organization string
	Database     string
	Branch       string
	Metrics      []string
	TagSets      []string
	Period       string
	From         string
	To           string
	Steps        int
	TabletType   string
	BudgetID     string
	RuleID       string
	Search       string
}

func (s *metricsService) GetSeries(ctx context.Context, getReq *GetMetricSeriesRequest) (*MetricSeries, error) {
	query := url.Values{}
	addQueryValues(query, "metrics[]", getReq.Metrics)
	setQueryValue(query, "period", getReq.Period)
	setQueryValue(query, "from", getReq.From)
	setQueryValue(query, "to", getReq.To)
	if getReq.Steps > 0 {
		query.Set("steps", strconv.Itoa(getReq.Steps))
	}
	setQueryValue(query, "tablet_type", getReq.TabletType)
	setQueryValue(query, "keyspace", getReq.Keyspace)
	setQueryValue(query, "shard", getReq.Shard)
	setQueryValue(query, "role", getReq.Role)
	setQueryValue(query, "container", getReq.Container)
	setQueryValue(query, "pod", getReq.Pod)
	addQueryValues(query, "pods[]", getReq.Pods)
	addQueryValues(query, "query_ids[]", getReq.QueryIDs)
	setQueryValue(query, "fingerprint", getReq.Fingerprint)
	setQueryValue(query, "budget_id", getReq.BudgetID)
	setQueryValue(query, "rule_id", getReq.RuleID)
	setQueryValue(query, "q", getReq.Search)

	req, err := s.client.newRequest(http.MethodGet, metricsAPIPath(getReq.Organization, getReq.Database, getReq.Branch), nil, WithQueryParams(query))
	if err != nil {
		return nil, fmt.Errorf("error creating request for branch metrics: %w", err)
	}

	series := &MetricSeries{}
	if err := s.client.do(ctx, req, series); err != nil {
		return nil, err
	}

	return series, nil
}

func (s *metricsService) GetInstant(ctx context.Context, getReq *GetInstantMetricsRequest) (*InstantMetrics, error) {
	query := url.Values{}
	addQueryValues(query, "metrics[]", getReq.Metrics)
	setQueryValue(query, "role", getReq.Role)
	setQueryValue(query, "shard", getReq.Shard)
	setQueryValue(query, "container", getReq.Container)
	setQueryValue(query, "pod", getReq.Pod)

	req, err := s.client.newRequest(http.MethodGet, path.Join(metricsAPIPath(getReq.Organization, getReq.Database, getReq.Branch), "instant"), nil, WithQueryParams(query))
	if err != nil {
		return nil, fmt.Errorf("error creating request for instant branch metrics: %w", err)
	}

	metrics := &InstantMetrics{}
	if err := s.client.do(ctx, req, metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

func (s *metricsService) GetQuerySeries(ctx context.Context, getReq *GetQueryMetricSeriesRequest) (*MetricSeries, error) {
	query := url.Values{}
	addQueryValues(query, "metrics[]", getReq.Metrics)
	addQueryValues(query, "query_ids[]", getReq.QueryIDs)
	setQueryValue(query, "fingerprint", getReq.Fingerprint)
	setQueryValue(query, "keyspace", getReq.Keyspace)
	setSeriesRange(query, getReq.Period, getReq.From, getReq.To, getReq.Steps)
	setQueryValue(query, "tablet_type", getReq.TabletType)
	setQueryValue(query, "budget_id", getReq.BudgetID)
	setQueryValue(query, "rule_id", getReq.RuleID)
	setQueryValue(query, "q", getReq.Search)

	return s.getSpecializedSeries(ctx, getReq.Organization, getReq.Database, getReq.Branch, "query", query)
}

func (s *metricsService) GetTables(ctx context.Context, getReq *GetBranchMetricsRequest) (json.RawMessage, error) {
	return s.getRawMetrics(ctx, getReq, "tables")
}

func (s *metricsService) GetKeyspaceTables(ctx context.Context, getReq *GetBranchMetricsRequest) (json.RawMessage, error) {
	return s.getRawMetrics(ctx, getReq, "keyspace-tables")
}

func (s *metricsService) GetTabletSeries(ctx context.Context, getReq *GetTabletMetricSeriesRequest) (*MetricSeries, error) {
	query := url.Values{}
	addQueryValues(query, "metrics[]", getReq.Metrics)
	setSeriesRange(query, getReq.Period, getReq.From, getReq.To, getReq.Steps)
	setQueryValue(query, "keyspace", getReq.Keyspace)
	setQueryValue(query, "shard", getReq.Shard)
	setQueryValue(query, "pod", getReq.Pod)
	setQueryValue(query, "workflow", getReq.Workflow)

	return s.getSpecializedSeries(ctx, getReq.Organization, getReq.Database, getReq.Branch, "tablets", query)
}

func (s *metricsService) GetInstantTablets(ctx context.Context, getReq *GetInstantTabletMetricsRequest) (*InstantMetrics, error) {
	query := url.Values{}
	addQueryValues(query, "metrics[]", getReq.Metrics)
	setQueryValue(query, "keyspace", getReq.Keyspace)
	setQueryValue(query, "shard", getReq.Shard)

	req, err := s.client.newRequest(http.MethodGet, path.Join(metricsAPIPath(getReq.Organization, getReq.Database, getReq.Branch), "tablets-instant"), nil, WithQueryParams(query))
	if err != nil {
		return nil, fmt.Errorf("error creating request for instant tablet metrics: %w", err)
	}

	metrics := &InstantMetrics{}
	if err := s.client.do(ctx, req, metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

func (s *metricsService) GetTagSeries(ctx context.Context, getReq *GetTagMetricSeriesRequest) (*MetricSeries, error) {
	query := url.Values{}
	addQueryValues(query, "metrics[]", getReq.Metrics)
	addQueryValues(query, "tag_sets[]", getReq.TagSets)
	setSeriesRange(query, getReq.Period, getReq.From, getReq.To, getReq.Steps)
	setQueryValue(query, "tablet_type", getReq.TabletType)
	setQueryValue(query, "budget_id", getReq.BudgetID)
	setQueryValue(query, "rule_id", getReq.RuleID)
	setQueryValue(query, "q", getReq.Search)

	return s.getSpecializedSeries(ctx, getReq.Organization, getReq.Database, getReq.Branch, "tag", query)
}

func (s *metricsService) getSpecializedSeries(ctx context.Context, org, database, branch, endpoint string, query url.Values) (*MetricSeries, error) {
	req, err := s.client.newRequest(http.MethodGet, path.Join(metricsAPIPath(org, database, branch), endpoint), nil, WithQueryParams(query))
	if err != nil {
		return nil, fmt.Errorf("error creating request for branch %s metrics: %w", endpoint, err)
	}

	series := &MetricSeries{}
	if err := s.client.do(ctx, req, series); err != nil {
		return nil, err
	}

	return series, nil
}

func (s *metricsService) getRawMetrics(ctx context.Context, getReq *GetBranchMetricsRequest, endpoint string) (json.RawMessage, error) {
	req, err := s.client.newRequest(http.MethodGet, path.Join(metricsAPIPath(getReq.Organization, getReq.Database, getReq.Branch), endpoint), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for branch %s metrics: %w", endpoint, err)
	}

	var response json.RawMessage
	if err := s.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return response, nil
}

func metricsAPIPath(org, db, branch string) string {
	return path.Join("v1/organizations", org, "databases", db, "branches", branch, "metrics")
}

func setSeriesRange(query url.Values, period, from, to string, steps int) {
	setQueryValue(query, "period", period)
	setQueryValue(query, "from", from)
	setQueryValue(query, "to", to)
	if steps > 0 {
		query.Set("steps", strconv.Itoa(steps))
	}
}

func setQueryValue(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}

func addQueryValues(query url.Values, key string, values []string) {
	for _, value := range values {
		if value != "" {
			query.Add(key, value)
		}
	}
}
