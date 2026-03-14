package vtctld

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func setNonTTYProgress(t *testing.T) {
	t.Helper()

	previous := printer.IsTTY
	printer.IsTTY = false

	t.Cleanup(func() {
		printer.IsTTY = previous
	})
}

func newHumanProgressHelper(org string, humanOut io.Writer, client *ps.Client) *cmdutil.Helper {
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(humanOut)
	p.SetResourceOutput(io.Discard)

	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return client, nil
		},
	}
}

func TestListWorkflowsProgressIncludesOrganization(t *testing.T) {
	c := qt.New(t)
	setNonTTYProgress(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.VtctldService{
		ListWorkflowsFn: func(ctx context.Context, req *ps.VtctldListWorkflowsRequest) (json.RawMessage, error) {
			return json.RawMessage(`{"workflows":[]}`), nil
		},
	}

	var progress bytes.Buffer
	ch := newHumanProgressHelper(org, &progress, &ps.Client{Vtctld: svc})

	cmd := ListWorkflowsCmd(ch)
	cmd.SetArgs([]string{db, branch, "--keyspace", "my-keyspace"})

	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(progress.String(), qt.Contains, "Fetching workflows for my-org/my-db/my-branch…")
}

func TestStartWorkflowProgressIncludesOrganization(t *testing.T) {
	c := qt.New(t)
	setNonTTYProgress(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.VtctldService{
		StartWorkflowFn: func(ctx context.Context, req *ps.VtctldStartWorkflowRequest) (json.RawMessage, error) {
			return json.RawMessage(`{"summary":"started"}`), nil
		},
	}

	var progress bytes.Buffer
	ch := newHumanProgressHelper(org, &progress, &ps.Client{Vtctld: svc})

	cmd := StartWorkflowCmd(ch)
	cmd.SetArgs([]string{db, branch, "my-workflow", "--keyspace", "my-keyspace"})

	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(progress.String(), qt.Contains, "Starting workflow my-workflow on my-org/my-db/my-branch…")
}

func TestMoveTablesShowProgressIncludesOrganization(t *testing.T) {
	c := qt.New(t)
	setNonTTYProgress(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.MoveTablesService{
		ShowFn: func(ctx context.Context, req *ps.MoveTablesShowRequest) (json.RawMessage, error) {
			return json.RawMessage(`{"workflow":"my-workflow"}`), nil
		},
	}

	var progress bytes.Buffer
	ch := newHumanProgressHelper(org, &progress, &ps.Client{MoveTables: svc})

	cmd := MoveTablesCmd(ch)
	cmd.SetArgs([]string{"show", db, branch, "--workflow", "my-workflow", "--target-keyspace", "target-ks"})

	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(progress.String(), qt.Contains, "Fetching MoveTables workflow my-workflow on my-org/my-db/my-branch…")
}

func TestVDiffListProgressIncludesOrganization(t *testing.T) {
	c := qt.New(t)
	setNonTTYProgress(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.VDiffService{
		ListFn: func(ctx context.Context, req *ps.VDiffListRequest) (json.RawMessage, error) {
			return json.RawMessage(`{"vdiffs":[]}`), nil
		},
	}

	var progress bytes.Buffer
	ch := newHumanProgressHelper(org, &progress, &ps.Client{VDiff: svc})

	cmd := VDiffCmd(ch)
	cmd.SetArgs([]string{"list", db, branch, "--workflow", "my-workflow", "--target-keyspace", "target-ks"})

	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(progress.String(), qt.Contains, "Fetching VDiffs for workflow my-workflow on my-org/my-db/my-branch…")
}
