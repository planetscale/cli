package vtctld

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestListWorkflows(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.VtctldService{
		ListWorkflowsFn: func(ctx context.Context, req *ps.VtctldListWorkflowsRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, "my-keyspace")
			return json.RawMessage(`{"workflows":[]}`), nil
		},
	}

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Vtctld: svc,
			}, nil
		},
	}

	cmd := ListWorkflowsCmd(ch)
	cmd.SetArgs([]string{db, branch,
		"--keyspace", "my-keyspace",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListWorkflowsFnInvoked, qt.IsTrue)
}
