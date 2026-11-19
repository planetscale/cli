package planetscale

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestInsightsAgent_RateRun(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPut)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/insights/agent/runs/12/feedback")
		body, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(string(body), qt.JSONEquals, map[string]any{"rating": "not_useful", "reason": "too_risky"})

		_, err = w.Write([]byte(`{"rating":"not_useful","reason":"too_risky","note":null,"created_at":"2026-11-19T10:00:00.000Z","updated_at":"2026-11-19T10:00:00.000Z"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	feedback, err := client.InsightsAgent.RateRun(context.Background(), &RateInsightsAgentRunRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       "main",
		Number:       12,
		Rating:       "not_useful",
		Reason:       "too_risky",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(feedback.Rating, qt.Equals, "not_useful")
	c.Assert(feedback.Reason, qt.Equals, "too_risky")
}
