package insights

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func TestInsights_ErrorsShowCmd(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{
		ListErrorQueriesFn: func(ctx context.Context, req *ps.ListErrorQueriesRequest, opts ...ps.ListOption) ([]*ps.QuerySample, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "mydb")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.Fingerprint, qt.Equals, "b129e8fa")
			return []*ps.QuerySample{{
				ID:            "exec-1",
				NormalizedSQL: "select * from users where id = ?",
				Keyspace:      "main",
				Username:      "app",
				ErrorMessage:  "vttablet: rpc error",
				StartedAt:     time.Date(2026, 8, 11, 18, 0, 0, 0, time.UTC),
			}}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{QueryInsights: svc})

	cmd := ErrorsShowCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "b129e8fa"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListErrorQueriesFnInvoked, qt.IsTrue)

	var out []map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out, qt.HasLen, 1)
	c.Assert(out[0]["error_message"], qt.Equals, "vttablet: rpc error")
}

func TestInsights_ErrorsShowCmd_NotFound(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{
		ListErrorQueriesFn: func(ctx context.Context, req *ps.ListErrorQueriesRequest, opts ...ps.ListOption) ([]*ps.QuerySample, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{QueryInsights: svc})

	cmd := ErrorsShowCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "b129e8fa"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
}

func TestInsights_ErrorsCmd_HasShowSubcommand(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{})

	cmd, _, err := ErrorsCmd(ch).Find([]string{"show"})
	c.Assert(err, qt.IsNil)
	c.Assert(cmd.Name(), qt.Equals, "show")
}
