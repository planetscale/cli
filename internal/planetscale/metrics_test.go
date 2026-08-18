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
