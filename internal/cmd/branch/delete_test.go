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

func TestBranch_DeleteCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"

	svc := &mock.DatabaseBranchesService{
		DeleteFn: func(ctx context.Context, req *ps.DeleteDatabaseBranchRequest) error {
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			return nil
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

	cmd := DeleteCmd(ch)
	cmd.SetArgs([]string{db, branch, "--force"})
	err := cmd.Execute()

	res := map[string]string{
		"result": "branch deleted",
		"branch": branch,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBranch_DeleteCmd_ServiceTokenPermissionError(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"

	// Mock service that returns 404 for branch deletion
	branchSvc := &mock.DatabaseBranchesService{
		DeleteFn: func(ctx context.Context, req *ps.DeleteDatabaseBranchRequest) error {
			return &ps.Error{Code: ps.ErrNotFound}
		},
	}

	// Mock database service that succeeds
	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Kind: "mysql"}, nil
		},
	}

	// Mock organization service that succeeds (simulating valid service token)
	orgSvc := &mock.OrganizationsService{
		ListFn: func(ctx context.Context) ([]*ps.Organization, error) {
			return []*ps.Organization{{Name: org}}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization:   org,
			ServiceTokenID: "valid-token-id",
			ServiceToken:   "valid-token",
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DatabaseBranches: branchSvc,
				Databases:        dbSvc,
				Organizations:    orgSvc,
			}, nil
		},
	}

	cmd := DeleteCmd(ch)
	cmd.SetArgs([]string{db, branch, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
	c.Assert(err.Error(), qt.Contains, "service token for authentication")
	c.Assert(err.Error(), qt.Contains, "delete_branch")
	c.Assert(branchSvc.DeleteFnInvoked, qt.IsTrue)
	c.Assert(orgSvc.ListFnInvoked, qt.IsTrue)
}

func TestBranch_DeleteCmd_PostgreSQL(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"

	svc := &mock.PostgresBranchesService{
		DeleteFn: func(ctx context.Context, req *ps.DeletePostgresBranchRequest) error {
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			return nil
		},
	}

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			return &ps.Database{Kind: "postgres"}, nil
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

	cmd := DeleteCmd(ch)
	cmd.SetArgs([]string{db, branch, "--force"})
	err := cmd.Execute()

	res := map[string]string{
		"result": "branch deleted",
		"branch": branch,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}
