package workflow

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestWorkflow_CreateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	name := "test-workflow"
	sourceKeyspace := "source_ks"
	targetKeyspace := "target_ks"
	globalKeyspace := "global_ks"
	tables := []string{"table1", "table2"}
	deferSecondaryKeys := true
	onDDL := "STOP"

	createdAt := time.Now()

	// Create expected workflow response with all required fields
	expectedWorkflow := &ps.Workflow{
		ID:        "workflow1",
		Name:      name,
		State:     "pending",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Tables:    []*ps.WorkflowTable{{Name: "table1"}, {Name: "table2"}},
		SourceKeyspace: ps.Keyspace{
			Name: sourceKeyspace,
		},
		TargetKeyspace: ps.Keyspace{
			Name: targetKeyspace,
		},
		Branch: ps.DatabaseBranch{
			Name: branch,
		},
		Actor: ps.Actor{
			Name: "test-user",
		},
	}

	// Mock the workflow service
	svc := &mock.WorkflowsService{
		CreateFn: func(ctx context.Context, req *ps.CreateWorkflowRequest) (*ps.Workflow, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Name, qt.Equals, name)
			c.Assert(req.SourceKeyspace, qt.Equals, sourceKeyspace)
			c.Assert(req.TargetKeyspace, qt.Equals, targetKeyspace)
			c.Assert(*req.GlobalKeyspace, qt.Equals, globalKeyspace)
			c.Assert(req.Tables, qt.DeepEquals, tables)
			c.Assert(*req.DeferSecondaryKeys, qt.Equals, deferSecondaryKeys)
			c.Assert(*req.OnDDL, qt.Equals, onDDL)

			return expectedWorkflow, nil
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

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{
		db,
		branch,
		"--name", name,
		"--source-keyspace", sourceKeyspace,
		"--target-keyspace", targetKeyspace,
		"--global-keyspace", globalKeyspace,
		"--tables", "table1,table2",
		"--defer-secondary-keys",
		"--on-ddl", onDDL,
	})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, expectedWorkflow)
}

// Note: Testing the interactive mode is challenging since it uses huh forms
// which require terminal interaction. This would typically be covered by
// integration tests.

func TestWorkflow_CreateCmd_Error(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human // Use human format to avoid calling toWorkflow
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	name := "test-workflow"
	sourceKeyspace := "source_ks"
	targetKeyspace := "target_ks"
	globalKeyspace := "global_ks"

	// Mock the workflow service to return an error
	svc := &mock.WorkflowsService{
		CreateFn: func(ctx context.Context, req *ps.CreateWorkflowRequest) (*ps.Workflow, error) {
			return nil, &ps.Error{
				Code: ps.ErrNotFound,
			}
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

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{
		db,
		branch,
		"--name", name,
		"--source-keyspace", sourceKeyspace,
		"--target-keyspace", targetKeyspace,
		"--global-keyspace", globalKeyspace,
		"--tables", "table1,table2",
		"--defer-secondary-keys",
		"--on-ddl", "STOP",
	})

	err := cmd.Execute()

	c.Assert(err, qt.Not(qt.IsNil))
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
}
