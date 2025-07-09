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

func TestBranch_ShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"

	res := &ps.DatabaseBranch{Name: branch}

	svc := &mock.DatabaseBranchesService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)

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

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBranch_ShowCmd_ServiceTokenAuthError(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"

	// Mock service that returns 404 for branch lookup
	branchSvc := &mock.DatabaseBranchesService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	// Mock database service that succeeds
	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Kind: "mysql"}, nil
		},
	}

	// Mock organization service that returns error for auth check (simulating invalid service token)
	orgSvc := &mock.OrganizationsService{
		ListFn: func(ctx context.Context) ([]*ps.Organization, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization:   org,
			ServiceTokenID: "test-token-id",
			ServiceToken:   "test-token",
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DatabaseBranches: branchSvc,
				Databases:        dbSvc,
				Organizations:    orgSvc,
			}, nil
		},
	}

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "Authentication failed")
	c.Assert(err.Error(), qt.Contains, "service token appears to be invalid")
	c.Assert(err.Error(), qt.Contains, "Service tokens can be provided in two ways")
	c.Assert(branchSvc.GetFnInvoked, qt.IsTrue)
	c.Assert(orgSvc.ListFnInvoked, qt.IsTrue)
}

func TestBranch_ShowCmd_GenuineNotFound(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "nonexistent"

	// Mock service that returns 404 for branch lookup
	branchSvc := &mock.DatabaseBranchesService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
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

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
	c.Assert(err.Error(), qt.Contains, "service token for authentication")
	c.Assert(err.Error(), qt.Contains, "read_branch")
	c.Assert(err.Error(), qt.Not(qt.Contains), "Authentication failed")
	c.Assert(branchSvc.GetFnInvoked, qt.IsTrue)
	c.Assert(orgSvc.ListFnInvoked, qt.IsTrue)
}

func TestBranch_ShowCmd_GenuineNotFound_NoServiceToken(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "nonexistent"

	// Mock service that returns 404 for branch lookup
	branchSvc := &mock.DatabaseBranchesService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	// Mock database service that succeeds
	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Kind: "mysql"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
			// No service token set - using regular authentication
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DatabaseBranches: branchSvc,
				Databases:        dbSvc,
			}, nil
		},
	}

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
	c.Assert(err.Error(), qt.Not(qt.Contains), "service token for authentication")
	c.Assert(err.Error(), qt.Not(qt.Contains), "read_branch")
	c.Assert(err.Error(), qt.Not(qt.Contains), "Authentication failed")
	c.Assert(branchSvc.GetFnInvoked, qt.IsTrue)
}

func TestBranch_ShowCmd_GenuineNotFound_EmptyPermission(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "nonexistent"

	// Mock service that returns 404 for branch lookup
	branchSvc := &mock.DatabaseBranchesService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
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

	// Manually call the function with empty permission to test this case
	ctx := context.Background()
	err := cmdutil.HandleNotFoundWithServiceTokenCheck(
		ctx, nil, ch.Config, ch.Client, &ps.Error{Code: ps.ErrNotFound}, "",
		"branch %s does not exist in database %s (organization: %s)",
		printer.BoldBlue(branch), printer.BoldBlue(db), printer.BoldBlue(ch.Config.Organization))

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
	c.Assert(err.Error(), qt.Not(qt.Contains), "service token for authentication")
	c.Assert(err.Error(), qt.Not(qt.Contains), "permission")
}

func TestBranch_ShowCmd_PostgreSQL(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"

	res := &ps.PostgresBranch{Name: branch}

	svc := &mock.PostgresBranchesService{
		GetFn: func(ctx context.Context, req *ps.GetPostgresBranchRequest) (*ps.PostgresBranch, error) {
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)

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

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}
