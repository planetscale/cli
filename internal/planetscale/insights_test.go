package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestQueryInsights_ListQueries(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/insights")
		c.Assert(r.URL.Query().Get("sort"), qt.Equals, "totalTime")
		c.Assert(r.URL.Query().Get("dir"), qt.Equals, "desc")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "5")
		c.Assert(r.URL.Query().Get("period"), qt.Equals, "24h")

		out := `{
			"type": "list",
			"current_page": 1,
			"data": [{
				"id": "f5a67b04ee9f",
				"type": "HourlyBranchQuery",
				"query_count": 8,
				"error_count": 1,
				"tables": ["users"],
				"index_usages": [{"name": "public.users.users_pkey", "count": 5757, "percent": 100.0}],
				"sum_rows_read": 100,
				"sum_rows_returned": 25,
				"sum_rows_affected": 0,
				"rows_read_per_returned": 4.0,
				"sum_total_duration_millis": 14.595,
				"sum_total_duration_percent": 57.57,
				"sum_cpu_duration_millis": 1.5,
				"sum_io_duration_millis": 0.5,
				"last_run_at": "2026-07-22T23:22:14.000Z",
				"time_per_query": 1.824375,
				"p50_latency": 1.75,
				"p99_latency": 2.32,
				"max_latency": 2.32,
				"block_cache_hit_ratio": 0.9,
				"fingerprint": "b129e8fa",
				"statement_type": "SELECT",
				"keyspace": "mydb",
				"normalized_sql": "select * from users where id = ?",
				"multishard": false
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	insights, err := client.QueryInsights.ListQueries(context.Background(), &ListQueryInsightsRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       "main",
	}, WithPerPage(5), WithSort("totalTime", "desc"), WithPeriod("24h"))

	c.Assert(err, qt.IsNil)
	c.Assert(len(insights), qt.Equals, 1)
	c.Assert(insights[0].ID, qt.Equals, "f5a67b04ee9f")
	c.Assert(insights[0].QueryCount, qt.Equals, int64(8))
	c.Assert(insights[0].ErrorCount, qt.Equals, int64(1))
	c.Assert(insights[0].NormalizedSQL, qt.Equals, "select * from users where id = ?")
	c.Assert(insights[0].StatementType, qt.Equals, "SELECT")
	c.Assert(insights[0].Keyspace, qt.Equals, "mydb")
	c.Assert(insights[0].SumRowsRead, qt.Equals, int64(100))
	c.Assert(insights[0].RowsReadPerReturned, qt.Equals, 4.0)
	c.Assert(insights[0].SumTotalDurationMillis, qt.Equals, 14.595)
	c.Assert(insights[0].P99Latency, qt.Equals, 2.32)
	c.Assert(insights[0].Tables, qt.DeepEquals, []string{"users"})
	c.Assert(insights[0].IndexUsages, qt.DeepEquals, []IndexUsage{{Name: "public.users.users_pkey", Count: 5757, Percent: 100.0}})
}

func TestQueryInsights_ListErrors(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/insights/errors")

		out := `{
			"type": "list",
			"current_page": 1,
			"data": [{
				"id": "5d4485f99812",
				"type": "BranchQuery",
				"error_fingerprint": "5d4485f9981294d1",
				"started_at": "2026-07-22T14:05:50.000Z",
				"total_duration_millis": 12.5,
				"time_per_query": 3.1,
				"error_count": 4,
				"error_message": "relation \"widgets\" does not exist"
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	errs, err := client.QueryInsights.ListErrors(context.Background(), &ListQueryInsightsErrorsRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       "main",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(len(errs), qt.Equals, 1)
	c.Assert(errs[0].ID, qt.Equals, "5d4485f99812")
	c.Assert(errs[0].ErrorCount, qt.Equals, int64(4))
	c.Assert(errs[0].ErrorMessage, qt.Equals, `relation "widgets" does not exist`)
	c.Assert(errs[0].TotalDurationMillis, qt.Equals, 12.5)
}

func TestQueryInsights_ListAnomalies(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/insights/anomalies")

		out := `{
			"type": "list",
			"current_page": 1,
			"data": [{
				"id": "anomaly-123",
				"period_start": "2026-07-22T14:00:00.000Z",
				"period_end": "2026-07-22T14:30:00.000Z",
				"minutes_in_violation": 12,
				"active": false,
				"duration": 1800.0,
				"metrics_start": "2026-07-22T13:30:00.000Z",
				"metrics_end": "2026-07-22T15:00:00.000Z"
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	anomalies, err := client.QueryInsights.ListAnomalies(context.Background(), &ListAnomaliesRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       "main",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(len(anomalies), qt.Equals, 1)
	c.Assert(anomalies[0].ID, qt.Equals, "anomaly-123")
	c.Assert(anomalies[0].MinutesInViolation, qt.Equals, int64(12))
	c.Assert(anomalies[0].Active, qt.Equals, false)
	c.Assert(anomalies[0].Duration, qt.Equals, 1800.0)
}
