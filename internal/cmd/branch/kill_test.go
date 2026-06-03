package branch

import (
	"bytes"
	"context"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestKill(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	svc := &mock.ProcesslistService{
		KillFn: func(ctx context.Context, req *ps.KillProcessRequest) (*ps.KillProcessResult, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.ID, qt.Equals, int64(101))
			c.Assert(req.Kind, qt.Equals, "connection")
			return &ps.KillProcessResult{
				Success: true, Keyspace: "main", Shard: "-", Tablet: "zone1-2001", ID: 101, Kind: "connection",
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := processlistTestHelper(org, svc, printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"kill", db, branch, "101"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.KillFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"success": true`)
}

func TestKill_QueryFlag(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	svc := &mock.ProcesslistService{
		KillFn: func(ctx context.Context, req *ps.KillProcessRequest) (*ps.KillProcessResult, error) {
			c.Assert(req.Kind, qt.Equals, "query")
			return &ps.KillProcessResult{Success: true, ID: req.ID, Kind: "query"}, nil
		},
	}

	var buf bytes.Buffer
	ch := processlistTestHelper(org, svc, printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"kill", db, branch, "101", "--query"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.KillFnInvoked, qt.IsTrue)
}

func TestKill_InvalidID(t *testing.T) {
	c := qt.New(t)

	svc := &mock.ProcesslistService{
		KillFn: func(ctx context.Context, req *ps.KillProcessRequest) (*ps.KillProcessResult, error) {
			return nil, nil
		},
	}

	var buf bytes.Buffer
	ch := processlistTestHelper("my-org", svc, printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"kill", "my-db", "my-branch", "not-a-number"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(svc.KillFnInvoked, qt.IsFalse)
}

func TestKill_NotFound(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "missing-db", "missing-branch"

	svc := &mock.ProcesslistService{
		KillFn: func(ctx context.Context, req *ps.KillProcessRequest) (*ps.KillProcessResult, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	var buf bytes.Buffer
	ch := processlistTestHelper(org, svc, printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"kill", db, branch, "101"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "process")
	c.Assert(err.Error(), qt.Contains, "101")
	c.Assert(err.Error(), qt.Contains, branch)
	c.Assert(err.Error(), qt.Contains, db)
	c.Assert(err.Error(), qt.Contains, org)
	c.Assert(err.Error(), qt.Not(qt.Contains), "Not Found")
	c.Assert(svc.KillFnInvoked, qt.IsTrue)
}
