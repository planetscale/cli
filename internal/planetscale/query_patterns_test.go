package planetscale

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

const testQueryPatternsCSV = "normalized_sql,query_count\nselect ?,10\n"

func TestQueryPatterns_CreateReport(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/query-patterns")
		w.WriteHeader(201)
		out := `{"id":"report1","state":"pending","actor":{"id":"actor1","type":"User","display_name":"user@example.com"},"url":"https://api.planetscale.com/v1/organizations/my-org/databases/my-db/branches/my-branch/query-patterns/report1","created_at":"2021-01-14T10:19:23.000Z"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))
	t.Cleanup(ts.Close)

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	report, err := client.QueryPatterns.CreateReport(context.Background(), &CreateQueryPatternsReportRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
	})

	want := &QueryPatternsReport{
		PublicID: "report1",
		State:    "pending",
		Actor: &Actor{
			ID:   "actor1",
			Type: "User",
			Name: "user@example.com",
		},
		URL:       "https://api.planetscale.com/v1/organizations/my-org/databases/my-db/branches/my-branch/query-patterns/report1",
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(report, qt.DeepEquals, want)
}

func TestQueryPatterns_GetReport(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/query-patterns/report1")
		w.WriteHeader(200)
		out := `{"id":"report1","state":"completed","download_url":"https://api.planetscale.com/v1/organizations/my-org/databases/my-db/branches/my-branch/query-patterns/report1/download","created_at":"2021-01-14T10:19:23.000Z","finished_at":"2021-01-14T10:20:23.000Z"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))
	t.Cleanup(ts.Close)

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	report, err := client.QueryPatterns.GetReport(context.Background(), &GetQueryPatternsReportRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Report:       "report1",
	})

	want := &QueryPatternsReport{
		PublicID:    "report1",
		State:       "completed",
		DownloadURL: "https://api.planetscale.com/v1/organizations/my-org/databases/my-db/branches/my-branch/query-patterns/report1/download",
		CreatedAt:   time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		FinishedAt:  time.Date(2021, time.January, 14, 10, 20, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(report, qt.DeepEquals, want)
}

func TestQueryPatterns_ListReports(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/query-patterns")
		c.Assert(r.URL.Query().Get("limit"), qt.Equals, "10")
		c.Assert(r.URL.Query().Get("starting_after"), qt.Equals, "report0")
		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":[{"id":"report1","state":"completed","created_at":"2021-01-14T10:19:23.000Z"}]}`))
		c.Assert(err, qt.IsNil)
	}))
	t.Cleanup(ts.Close)

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	reports, err := client.QueryPatterns.ListReports(context.Background(), &ListQueryPatternsReportsRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
	}, WithLimit(10), WithStartingAfter("report0"))
	c.Assert(err, qt.IsNil)
	c.Assert(reports, qt.HasLen, 1)
	c.Assert(reports[0].PublicID, qt.Equals, "report1")
	c.Assert(reports[0].State, qt.Equals, "completed")
}

func TestQueryPatterns_DeleteReport(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/query-patterns/report1")
		w.WriteHeader(204)
	}))
	t.Cleanup(ts.Close)

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	err = client.QueryPatterns.DeleteReport(context.Background(), &DeleteQueryPatternsReportRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Report:       "report1",
	})
	c.Assert(err, qt.IsNil)
}

func TestQueryPatterns_DownloadReport(t *testing.T) {
	c := qt.New(t)

	// blob simulates presigned storage: it rejects requests that carry the
	// API's Authorization header.
	blob := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Header.Get("Authorization"), qt.Equals, "")
		w.Header().Set("Content-Type", "application/gzip")
		gz := gzip.NewWriter(w)
		_, err := gz.Write([]byte(testQueryPatternsCSV))
		c.Assert(err, qt.IsNil)
		c.Assert(gz.Close(), qt.IsNil)
	}))
	t.Cleanup(blob.Close)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/query-patterns/report1/download")
		c.Assert(r.Header.Get("Authorization"), qt.Equals, "tid:secret")
		http.Redirect(w, r, blob.URL, http.StatusFound)
	}))
	t.Cleanup(ts.Close)

	client, err := NewClient(WithBaseURL(ts.URL), WithServiceToken("tid", "secret"))
	c.Assert(err, qt.IsNil)

	body, err := client.QueryPatterns.DownloadReport(context.Background(), &DownloadQueryPatternsReportRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Report:       "report1",
	})
	c.Assert(err, qt.IsNil)
	defer body.Close()

	data, err := io.ReadAll(body)
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, testQueryPatternsCSV)
}

func TestQueryPatterns_DownloadReportUncompressed(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, err := w.Write([]byte(testQueryPatternsCSV))
		c.Assert(err, qt.IsNil)
	}))
	t.Cleanup(ts.Close)

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	body, err := client.QueryPatterns.DownloadReport(context.Background(), &DownloadQueryPatternsReportRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Report:       "report1",
	})
	c.Assert(err, qt.IsNil)
	defer body.Close()

	data, err := io.ReadAll(body)
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, testQueryPatternsCSV)
}

func TestQueryPatterns_DownloadReportNotFound(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, err := w.Write([]byte(`{"code":"not_found","message":"Not Found"}`))
		c.Assert(err, qt.IsNil)
	}))
	t.Cleanup(ts.Close)

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	_, err = client.QueryPatterns.DownloadReport(context.Background(), &DownloadQueryPatternsReportRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Report:       "report1",
	})
	c.Assert(err, qt.IsNotNil)

	var perr *Error
	c.Assert(err, qt.ErrorAs, &perr)
	c.Assert(perr.Code, qt.Equals, ErrNotFound)
}
