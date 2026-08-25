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

func TestQueryInsights_ListErrorQueries(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/insights/errors/b129e8fa")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "10")
		c.Assert(r.URL.Query().Get("period"), qt.Equals, "1h")

		w.WriteHeader(200)
		out := `{
			"type": "list",
			"data": [{
				"id": "exec-1",
				"fingerprint": "b129e8fa",
				"normalized_sql": "select * from users where id = ?",
				"keyspace": "main",
				"username": "app",
				"total_duration_millis": 2.5,
				"started_at": "2026-08-11T18:00:00.000Z",
				"error_message": "target: main.-.primary: vttablet: rpc error"
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	queries, err := client.QueryInsights.ListErrorQueries(context.Background(), &ListErrorQueriesRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       "main",
		Fingerprint:  "b129e8fa",
	}, WithPerPage(10), WithPeriod("1h"))

	c.Assert(err, qt.IsNil)
	c.Assert(queries, qt.HasLen, 1)
	c.Assert(queries[0].ID, qt.Equals, "exec-1")
	c.Assert(queries[0].Keyspace, qt.Equals, "main")
	c.Assert(queries[0].ErrorMessage, qt.Equals, "target: main.-.primary: vttablet: rpc error")
}

func TestQueryInsights_GetAnomaly(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/insights/anomalies/anomaly-123")

		w.WriteHeader(200)
		out := `{
			"id": "anomaly-123",
			"period_start": "2026-08-11T18:00:00.000Z",
			"period_end": "2026-08-11T18:30:00.000Z",
			"minutes_in_violation": 12,
			"active": false,
			"duration": 1800.0,
			"correlations": [{
				"id": "corr-1",
				"r": 0.94,
				"keyspace": "main",
				"fingerprint": "b129e8fa",
				"normalized_sql": "select * from users where id = ?",
				"tablet_type": "primary"
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	anomaly, err := client.QueryInsights.GetAnomaly(context.Background(), &GetAnomalyRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       "main",
		AnomalyID:    "anomaly-123",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(anomaly.ID, qt.Equals, "anomaly-123")
	c.Assert(anomaly.Correlations, qt.HasLen, 1)
	c.Assert(anomaly.Correlations[0].Fingerprint, qt.Equals, "b129e8fa")
	c.Assert(anomaly.Correlations[0].R, qt.Equals, 0.94)
}

func TestQueryInsights_ListQuerySamples(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/insights/b129e8fa")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "10")
		c.Assert(r.URL.Query().Get("period"), qt.Equals, "1h")
		c.Assert(r.URL.Query().Get("keyspace"), qt.Equals, "public")

		out := `{
			"type": "list",
			"data": [{
				"id": "exec-1",
				"fingerprint": "b129e8fa",
				"normalized_sql": "select * from users where id = ?",
				"username": "app",
				"rows_read": 1,
				"rows_returned": 1,
				"total_duration_millis": 2.5,
				"started_at": "2026-08-11T18:00:00.000Z",
				"error_message": "",
				"tags": [{"name": "Sapp", "value": "web"}, {"name": "Busername", "value": "alice"}]
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	samples, err := client.QueryInsights.ListQuerySamples(context.Background(), &ListQuerySamplesRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       "main",
		Fingerprint:  "b129e8fa",
	}, WithPerPage(10), WithPeriod("1h"), WithKeyspace("public"))

	c.Assert(err, qt.IsNil)
	c.Assert(samples, qt.HasLen, 1)
	c.Assert(samples[0].ID, qt.Equals, "exec-1")
	c.Assert(samples[0].Username, qt.Equals, "app")
	c.Assert(samples[0].TotalDurationMillis, qt.Equals, 2.5)
	c.Assert(samples[0].Tags, qt.DeepEquals, []QuerySampleTag{
		{Name: "Sapp", Value: "web"},
		{Name: "Busername", Value: "alice"},
	})
}

func TestQueryInsights_ListQueryTrafficBudgets(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/insights/b129e8fa/traffic/budgets")
		c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "10")
		c.Assert(r.URL.Query().Get("keyspace"), qt.Equals, "public")

		out := `{
			"type": "list",
			"data": [{
				"id": "budget-1",
				"name": "App queries",
				"mode": "enforce",
				"capacity": 100,
				"rate": 50,
				"burst": null,
				"concurrency": null,
				"warning_threshold": 80,
				"rules": [],
				"actor": {"id": "actor-1", "display_name": "User", "avatar_url": ""},
				"created_at": "2026-08-11T18:00:00.000Z",
				"updated_at": "2026-08-11T18:00:00.000Z"
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	budgets, err := client.QueryInsights.ListQueryTrafficBudgets(context.Background(), &ListQueryTrafficBudgetsRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       "main",
		Fingerprint:  "b129e8fa",
	}, WithPage(2), WithPerPage(10), WithKeyspace("public"))

	c.Assert(err, qt.IsNil)
	c.Assert(budgets, qt.HasLen, 1)
	c.Assert(budgets[0].ID, qt.Equals, "budget-1")
	c.Assert(budgets[0].Name, qt.Equals, "App queries")
	c.Assert(*budgets[0].Capacity, qt.Equals, 100)
}

func TestQueryInsights_GetQuery(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/insights/queries/exec-1")

		w.WriteHeader(200)
		out := `{
			"id": "exec-1",
			"password": {"id": "password-1"},
			"tags": [{"name": "Sapp", "value": "web"}],
			"fingerprint": "b129e8fa",
			"started_at": "2026-08-25T18:00:00.000Z",
			"statement_type": "SELECT",
			"keyspace": "main",
			"tables": ["users"],
			"username": "app",
			"remote_address": "192.0.2.10",
			"shard_queries": 1,
			"rows_read": 2,
			"rows_affected": 0,
			"rows_returned": 1,
			"total_duration_millis": 2.5,
			"error_message": "",
			"normalized_sql": "select * from users where id = ?",
			"syntax_highlighted_sql": "select * from users where id = ?",
			"created_at": "2026-08-25T18:00:01.000Z",
			"updated_at": "2026-08-25T18:00:01.000Z",
			"explainable": true,
			"truncated": false
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	query, err := client.QueryInsights.GetQuery(context.Background(), &GetQueryRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       "main",
		QueryID:      "exec-1",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(query.ID, qt.Equals, "exec-1")
	c.Assert(query.Fingerprint, qt.Equals, "b129e8fa")
	c.Assert(query.TotalDurationMillis, qt.Equals, 2.5)
	c.Assert(query.StartedAt.IsZero(), qt.IsFalse)
	c.Assert(query.Password["id"], qt.Equals, "password-1")
	c.Assert(query.Tags[0]["value"], qt.Equals, "web")
}

func TestQueryInsights_GetQuerySummary(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/insights/b129e8fa/summary")
		c.Assert(r.URL.Query().Get("keyspace"), qt.Equals, "main")
		c.Assert(r.URL.Query().Get("from"), qt.Equals, "2026-08-25T17:00:00Z")
		c.Assert(r.URL.Query().Get("to"), qt.Equals, "2026-08-25T18:00:00Z")

		w.WriteHeader(200)
		out := `{
			"id": "summary-1",
			"fingerprint": "b129e8fa",
			"statement_type": "SELECT",
			"keyspace": "main",
			"normalized_sql": "select * from users where id = ?",
			"query_count": 20,
			"error_count": 1,
			"tables": ["users"],
			"index_usages": [{"name": "PRIMARY", "count": 20}],
			"sum_rows_read": 40,
			"sum_rows_returned": 20,
			"rows_read_per_returned": 2,
			"sum_total_duration_millis": 50.5,
			"last_run_at": "2026-08-25T18:00:00.000Z",
			"time_per_query": 2.525,
			"p50_latency": 2.1,
			"p99_latency": 4.8,
			"max_latency": 5.2
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	summary, err := client.QueryInsights.GetQuerySummary(context.Background(), &GetQuerySummaryRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       "main",
		Fingerprint:  "b129e8fa",
	}, WithKeyspace("main"), WithTimeRange("2026-08-25T17:00:00Z", "2026-08-25T18:00:00Z"))

	c.Assert(err, qt.IsNil)
	c.Assert(summary.ID, qt.Equals, "summary-1")
	c.Assert(summary.Fingerprint, qt.Equals, "b129e8fa")
	c.Assert(summary.QueryCount, qt.Equals, int64(20))
	c.Assert(summary.SumTotalDurationMillis, qt.Equals, 50.5)
	c.Assert(summary.IndexUsages[0]["name"], qt.Equals, "PRIMARY")
	c.Assert(summary.LastRunAt.IsZero(), qt.IsFalse)
}

func TestQueryInsights_ListTags(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/insights/tags")

		out := `{
			"type": "list",
			"data": [{
				"id": "Sapp",
				"name": "app",
				"source": "sql",
				"query_count": 100,
				"values": [{"name": "web", "query_count": 80, "kind": "literal"}]
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	tags, err := client.QueryInsights.ListTags(context.Background(), &ListQueryTagsRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       "main",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(tags, qt.HasLen, 1)
	c.Assert(tags[0].ID, qt.Equals, "Sapp")
	c.Assert(tags[0].Name, qt.Equals, "app")
	c.Assert(tags[0].Source, qt.Equals, "sql")
}

func TestQueryInsights_GetTag(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/insights/tags/Sapp")

		out := `{"id":"Sapp","name":"app","source":"sql","query_count":100,"values":[{"name":"web","query_count":80,"kind":"literal"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	tag, err := client.QueryInsights.GetTag(context.Background(), &GetQueryTagRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       "main",
		Tag:          "Sapp",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(tag.ID, qt.Equals, "Sapp")
	c.Assert(tag.Values, qt.HasLen, 1)
}

func TestQueryInsights_ListTagSummaries(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/insights/tags/summaries")
		c.Assert(r.URL.Query()["tags[]"], qt.DeepEquals, []string{"Sapp", "Busername"})
		c.Assert(r.URL.Query().Get("sort"), qt.Equals, "totalTime")

		out := `{
			"type": "list",
			"data": [{
				"dimensions": {"Sapp": "web", "Busername": "alice"},
				"query_count": 10,
				"sum_total_duration_millis": 50.5,
				"p99_latency": 3.2
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	summaries, err := client.QueryInsights.ListTagSummaries(context.Background(), &ListTagSummariesRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       "main",
		Tags:         []string{"Sapp", "Busername"},
	}, WithSort("totalTime", "desc"))

	c.Assert(err, qt.IsNil)
	c.Assert(summaries, qt.HasLen, 1)
	c.Assert(summaries[0].QueryCount, qt.Equals, int64(10))
	c.Assert(summaries[0].Dimensions["Sapp"], qt.Equals, "web")
}
