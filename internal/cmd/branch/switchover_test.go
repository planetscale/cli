package branch

import (
	"bytes"
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestBranch_SwitchoverCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"

	res := &ps.PostgresSwitchover{
		ID:    "switchover-1",
		State: "pending",
	}

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: db, Kind: ps.DatabaseEnginePostgres}, nil
		},
	}
	svc := &mock.PostgresSwitchoversService{
		CreateFn: func(ctx context.Context, req *ps.CreatePostgresSwitchoverRequest) (*ps.PostgresSwitchover, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Candidate, qt.Equals, "hzi-replica-2")

			return res, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases:           dbSvc,
				PostgresSwitchovers: svc,
			}, nil
		},
	}

	cmd := SwitchoverCmd(ch)
	cmd.SetArgs([]string{db, branch, "--candidate", "hzi-replica-2"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBranch_SwitchoverCmd_VitessDatabase(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: req.Database, Kind: ps.DatabaseEngineMySQL}, nil
		},
	}
	svc := &mock.PostgresSwitchoversService{
		CreateFn: func(ctx context.Context, req *ps.CreatePostgresSwitchoverRequest) (*ps.PostgresSwitchover, error) {
			return nil, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: "planetscale",
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases:           dbSvc,
				PostgresSwitchovers: svc,
			}, nil
		},
	}

	cmd := SwitchoverCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*only available for PostgreSQL.*mysql.*`)
	c.Assert(svc.CreateFnInvoked, qt.IsFalse)
}
