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

func TestMoveTablesCreate(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MoveTablesService{
		CreateFn: func(ctx context.Context, req *ps.MoveTablesCreateRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			c.Assert(req.SourceKeyspace, qt.Equals, "source-ks")
			return json.RawMessage(`{"summary":"created"}`), nil
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
				MoveTables: svc,
			}, nil
		},
	}

	cmd := MoveTablesCmd(ch)
	cmd.SetArgs([]string{"create", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
		"--source-keyspace", "source-ks",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
}

func TestMoveTablesShow(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MoveTablesService{
		ShowFn: func(ctx context.Context, req *ps.MoveTablesShowRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			return json.RawMessage(`{"workflow":"my-workflow"}`), nil
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
				MoveTables: svc,
			}, nil
		},
	}

	cmd := MoveTablesCmd(ch)
	cmd.SetArgs([]string{"show", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ShowFnInvoked, qt.IsTrue)
}
