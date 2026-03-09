package vtctld

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func moveTablesTestHelper(org string, moveTables ps.MoveTablesService, vtctld ps.VtctldService, buf *bytes.Buffer) *cmdutil.Helper {
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)

	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				MoveTables: moveTables,
				Vtctld:     vtctld,
			}, nil
		},
	}
}

func setMoveTablesPollInterval(t *testing.T, interval time.Duration) {
	t.Helper()

	previous := moveTablesOperationPollInterval
	moveTablesOperationPollInterval = interval

	t.Cleanup(func() {
		moveTablesOperationPollInterval = previous
	})
}

func setMoveTablesOperationTimeouts(t *testing.T, defaultTimeout, timeoutBuffer time.Duration) {
	t.Helper()

	previousDefaultTimeout := moveTablesOperationDefaultTimeout
	previousTimeoutBuffer := moveTablesOperationTimeoutBuffer
	moveTablesOperationDefaultTimeout = defaultTimeout
	moveTablesOperationTimeoutBuffer = timeoutBuffer

	t.Cleanup(func() {
		moveTablesOperationDefaultTimeout = previousDefaultTimeout
		moveTablesOperationTimeoutBuffer = previousTimeoutBuffer
	})
}

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
			// Verify optional fields are not set when not provided
			c.Assert(req.DeferSecondaryKeys, qt.IsNil)
			c.Assert(req.Cells, qt.IsNil)
			c.Assert(req.TabletTypes, qt.IsNil)
			c.Assert(req.ExcludeTables, qt.IsNil)
			c.Assert(req.AtomicCopy, qt.IsNil)
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

