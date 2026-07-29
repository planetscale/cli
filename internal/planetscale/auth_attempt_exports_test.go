package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestAuthAttemptExports_CreateExport(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/auth-attempt-exports")

		var got map[string]any
		c.Assert(json.NewDecoder(r.Body).Decode(&got), qt.IsNil)
		want := map[string]any{
			"start_at": "2026-07-29T00:00:00Z",
			"end_at":   "2026-07-29T01:00:00Z",
			"format":   "parquet",
			"filters": map[string]any{
				"source_ips":        []any{"203.0.113.0/24", "2001:db8::/112"},
				"branches":          []any{"db/production"},
				"outcomes":          []any{"deny"},
				"usernames":         []any{"incident-user"},
				"startup_databases": []any{""},
				"failure_reasons":   []any{"bad_password"},
				"backend_routes":    []any{"postgres"},
			},
		}
		c.Assert(got, qt.DeepEquals, want)

		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(`{"id":"export1","state":"pending","format":"parquet"}`))
		c.Assert(err, qt.IsNil)
	}))
	t.Cleanup(ts.Close)

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	export, err := client.AuthAttemptExports.CreateExport(context.Background(), &CreateAuthAttemptExportRequest{
		Organization: "my-org",
		StartAt:      time.Date(2026, time.July, 29, 0, 0, 0, 0, time.UTC),
		EndAt:        time.Date(2026, time.July, 29, 1, 0, 0, 0, time.UTC),
		Format:       "parquet",
		Filters: AuthAttemptExportFilters{
			SourceIPs:        []string{"203.0.113.0/24", "2001:db8::/112"},
			Branches:         []string{"db/production"},
			Outcomes:         []string{"deny"},
			Usernames:        []string{"incident-user"},
			StartupDatabases: []string{""},
			FailureReasons:   []string{"bad_password"},
			BackendRoutes:    []string{"postgres"},
		},
	})

	c.Assert(err, qt.IsNil)
	c.Assert(export.PublicID, qt.Equals, "export1")
	c.Assert(export.State, qt.Equals, "pending")
	c.Assert(export.Format, qt.Equals, "parquet")
	c.Assert(export.RetryAfter, qt.Equals, 5*time.Second)
}

func TestAuthAttemptExports_GetExport(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/auth-attempt-exports/export1")

		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"id":"export1",
			"state":"failed",
			"start_at":"2026-07-29T00:00:00Z",
			"end_at":"2026-07-29T01:00:00Z",
			"format":"parquet",
			"filters":{
				"source_ips":["203.0.113.0/24"],
				"branches":["db/production"],
				"outcomes":["deny"],
				"usernames":["incident-user"],
				"startup_databases":[""],
				"failure_reasons":["bad_password"],
				"backend_routes":["postgres"]
			},
			"resolved_branch_public_ids":["branch1"],
			"created_at":"2026-07-29T01:01:00Z",
			"started_at":"2026-07-29T01:02:00Z",
			"generated_at":"2026-07-29T01:03:00Z",
			"finished_at":"2026-07-29T01:04:00Z",
			"expires_at":"2026-07-30T01:04:00Z",
			"failure_reason":"generation_failed",
			"failure_detail":"The export could not be generated.",
			"recovery_hint":"Re-create the export; if the error repeats, contact support."
		}`))
		c.Assert(err, qt.IsNil)
	}))
	t.Cleanup(ts.Close)

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	export, err := client.AuthAttemptExports.GetExport(context.Background(), &GetAuthAttemptExportRequest{
		Organization: "my-org",
		Export:       "export1",
	})

	startedAt := time.Date(2026, time.July, 29, 1, 2, 0, 0, time.UTC)
	generatedAt := time.Date(2026, time.July, 29, 1, 3, 0, 0, time.UTC)
	finishedAt := time.Date(2026, time.July, 29, 1, 4, 0, 0, time.UTC)
	expiresAt := time.Date(2026, time.July, 30, 1, 4, 0, 0, time.UTC)
	want := &AuthAttemptExport{
		PublicID: "export1",
		State:    "failed",
		StartAt:  time.Date(2026, time.July, 29, 0, 0, 0, 0, time.UTC),
		EndAt:    time.Date(2026, time.July, 29, 1, 0, 0, 0, time.UTC),
		Format:   "parquet",
		Filters: AuthAttemptExportFilters{
			SourceIPs:        []string{"203.0.113.0/24"},
			Branches:         []string{"db/production"},
			Outcomes:         []string{"deny"},
			Usernames:        []string{"incident-user"},
			StartupDatabases: []string{""},
			FailureReasons:   []string{"bad_password"},
			BackendRoutes:    []string{"postgres"},
		},
		ResolvedBranchPublicIDs: []string{"branch1"},
		CreatedAt:               time.Date(2026, time.July, 29, 1, 1, 0, 0, time.UTC),
		StartedAt:               &startedAt,
		GeneratedAt:             &generatedAt,
		FinishedAt:              &finishedAt,
		ExpiresAt:               &expiresAt,
		FailureReason:           "generation_failed",
		FailureDetail:           "The export could not be generated.",
		RecoveryHint:            "Re-create the export; if the error repeats, contact support.",
		RetryAfter:              2 * time.Second,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(export, qt.DeepEquals, want)
}

func TestParseRetryAfter(t *testing.T) {
	c := qt.New(t)
	future := time.Now().Add(2 * time.Minute).UTC().Format(http.TimeFormat)
	past := time.Now().Add(-time.Minute).UTC().Format(http.TimeFormat)
	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{name: "positive delta seconds", header: "5", want: 5 * time.Second},
		{name: "future HTTP date", header: future},
		{name: "absent", header: "", want: 0},
		{name: "malformed", header: "soon", want: 0},
		{name: "fractional", header: "1.5", want: 0},
		{name: "zero", header: "0", want: 0},
		{name: "past HTTP date", header: past, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := parseRetryAfter(test.header)
			if test.name == "future HTTP date" {
				c.Assert(got > 0, qt.IsTrue)
				return
			}
			c.Assert(got, qt.Equals, test.want)
		})
	}
}
