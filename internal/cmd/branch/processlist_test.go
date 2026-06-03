package branch

import (
	"bytes"
	"context"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func processlistTestHelper(org string, svc ps.ProcesslistService, format printer.Format, buf *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)

	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Processlist: svc}, nil
		},
	}
}

func TestProcesslist(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	svc := &mock.ProcesslistService{
		ListFn: func(ctx context.Context, req *ps.ProcesslistRequest) (*ps.ProcesslistResult, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, "commerce")
			c.Assert(req.Shard, qt.Equals, "-80")
			return &ps.ProcesslistResult{
				Keyspace: "commerce",
				Shard:    "-80",
				Tablet:   "zone1-1001",
				Processes: []ps.Process{
					{ID: 101, User: "vt_app", Command: "Query", Time: 42, Info: "SELECT 1"},
				},
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := processlistTestHelper(org, svc, printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"show", db, branch, "--keyspace", "commerce", "--shard", "-80"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"tablet": "zone1-1001"`)
	c.Assert(buf.String(), qt.Contains, `"user": "vt_app"`)
}

func TestProcesslist_CSVOutput(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	svc := &mock.ProcesslistService{
		ListFn: func(ctx context.Context, req *ps.ProcesslistRequest) (*ps.ProcesslistResult, error) {
			return &ps.ProcesslistResult{
				Keyspace: "commerce",
				Shard:    "-80",
				Tablet:   "zone1-1001",
				Processes: []ps.Process{
					{ID: 101, User: "vt_app", Host: "10.0.0.1", DB: "main", Command: "Query", Time: 42, State: "running", Info: "SELECT 1"},
				},
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := processlistTestHelper(org, svc, printer.CSV, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"show", db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "101,vt_app,10.0.0.1,main,Query,42,running,SELECT 1")
	c.Assert(buf.String(), qt.Not(qt.Contains), "{")
	c.Assert(buf.String(), qt.Not(qt.Contains), `"processes"`)
}

func TestProcesslist_NoTargetFlags(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	svc := &mock.ProcesslistService{
		ListFn: func(ctx context.Context, req *ps.ProcesslistRequest) (*ps.ProcesslistResult, error) {
			c.Assert(req.Keyspace, qt.Equals, "")
			c.Assert(req.Shard, qt.Equals, "")
			return &ps.ProcesslistResult{Keyspace: "main", Shard: "-", Tablet: "zone1-2001"}, nil
		},
	}

	var buf bytes.Buffer
	ch := processlistTestHelper(org, svc, printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"show", db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
}

func TestProcesslist_HumanOutputDoesNotAbbreviateNumericFields(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	svc := &mock.ProcesslistService{
		ListFn: func(ctx context.Context, req *ps.ProcesslistRequest) (*ps.ProcesslistResult, error) {
			return &ps.ProcesslistResult{
				Keyspace: "main",
				Shard:    "-",
				Tablet:   "zone1-2001",
				Processes: []ps.Process{
					{ID: 121500, User: "vt_app", Command: "Sleep", Time: 2100},
				},
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := processlistTestHelper(org, svc, printer.Human, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"show", db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "TIME (SECONDS)")
	c.Assert(buf.String(), qt.Contains, "121500")
	c.Assert(buf.String(), qt.Not(qt.Contains), "121.5K")
	c.Assert(buf.String(), qt.Contains, "2100")
	c.Assert(buf.String(), qt.Not(qt.Contains), "2.1K")
}

func TestProcesslist_NotFound(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "missing-db", "missing-branch"

	svc := &mock.ProcesslistService{
		ListFn: func(ctx context.Context, req *ps.ProcesslistRequest) (*ps.ProcesslistResult, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	var buf bytes.Buffer
	ch := processlistTestHelper(org, svc, printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"show", db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "branch")
	c.Assert(err.Error(), qt.Contains, branch)
	c.Assert(err.Error(), qt.Contains, db)
	c.Assert(err.Error(), qt.Contains, org)
	c.Assert(err.Error(), qt.Not(qt.Contains), "Not Found")
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
}
