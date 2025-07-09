package branch

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

func TestBranchSchemaCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "feature"

	res := []*ps.Diff{
		{Name: "foo"},
		{Name: "bar"},
	}

	svc := &mock.DatabaseBranchesService{
		SchemaFn: func(ctx context.Context, req *ps.BranchSchemaRequest) ([]*ps.Diff, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)

			return res, nil
		},
	}

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			return &ps.Database{Kind: "mysql"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DatabaseBranches: svc,
				Databases:        dbSvc,
			}, nil
		},
	}

	cmd := SchemaCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.SchemaFnInvoked, qt.IsTrue)

	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBranchSchemaCmd_PostgreSQL(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "feature"

	res := []*ps.PostgresBranchSchema{
		{Name: "foo", Raw: "CREATE TABLE foo (id INT);"},
		{Name: "bar", Raw: "CREATE TABLE bar (id INT);"},
	}

	svc := &mock.PostgresBranchesService{
		SchemaFn: func(ctx context.Context, req *ps.PostgresBranchSchemaRequest) ([]*ps.PostgresBranchSchema, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)

			return res, nil
		},
	}

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			return &ps.Database{Kind: "postgresql"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				PostgresBranches: svc,
				Databases:        dbSvc,
			}, nil
		},
	}

	cmd := SchemaCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.SchemaFnInvoked, qt.IsTrue)

	c.Assert(buf.String(), qt.JSONEquals, res)
}
