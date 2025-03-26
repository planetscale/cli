package workflow

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestWorkflow_ShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"

	workflow := &ps.Workflow{
		ID:     "workflow1",
		Number: 1,
	}

	svc := &mock.WorkflowsService{
		GetFn: func(ctx context.Context, req *ps.GetWorkflowRequest) (*ps.Workflow, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.WorkflowNumber, qt.Equals, uint64(1))

			return workflow, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Workflows: svc,
			}, nil
		},
	}

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, "1"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, workflow)
}
