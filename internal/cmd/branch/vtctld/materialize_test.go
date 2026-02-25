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

func TestMaterializeCreate(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MaterializeService{
		CreateFn: func(ctx context.Context, req *ps.MaterializeCreateRequest) (json.RawMessage, error) {
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
				Materialize: svc,
			}, nil
		},
	}

	cmd := MaterializeCmd(ch)
	cmd.SetArgs([]string{"create", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
		"--source-keyspace", "source-ks",
		"--table-settings", `[{"target_table":"t1","source_expression":"select * from t1"}]`,
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
}

func TestMaterializeShow(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MaterializeService{
		ShowFn: func(ctx context.Context, req *ps.MaterializeShowRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			c.Assert(*req.IncludeLogs, qt.IsTrue)
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
				Materialize: svc,
			}, nil
		},
	}

	cmd := MaterializeCmd(ch)
	cmd.SetArgs([]string{"show", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
		"--include-logs",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ShowFnInvoked, qt.IsTrue)
}

func TestMaterializeStart(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MaterializeService{
		StartFn: func(ctx context.Context, req *ps.MaterializeStartRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			return json.RawMessage(`{"result":"started"}`), nil
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
				Materialize: svc,
			}, nil
		},
	}

	cmd := MaterializeCmd(ch)
	cmd.SetArgs([]string{"start", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.StartFnInvoked, qt.IsTrue)
}

func TestMaterializeStop(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MaterializeService{
		StopFn: func(ctx context.Context, req *ps.MaterializeStopRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			return json.RawMessage(`{"result":"stopped"}`), nil
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
				Materialize: svc,
			}, nil
		},
	}

	cmd := MaterializeCmd(ch)
	cmd.SetArgs([]string{"stop", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.StopFnInvoked, qt.IsTrue)
}

func TestMaterializeCancel(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MaterializeService{
		CancelFn: func(ctx context.Context, req *ps.MaterializeCancelRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			return json.RawMessage(`{"result":"canceled"}`), nil
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
				Materialize: svc,
			}, nil
		},
	}

	cmd := MaterializeCmd(ch)
	cmd.SetArgs([]string{"cancel", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CancelFnInvoked, qt.IsTrue)
}
