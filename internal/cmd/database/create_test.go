package database

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

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

func TestDatabase_CreateCmdWithStorageMySQLError(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"

	svc := &mock.DatabaseService{
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseRequest) (*ps.Database, error) {
			c.Fatal("CreateFn should not be called for MySQL with storage flags")
			return nil, nil
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
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, "--region", "us-east", "--min-storage", "10737418240"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err, qt.ErrorMatches, ".*only supported for PostgreSQL.*")
	c.Assert(svc.CreateFnInvoked, qt.IsFalse)
}

func TestDatabase_CreateCmdWithStorage(t *testing.T) {
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
			c.Assert(req.Storage, qt.IsNotNil)
			c.Assert(req.Storage.MinimumStorageBytes, qt.IsNotNil)
			c.Assert(*req.Storage.MinimumStorageBytes, qt.Equals, int64(10737418240))
			c.Assert(req.Storage.MaximumStorageBytes, qt.IsNotNil)
			c.Assert(*req.Storage.MaximumStorageBytes, qt.Equals, int64(107374182400))

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
	cmd.SetArgs([]string{db, "--region", "us-east", "--engine", "postgresql", "--min-storage", "10737418240", "--max-storage", "107374182400"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDatabase_CreateCmdPostgresWithMajorVersion(t *testing.T) {
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
			c.Assert(req.MajorVersion, qt.Equals, "17")

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
	cmd.SetArgs([]string{db, "--region", "us-east", "--engine", "postgresql", "--major-version", "17"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDatabase_CreateCmdCloudflareBilling(t *testing.T) {
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
			c.Assert(req.CloudflareAccountID, qt.Equals, "cf_account_123")
			c.Assert(req.CloudflareTimestamp, qt.Equals, "1710000000")
			c.Assert(req.CloudflareSignature, qt.Equals, "abc123sig")

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
	cmd.SetArgs([]string{
		db,
		"--region", "us-east",
		"--cloudflare-account-id", "cf_account_123",
		"--cloudflare-timestamp", "1710000000",
		"--cloudflare-signature", "abc123sig",
	})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDatabase_CreateCmdCloudflareBillingIncomplete(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"

	svc := &mock.DatabaseService{
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseRequest) (*ps.Database, error) {
			c.Fatalf("Create should not be called with incomplete Cloudflare flags")
			return nil, nil
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
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, "--cloudflare-account-id", "cf_account_123"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, ".*cloudflare-account-id, --cloudflare-timestamp, and --cloudflare-signature are all required.*")
	c.Assert(svc.CreateFnInvoked, qt.IsFalse)
}

func TestDatabase_CreateCmdCloudflareFlagsHidden(t *testing.T) {
	c := qt.New(t)

	cmd := CreateCmd(&cmdutil.Helper{})
	for _, name := range []string{"cloudflare-account-id", "cloudflare-timestamp", "cloudflare-signature"} {
		flag := cmd.Flags().Lookup(name)
		c.Assert(flag, qt.IsNotNil, qt.Commentf("flag %q", name))
		c.Assert(flag.Hidden, qt.IsTrue, qt.Commentf("flag %q", name))
	}
}
