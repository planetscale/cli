package insights

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestQueryTrafficBudgetsCmd(t *testing.T) {
	c := qt.New(t)

	budgets := []*ps.TrafficBudget{{
		ID:        "budget-1",
		Name:      "App queries",
		Mode:      "enforce",
		CreatedAt: time.Date(2026, 8, 11, 18, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 8, 11, 18, 0, 0, 0, time.UTC),
	}}
	svc := &mock.QueryInsightsService{
		ListQueryTrafficBudgetsFn: func(ctx context.Context, req *ps.ListQueryTrafficBudgetsRequest, opts ...ps.ListOption) ([]*ps.TrafficBudget, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "mydb")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.Fingerprint, qt.Equals, "b129e8fa")

			values := url.Values{}
			listOpts := &ps.ListOptions{URLValues: &values}
			for _, opt := range opts {
				c.Assert(opt(listOpts), qt.IsNil)
			}
			c.Assert(values.Get("keyspace"), qt.Equals, "mydb")
			c.Assert(values.Get("page"), qt.Equals, "3")
			c.Assert(values.Get("per_page"), qt.Equals, "50")

			return budgets, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{QueryInsights: svc})

	cmd := QueryTrafficBudgetsCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "b129e8fa", "--keyspace", "mydb", "--page", "3", "--per-page", "50"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListQueryTrafficBudgetsFnInvoked, qt.IsTrue)
	c.Assert(cmd.Aliases, qt.HasLen, 0)

	var out []map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out, qt.HasLen, 1)
	c.Assert(out[0]["id"], qt.Equals, "budget-1")
	c.Assert(out[0]["name"], qt.Equals, "App queries")
}
