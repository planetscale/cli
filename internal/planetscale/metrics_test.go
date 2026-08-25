package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestMetrics_GetSeries(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/metrics")
		query := r.URL.Query()
		c.Assert(query["metrics[]"], qt.DeepEquals, []string{"queries", "latency_p99"})
		c.Assert(query.Get("period"), qt.Equals, "1h")
		c.Assert(query.Get("steps"), qt.Equals, "60")
		c.Assert(query.Get("tablet_type"), qt.Equals, "replica")
		c.Assert(query["pods[]"], qt.DeepEquals, []string{"pod-1", "pod-2"})
		c.Assert(query["query_ids[]"], qt.DeepEquals, []string{"abc-main"})
		c.Assert(query.Get("q"), qt.Equals, "checkout")

		_, err := w.Write([]byte(`{
			"type":"MetricSeries",
			"start_date":"2026-08-18T16:00:00Z",
			"end_date":"2026-08-18T17:00:00Z",
			"interval":60,
			"series":[{
				"type":"TimeSeries",
				"metric":"queries",
				"label":"Queries",
				"labels":{},
				"points":[[1787068800,912],[1787068860,1048]]
			}]
		}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	series, err := client.Metrics.GetSeries(context.Background(), &GetMetricSeriesRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "main",
		Metrics:      []string{"queries", "latency_p99"},
		Period:       "1h",
		Steps:        60,
		TabletType:   "replica",
		Pods:         []string{"pod-1", "pod-2"},
		QueryIDs:     []string{"abc-main"},
		Search:       "checkout",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(series.Type, qt.Equals, "MetricSeries")
	c.Assert(series.Interval, qt.Equals, 60)
	c.Assert(series.Series, qt.HasLen, 1)
	c.Assert(series.Series[0].Points[1][1], qt.Equals, 1048.0)
}

func TestMetrics_GetInstant(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/metrics/instant")
		query := r.URL.Query()
		c.Assert(query["metrics[]"], qt.DeepEquals, []string{"planetscale_volume_usage_percentage"})
		c.Assert(query.Get("role"), qt.Equals, "primary")

		_, err := w.Write([]byte(`{
			"type":"InstantMetrics",
			"branch":{"id":"branch-id","name":"main"},
			"metrics":[{
				"metric":"planetscale_volume_usage_percentage",
				"label":"volume_usage",
				"values":[{"pod":"postgres-0","role":"primary","value":71.4}]
			}]
		}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	metrics, err := client.Metrics.GetInstant(context.Background(), &GetInstantMetricsRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "main",
		Metrics:      []string{"planetscale_volume_usage_percentage"},
		Role:         "primary",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(metrics.Type, qt.Equals, "InstantMetrics")
	c.Assert(metrics.Branch["name"], qt.Equals, "main")
	c.Assert(metrics.Metrics, qt.HasLen, 1)
	c.Assert(metrics.Metrics[0].Values[0]["value"], qt.Equals, 71.4)
}

func TestMetrics_GetQuerySeries(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/metrics/query")
		query := r.URL.Query()
		c.Assert(query.Get("metrics"), qt.Equals, "queries,latency_p99")
		c.Assert(query.Get("query_ids"), qt.Equals, "query-1,query-2")
		c.Assert(query.Get("fingerprint"), qt.Equals, "select-from-users")
		c.Assert(query.Get("keyspace"), qt.Equals, "commerce")
		c.Assert(query.Get("period"), qt.Equals, "1h")
		c.Assert(query.Get("steps"), qt.Equals, "60")
		c.Assert(query.Get("tablet_type"), qt.Equals, "replica")
		c.Assert(query.Get("budget_id"), qt.Equals, "budget-1")
		c.Assert(query.Get("rule_id"), qt.Equals, "rule-1")
		c.Assert(query.Get("q"), qt.Equals, "checkout")
		writeMetricSeriesResponse(c, w)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	series, err := client.Metrics.GetQuerySeries(context.Background(), &GetQueryMetricSeriesRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "main",
		Metrics:      []string{"queries", "latency_p99"},
		QueryIDs:     []string{"query-1", "query-2"},
		Fingerprint:  "select-from-users",
		Keyspace:     "commerce",
		Period:       "1h",
		Steps:        60,
		TabletType:   "replica",
		BudgetID:     "budget-1",
		RuleID:       "rule-1",
		Search:       "checkout",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(series.Series, qt.HasLen, 1)
}

func TestMetrics_GetStorageMetrics(t *testing.T) {
	tests := []struct {
		name string
		path string
		get  func(context.Context, MetricsService, *GetBranchMetricsRequest) ([]byte, error)
	}{
		{
			name: "tables",
			path: "/v1/organizations/my-org/databases/my-db/branches/main/metrics/tables",
			get: func(ctx context.Context, service MetricsService, req *GetBranchMetricsRequest) ([]byte, error) {
				return service.GetTables(ctx, req)
			},
		},
		{
			name: "keyspace tables",
			path: "/v1/organizations/my-org/databases/my-db/branches/main/metrics/keyspace-tables",
			get: func(ctx context.Context, service MetricsService, req *GetBranchMetricsRequest) ([]byte, error) {
				return service.GetKeyspaceTables(ctx, req)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := qt.New(t)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Assert(r.URL.Path, qt.Equals, test.path)
				_, err := w.Write([]byte(`{"commerce":{"users":1048576}}`))
				c.Assert(err, qt.IsNil)
			}))
			defer ts.Close()

			client, err := NewClient(WithBaseURL(ts.URL))
			c.Assert(err, qt.IsNil)
			response, err := test.get(context.Background(), client.Metrics, &GetBranchMetricsRequest{
				Organization: "my-org",
				Database:     "my-db",
				Branch:       "main",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(response, qt.DeepEquals, []byte(`{"commerce":{"users":1048576}}`))
		})
	}
}

func TestMetrics_GetTabletSeries(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/metrics/tablets")
		query := r.URL.Query()
		c.Assert(query.Get("metrics"), qt.Equals, "replication_lag,pod_cpu_usage")
		c.Assert(query.Get("from"), qt.Equals, "2026-08-18T16:00:00Z")
		c.Assert(query.Get("to"), qt.Equals, "2026-08-18T17:00:00Z")
		c.Assert(query.Get("keyspace"), qt.Equals, "commerce")
		c.Assert(query.Get("shard"), qt.Equals, "-80")
		c.Assert(query.Get("pod"), qt.Equals, "zone-a-0")
		c.Assert(query.Get("workflow"), qt.Equals, "move-tables")
		writeMetricSeriesResponse(c, w)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	_, err = client.Metrics.GetTabletSeries(context.Background(), &GetTabletMetricSeriesRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "main",
		Metrics:      []string{"replication_lag", "pod_cpu_usage"},
		From:         "2026-08-18T16:00:00Z",
		To:           "2026-08-18T17:00:00Z",
		Keyspace:     "commerce",
		Shard:        "-80",
		Pod:          "zone-a-0",
		Workflow:     "move-tables",
	})
	c.Assert(err, qt.IsNil)
}

func TestMetrics_GetInstantTablets(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/metrics/tablets-instant")
		query := r.URL.Query()
		c.Assert(query.Get("metrics"), qt.Equals, "replication_lag,primary_cpu_usage")
		c.Assert(query.Get("keyspace"), qt.Equals, "commerce")
		c.Assert(query.Get("shard"), qt.Equals, "-80")
		_, err := w.Write([]byte(`{
			"type":"InstantMetrics",
			"branch":{"name":"main"},
			"metrics":[{"metric":"replication_lag","label":"Replication lag","values":[{"value":0.2}]}]
		}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	metrics, err := client.Metrics.GetInstantTablets(context.Background(), &GetInstantTabletMetricsRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "main",
		Metrics:      []string{"replication_lag", "primary_cpu_usage"},
		Keyspace:     "commerce",
		Shard:        "-80",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(metrics.Metrics, qt.HasLen, 1)
}

func TestMetrics_GetTagSeries(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/metrics/tag")
		query := r.URL.Query()
		c.Assert(query.Get("metrics"), qt.Equals, "queries,latency_p99")
		c.Assert(query.Get("tag_sets"), qt.Equals, "service=checkout,region=us-east")
		c.Assert(query.Get("period"), qt.Equals, "1d")
		c.Assert(query.Get("tablet_type"), qt.Equals, "primary")
		c.Assert(query.Get("budget_id"), qt.Equals, "budget-1")
		c.Assert(query.Get("rule_id"), qt.Equals, "rule-1")
		c.Assert(query.Get("q"), qt.Equals, "checkout")
		writeMetricSeriesResponse(c, w)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	_, err = client.Metrics.GetTagSeries(context.Background(), &GetTagMetricSeriesRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "main",
		Metrics:      []string{"queries", "latency_p99"},
		TagSets:      []string{"service=checkout", "region=us-east"},
		Period:       "1d",
		TabletType:   "primary",
		BudgetID:     "budget-1",
		RuleID:       "rule-1",
		Search:       "checkout",
	})
	c.Assert(err, qt.IsNil)
}

func writeMetricSeriesResponse(c *qt.C, w http.ResponseWriter) {
	_, err := w.Write([]byte(`{
		"type":"MetricSeries",
		"start_date":"2026-08-18T16:00:00Z",
		"end_date":"2026-08-18T17:00:00Z",
		"interval":60,
		"series":[{"type":"TimeSeries","metric":"queries","label":"Queries","labels":{},"points":[[1787068800,912]]}]
	}`))
	c.Assert(err, qt.IsNil)
}