func TestMoveTablesCreateWithDeferSecondaryKeysFalse(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MoveTablesService{
		CreateFn: func(ctx context.Context, req *ps.MoveTablesCreateRequest) (json.RawMessage, error) {
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			c.Assert(req.SourceKeyspace, qt.Equals, "source-ks")
			c.Assert(req.DeferSecondaryKeys, qt.IsNotNil)
			c.Assert(*req.DeferSecondaryKeys, qt.IsFalse)
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
		"--defer-secondary-keys=false",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
}

func TestMoveTablesCreateWithAllFlags(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MoveTablesService{
		CreateFn: func(ctx context.Context, req *ps.MoveTablesCreateRequest) (json.RawMessage, error) {
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			c.Assert(req.SourceKeyspace, qt.Equals, "source-ks")
			c.Assert(req.Cells, qt.DeepEquals, []string{"us-east-1", "us-west-2"})
			c.Assert(req.TabletTypes, qt.DeepEquals, []string{"PRIMARY", "REPLICA"})
			c.Assert(req.ExcludeTables, qt.DeepEquals, []string{"t1", "t2"})
			c.Assert(req.AtomicCopy, qt.IsNotNil)
			c.Assert(*req.AtomicCopy, qt.IsTrue)
			c.Assert(req.AllTables, qt.IsNotNil)
			c.Assert(*req.AllTables, qt.IsTrue)
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
		"--all-tables",
		"--cells", "us-east-1,us-west-2",
		"--tablet-types", "PRIMARY,REPLICA",
		"--exclude-tables", "t1,t2",
		"--atomic-copy",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
}

func TestMoveTablesSwitchTrafficWithMaxLag(t *testing.T) {
	c := qt.New(t)
	setMoveTablesPollInterval(t, 0)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MoveTablesService{
		SwitchTrafficFn: func(ctx context.Context, req *ps.MoveTablesSwitchTrafficRequest) (*ps.VtctldOperationReference, error) {
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			c.Assert(req.TabletTypes, qt.DeepEquals, []string{"PRIMARY"})
			c.Assert(req.MaxReplicationLagAllowed, qt.IsNotNil)
			c.Assert(*req.MaxReplicationLagAllowed, qt.Equals, int64(30))
			return &ps.VtctldOperationReference{ID: "switch-op"}, nil
		},
	}

	getOperationCalls := 0
	vtctldSvc := &mock.VtctldService{
		GetOperationFn: func(ctx context.Context, req *ps.GetVtctldOperationRequest) (*ps.VtctldOperation, error) {
			getOperationCalls++

			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.ID, qt.Equals, "switch-op")

			if getOperationCalls == 1 {
				return &ps.VtctldOperation{ID: "switch-op", State: "running", Completed: false}, nil
			}

			return &ps.VtctldOperation{
				ID:        "switch-op",
				State:     "completed",
				Completed: true,
				Result:    json.RawMessage(`{"summary":"switched"}`),
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := moveTablesTestHelper(org, svc, vtctldSvc, &buf)

	cmd := MoveTablesCmd(ch)
	cmd.SetArgs([]string{"switch-traffic", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
		"--tablet-types", "PRIMARY",
		"--max-replication-lag-allowed", "30",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.SwitchTrafficFnInvoked, qt.IsTrue)
	c.Assert(vtctldSvc.GetOperationFnInvoked, qt.IsTrue)
	c.Assert(getOperationCalls, qt.Equals, 2)
	c.Assert(buf.String(), qt.JSONEquals, map[string]string{"summary": "switched"})
}

func TestMoveTablesReverseTrafficWithFlags(t *testing.T) {
	c := qt.New(t)
	setMoveTablesPollInterval(t, 0)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MoveTablesService{
		ReverseTrafficFn: func(ctx context.Context, req *ps.MoveTablesReverseTrafficRequest) (*ps.VtctldOperationReference, error) {
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			c.Assert(req.TabletTypes, qt.DeepEquals, []string{"REPLICA", "RDONLY"})
			c.Assert(req.MaxReplicationLagAllowed, qt.IsNotNil)
			c.Assert(*req.MaxReplicationLagAllowed, qt.Equals, int64(60))
			return &ps.VtctldOperationReference{ID: "reverse-op"}, nil
		},
	}
	vtctldSvc := &mock.VtctldService{
		GetOperationFn: func(ctx context.Context, req *ps.GetVtctldOperationRequest) (*ps.VtctldOperation, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.ID, qt.Equals, "reverse-op")

			return &ps.VtctldOperation{
				ID:        "reverse-op",
				State:     "completed",
				Completed: true,
				Result:    json.RawMessage(`{"summary":"reversed"}`),
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := moveTablesTestHelper(org, svc, vtctldSvc, &buf)

	cmd := MoveTablesCmd(ch)
	cmd.SetArgs([]string{"reverse-traffic", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
		"--tablet-types", "REPLICA,RDONLY",
		"--max-replication-lag-allowed", "60",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ReverseTrafficFnInvoked, qt.IsTrue)
	c.Assert(vtctldSvc.GetOperationFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]string{"summary": "reversed"})
}

func TestMoveTablesCompleteWithFlags(t *testing.T) {
	c := qt.New(t)
	setMoveTablesPollInterval(t, 0)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MoveTablesService{
		CompleteFn: func(ctx context.Context, req *ps.MoveTablesCompleteRequest) (*ps.VtctldOperationReference, error) {
			c.Assert(req.Workflow, qt.Equals, "my-workflow")
			c.Assert(req.TargetKeyspace, qt.Equals, "target-ks")
			c.Assert(req.KeepData, qt.IsNotNil)
			c.Assert(*req.KeepData, qt.IsTrue)
			c.Assert(req.KeepRoutingRules, qt.IsNotNil)
			c.Assert(*req.KeepRoutingRules, qt.IsTrue)
			c.Assert(req.RenameTables, qt.IsNotNil)
			c.Assert(*req.RenameTables, qt.IsTrue)
			c.Assert(req.DryRun, qt.IsNotNil)
			c.Assert(*req.DryRun, qt.IsTrue)

			return &ps.VtctldOperationReference{ID: "complete-op"}, nil
		},
	}
	vtctldSvc := &mock.VtctldService{
		GetOperationFn: func(ctx context.Context, req *ps.GetVtctldOperationRequest) (*ps.VtctldOperation, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.ID, qt.Equals, "complete-op")

			return &ps.VtctldOperation{
				ID:        "complete-op",
				State:     "completed",
				Completed: true,
				Result:    json.RawMessage(`{"summary":"completed"}`),
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := moveTablesTestHelper(org, svc, vtctldSvc, &buf)

	cmd := MoveTablesCmd(ch)
	cmd.SetArgs([]string{"complete", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
		"--keep-data",
		"--keep-routing-rules",
		"--rename-tables",
		"--dry-run",
	})

	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CompleteFnInvoked, qt.IsTrue)
	c.Assert(vtctldSvc.GetOperationFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]string{"summary": "completed"})
}

func TestMoveTablesSwitchTrafficOperationFailure(t *testing.T) {
	c := qt.New(t)
	setMoveTablesPollInterval(t, 0)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MoveTablesService{
		SwitchTrafficFn: func(ctx context.Context, req *ps.MoveTablesSwitchTrafficRequest) (*ps.VtctldOperationReference, error) {
			return &ps.VtctldOperationReference{ID: "switch-op"}, nil
		},
	}
	vtctldSvc := &mock.VtctldService{
		GetOperationFn: func(ctx context.Context, req *ps.GetVtctldOperationRequest) (*ps.VtctldOperation, error) {
			return &ps.VtctldOperation{
				ID:        "switch-op",
				State:     "failed",
				Completed: true,
				Error:     "switch traffic failed",
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := moveTablesTestHelper(org, svc, vtctldSvc, &buf)

	cmd := MoveTablesCmd(ch)
	cmd.SetArgs([]string{"switch-traffic", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
	})

	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, "switch traffic failed")
	c.Assert(buf.String(), qt.Equals, "")
}

func TestMoveTablesSwitchTrafficOperationTimeout(t *testing.T) {
	c := qt.New(t)
	setMoveTablesPollInterval(t, 0)
	setMoveTablesOperationTimeouts(t, 0, 0)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MoveTablesService{
		SwitchTrafficFn: func(ctx context.Context, req *ps.MoveTablesSwitchTrafficRequest) (*ps.VtctldOperationReference, error) {
			return &ps.VtctldOperationReference{ID: "switch-op"}, nil
		},
	}
	vtctldSvc := &mock.VtctldService{
		GetOperationFn: func(ctx context.Context, req *ps.GetVtctldOperationRequest) (*ps.VtctldOperation, error) {
			return &ps.VtctldOperation{
				ID:        "switch-op",
				State:     "running",
				Completed: false,
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := moveTablesTestHelper(org, svc, vtctldSvc, &buf)

	cmd := MoveTablesCmd(ch)
	cmd.SetArgs([]string{"switch-traffic", db, branch,
		"--workflow", "my-workflow",
		"--target-keyspace", "target-ks",
	})

	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, "timed out waiting for vtctld operation switch-op to finish")
	c.Assert(buf.String(), qt.Equals, "")
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
