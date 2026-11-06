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

func TestBranch_DemoteCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"

	res := &ps.DatabaseBranch{
		Name:           branch,
		Production:     false,
		SafeMigrations: false,
	}

	svc := &mock.DatabaseBranchesService{
		DemoteFn: func(ctx context.Context, req *ps.DemoteRequest) (*ps.DatabaseBranch, error) {
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.ClusterSize, qt.Equals, "")
			c.Assert(req.Replicas, qt.IsNil)

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

	cmd := DemoteCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DemoteFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBranch_DemoteCmd_ClusterSizeAndReplicas(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	replicas := 0

	svc := &mock.DatabaseBranchesService{
		DemoteFn: func(ctx context.Context, req *ps.DemoteRequest) (*ps.DatabaseBranch, error) {
			c.Assert(req.ClusterSize, qt.Equals, "PS_DEV")
			c.Assert(req.Replicas, qt.DeepEquals, &replicas)
			return &ps.DatabaseBranch{Name: branch, Production: false}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DatabaseBranches: svc}, nil
		},
	}

	cmd := DemoteCmd(ch)
	cmd.SetArgs([]string{db, branch, "--cluster-size", "PS-DEV", "--replicas", "0"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.DemoteFnInvoked, qt.IsTrue)
}

func TestBranch_DemoteCmd_HelpStatesEngineSupport(t *testing.T) {
	c := qt.New(t)
	cmd := DemoteCmd(&cmdutil.Helper{})

	c.Assert(cmd.Long, qt.Contains, "This command is supported for MySQL databases and for enabled Postgres branches. Neki databases are not supported.")
}

func TestBranch_DemoteCmd_ServiceTokenPermissionError(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"

	// Mock service that returns 404 for branch demotion
	branchSvc := &mock.DatabaseBranchesService{
		DemoteFn: func(ctx context.Context, req *ps.DemoteRequest) (*ps.DatabaseBranch, error) {
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

	cmd := DemoteCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
	c.Assert(err.Error(), qt.Contains, "service token for authentication")
	c.Assert(err.Error(), qt.Contains, "connect_production_branch")
	c.Assert(branchSvc.DemoteFnInvoked, qt.IsTrue)
	c.Assert(orgSvc.ListFnInvoked, qt.IsTrue)
}
