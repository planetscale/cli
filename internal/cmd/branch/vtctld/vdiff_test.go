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
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestVDiffCreate(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.VDiffService{
		CreateFn: func(ctx context.Context, req *ps.VDiffCreateRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			return json.RawMessage(`{"uuid":"abc-123"}`), nil
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
				VDiff: svc,
			}, nil
		},
	}

	cmd := VDiffCmd(ch)
	cmd.SetArgs([]string{"create", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]any{
		"uuid": "abc-123",
		"next_steps": []any{map[string]any{
			"command": "pscale branch vtctld vdiff show my-db my-branch --org my-org --workflow my-workflow --target-keyspace target-ks --uuid abc-123 --format json",
			"reason":  "Check VDiff progress",
		}},
	})
}

func TestVDiffList(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.VDiffService{
		ListFn: func(ctx context.Context, req *ps.VDiffListRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			return json.RawMessage(`{"vdiffs":[]}`), nil
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
				VDiff: svc,
			}, nil
		},
	}

	cmd := VDiffCmd(ch)
	cmd.SetArgs([]string{"list", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
}

func TestVDiffShow(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.VDiffService{
		ShowFn: func(ctx context.Context, req *ps.VDiffShowRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.UUID, qt.Equals, "abc-123")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			return json.RawMessage(`{"summary":{"state":"STATE_COMPLETED","has_mismatch":false}}`), nil
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
				VDiff: svc,
			}, nil
		},
	}

	cmd := VDiffCmd(ch)
	cmd.SetArgs([]string{"show", db, branch,
		"--workflow", "my-workflow",
		"--uuid", "abc-123",
		"--target-keyspace", "target-ks",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ShowFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]any{
		"summary": map[string]any{
			"state":        "STATE_COMPLETED",
			"has_mismatch": false,
		},
		"next_steps": []any{map[string]any{
			"command": "pscale branch vtctld move-tables switch-traffic my-db my-branch --org my-org --workflow my-workflow --target-keyspace target-ks --tablet-types REPLICA,RDONLY --format json",
			"reason":  "Switch replica traffic to the target keyspace",
		}},
	})
}

func TestVDiffDelete(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.VDiffService{
		DeleteFn: func(ctx context.Context, req *ps.VDiffDeleteRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.UUID, qt.Equals, "abc-123")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			return json.RawMessage(`{"uuid":"abc-123"}`), nil
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
				VDiff: svc,
			}, nil
		},
	}

	cmd := VDiffCmd(ch)
	cmd.SetArgs([]string{"delete", db, branch,
		"--workflow", "my-workflow",
		"--uuid", "abc-123",
		"--target-keyspace", "target-ks",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)
}
