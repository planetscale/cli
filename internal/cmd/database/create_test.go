package database

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	qt "github.com/frankban/quicktest"
)

func TestDatabase_CreateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"

	res := &ps.Database{Name: "foo"}

	svc := &mock.DatabaseService{
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Name, qt.Equals, db)
			c.Assert(req.Region, qt.Equals, "us-east")
			c.Assert(req.Replicas, qt.IsNil) // Should be nil when not set

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
				Databases: svc,
				Organizations: &mock.OrganizationsService{
					GetFn: func(ctx context.Context, request *ps.GetOrganizationRequest) (*ps.Organization, error) {
						return &ps.Organization{
							RemainingFreeDatabases: 1,
							Name:                   request.Organization,
						}, nil
					},
				},
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, "--region", "us-east"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDatabase_CreateCmdWithReplicasZero(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"

	res := &ps.Database{Name: "foo"}

	svc := &mock.DatabaseService{
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Name, qt.Equals, db)
			c.Assert(req.Region, qt.Equals, "us-east")
			c.Assert(req.Replicas, qt.IsNotNil)   // Should be set when explicitly passed
			c.Assert(*req.Replicas, qt.Equals, 0) // Should be 0

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
				Databases: svc,
				Organizations: &mock.OrganizationsService{
					GetFn: func(ctx context.Context, request *ps.GetOrganizationRequest) (*ps.Organization, error) {
						return &ps.Organization{
							RemainingFreeDatabases: 1,
							Name:                   request.Organization,
						}, nil
					},
				},
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, "--region", "us-east", "--replicas", "0"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDatabase_CreateCmdPostgres(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"

	res := &ps.Database{Name: "foo"}

	svc := &mock.DatabaseService{
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Name, qt.Equals, db)
			c.Assert(req.Region, qt.Equals, "us-east")
			c.Assert(req.Kind, qt.Equals, ps.DatabaseEnginePostgres)

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
				Databases: svc,
				Organizations: &mock.OrganizationsService{
					GetFn: func(ctx context.Context, request *ps.GetOrganizationRequest) (*ps.Organization, error) {
						return &ps.Organization{
							RemainingFreeDatabases: 1,
							Name:                   request.Organization,
						}, nil
					},
				},
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, "--region", "us-east", "--engine", "postgresql"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDatabase_CreateCmdWithReplicas(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"

	res := &ps.Database{Name: "foo"}

	svc := &mock.DatabaseService{
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Name, qt.Equals, db)
			c.Assert(req.Region, qt.Equals, "us-east")
			c.Assert(req.Replicas, qt.IsNotNil)
			c.Assert(*req.Replicas, qt.Equals, 3)

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
				Databases: svc,
				Organizations: &mock.OrganizationsService{
					GetFn: func(ctx context.Context, request *ps.GetOrganizationRequest) (*ps.Organization, error) {
						return &ps.Organization{
							RemainingFreeDatabases: 1,
							Name:                   request.Organization,
						}, nil
					},
				},
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, "--region", "us-east", "--replicas", "3"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}
