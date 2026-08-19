package database

import (
	"bytes"
	"context"
	"strings"
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

func TestDatabase_CreateCmdWithWaitPrintsReadyDatabase(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	pending := &ps.Database{Name: db, State: ps.DatabasePending}
	ready := &ps.Database{Name: db, State: ps.DatabaseReady}

	svc := &mock.DatabaseService{
		CreateFn: func(_ context.Context, _ *ps.CreateDatabaseRequest) (*ps.Database, error) {
			return pending, nil
		},
		GetFn: func(_ context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			c.Assert(req, qt.DeepEquals, &ps.GetDatabaseRequest{
				Organization: org,
				Database:     db,
			})
			return ready, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases: svc,
				Organizations: &mock.OrganizationsService{
					GetFn: func(_ context.Context, req *ps.GetOrganizationRequest) (*ps.Organization, error) {
						return &ps.Organization{RemainingFreeDatabases: 1, Name: req.Organization}, nil
					},
				},
			}, nil
		},
	}
	debug := false
	ch.SetDebug(&debug)

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, "--region", "us-east", "--wait"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, ready)
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
	tests := []struct {
		name  string
		value string
		stdin string
	}{
		{
			name:  "inline JSON",
			value: `{"account_id":"cf_account_123","timestamp":"1710000000","signature":"abc123sig"}`,
		},
		{
			name:  "JSON from stdin",
			value: "@-",
			stdin: `{"account_id":"cf_account_123","timestamp":"1710000000","signature":"abc123sig"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			var buf bytes.Buffer
			format := printer.JSON
			p := printer.NewPrinter(&format)
			p.SetResourceOutput(&buf)

			const (
				org = "planetscale"
				db  = "planetscale"
			)
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
				Config:  &config.Config{Organization: org},
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
			cmd.SetIn(strings.NewReader(tt.stdin))
			cmd.SetArgs([]string{db, "--region", "us-east", "--cloudflare-billing", tt.value})
			err := cmd.Execute()

			c.Assert(err, qt.IsNil)
			c.Assert(svc.CreateFnInvoked, qt.IsTrue)
			c.Assert(buf.String(), qt.JSONEquals, res)
		})
	}
}

func TestParseCloudflareBillingErrors(t *testing.T) {
	tests := []struct {
		name  string
		value string
		match string
	}{
		{
			name:  "invalid JSON",
			value: `{`,
			match: `parsing --cloudflare-billing JSON: unexpected EOF`,
		},
		{
			name:  "unknown field",
			value: `{"account_id":"account","timestamp":"123","signature":"sig","extra":"value"}`,
			match: `parsing --cloudflare-billing JSON: json: unknown field "extra"`,
		},
		{
			name:  "missing field",
			value: `{"account_id":"account","timestamp":"123"}`,
			match: `--cloudflare-billing requires non-empty "account_id", "timestamp", and "signature" fields`,
		},
		{
			name:  "multiple values",
			value: `{"account_id":"account","timestamp":"123","signature":"sig"} {}`,
			match: `parsing --cloudflare-billing JSON: multiple JSON values are not allowed`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			_, err := parseCloudflareBilling(tt.value, strings.NewReader(""))
			c.Assert(err, qt.ErrorMatches, tt.match)
		})
	}
}

func TestDatabase_CreateCmdCloudflareBillingInvalid(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.DatabaseService{
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseRequest) (*ps.Database, error) {
			c.Fatalf("Create should not be called with invalid Cloudflare billing JSON")
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
				Databases: svc,
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{"planetscale", "--cloudflare-billing", `{"account_id":"cf_account_123"}`})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `.*--cloudflare-billing requires non-empty "account_id", "timestamp", and "signature" fields.*`)
	c.Assert(svc.CreateFnInvoked, qt.IsFalse)
}

func TestDatabase_CreateCmdCloudflareFlagHidden(t *testing.T) {
	c := qt.New(t)

	cmd := CreateCmd(&cmdutil.Helper{})
	flag := cmd.Flags().Lookup("cloudflare-billing")
	c.Assert(flag, qt.IsNotNil)
	c.Assert(flag.Hidden, qt.IsTrue)
	for _, removed := range []string{"cloudflare-account-id", "cloudflare-timestamp", "cloudflare-signature"} {
		c.Assert(cmd.Flags().Lookup(removed), qt.IsNil)
	}
}
