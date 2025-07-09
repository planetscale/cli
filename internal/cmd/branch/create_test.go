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

func TestBranch_CreateCmd(t *testing.T) {
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
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			c.Assert(req.Name, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Region, qt.Equals, "us-east")
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

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, branch, "--region", "us-east"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBranch_CreateCmdWithRestore(t *testing.T) {
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
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			c.Assert(req.Name, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Region, qt.Equals, "us-east")
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.BackupID, qt.Equals, "somebackupid")
			c.Assert(req.ClusterSize, qt.Equals, "PS-20")

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
				DatabaseBranches: svc,
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, branch, "--region", "us-east", "--restore", "somebackupid", "--cluster-size", "PS-20"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBranch_CreateCmdWithRestoreNoSize(t *testing.T) {
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
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			c.Assert(req.Name, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Region, qt.Equals, "us-east")
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.BackupID, qt.Equals, "somebackupid")
			c.Assert(req.ClusterSize, qt.Equals, "PS-10")

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
				DatabaseBranches: svc,
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, branch, "--region", "us-east", "--restore", "somebackupid"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBranch_CreateCmdWithSeedData(t *testing.T) {
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
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			c.Assert(req.Name, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Region, qt.Equals, "us-east")
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.SeedData, qt.Equals, "last_successful_backup")

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
				DatabaseBranches: svc,
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, branch, "--region", "us-east", "--seed-data"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBranch_CreateCmd_ServiceTokenAuthError(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"

	// Mock service that returns 404 for branch creation
	branchSvc := &mock.DatabaseBranchesService{
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
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
				Organizations:    orgSvc,
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "Authentication failed")
	c.Assert(err.Error(), qt.Contains, "service token appears to be invalid")
	c.Assert(err.Error(), qt.Contains, "Service tokens can be provided in two ways")
	c.Assert(branchSvc.CreateFnInvoked, qt.IsTrue)
	c.Assert(orgSvc.ListFnInvoked, qt.IsTrue)
}

func TestBranch_CreateCmd_GenuineNotFound(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "nonexistent"
	branch := "development"

	// Mock service that returns 404 for branch creation
	branchSvc := &mock.DatabaseBranchesService{
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
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
				Organizations:    orgSvc,
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
	c.Assert(err.Error(), qt.Contains, "service token for authentication")
	c.Assert(err.Error(), qt.Contains, "create_branch")
	c.Assert(err.Error(), qt.Not(qt.Contains), "Authentication failed")
	c.Assert(branchSvc.CreateFnInvoked, qt.IsTrue)
	c.Assert(orgSvc.ListFnInvoked, qt.IsTrue)
}
