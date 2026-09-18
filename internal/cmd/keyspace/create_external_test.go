package keyspace

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

func TestKeyspace_CreateExternalCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "commerce"

	ks := &ps.Keyspace{
		ID:          "wantid",
		Name:        keyspace,
		External:    true,
		ClusterSize: "PS_10",
		Shards:      1,
		Replicas:    1,
	}

	svc := &mock.KeyspacesService{
		CreateExternalFn: func(ctx context.Context, req *ps.CreateExternalKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, keyspace)
			c.Assert(req.ClusterSize, qt.Equals, "PS_10")
			c.Assert(req.ExternalDatasource.Hostname, qt.Equals, "db.example.com")
			c.Assert(req.ExternalDatasource.DatabaseName, qt.Equals, "commerce")
			c.Assert(req.ExternalDatasource.Username, qt.Equals, "import")
			c.Assert(req.ExternalDatasource.Password, qt.Equals, "secret")
			c.Assert(req.ExternalDatasource.Port, qt.Equals, 3306)
			c.Assert(req.ExternalDatasource.SSLMode, qt.Equals, "required")
			return ks, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Keyspaces: svc,
			}, nil
		},
	}

	cmd := CreateExternalCmd(ch)
	cmd.SetArgs([]string{
		db, branch, keyspace,
		"--host", "db.example.com",
		"--source-database", "commerce",
		"--username", "import",
		"--password", "secret",
		"--ssl-mode", "required",
		"--cluster-size", "PS_10",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CreateExternalFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, ks)
}

func TestKeyspace_CreateExternalCmdDryRun(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	resp := &ps.LintExternalKeyspaceResponse{
		CanConnect:        true,
		TotalStorageBytes: 100,
	}

	svc := &mock.KeyspacesService{
		LintExternalFn: func(ctx context.Context, req *ps.LintExternalKeyspaceRequest) (*ps.LintExternalKeyspaceResponse, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, "planetscale")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.ExternalDatasource.Hostname, qt.Equals, "db.example.com")
			return resp, nil
		},
		CreateExternalFn: func(ctx context.Context, req *ps.CreateExternalKeyspaceRequest) (*ps.Keyspace, error) {
			c.Fatalf("create should not be called during dry-run")
			return nil, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := CreateExternalCmd(ch)
	cmd.SetArgs([]string{
		"planetscale", "main", "commerce",
		"--host", "db.example.com",
		"--source-database", "commerce",
		"--username", "import",
		"--password", "secret",
		"--ssl-mode", "required",
		"--dry-run",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.LintExternalFnInvoked, qt.IsTrue)
	c.Assert(svc.CreateExternalFnInvoked, qt.IsFalse)
	c.Assert(buf.String(), qt.JSONEquals, resp)
}

func TestKeyspace_CreateExternalCmdDryRunReportsLintErrors(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)

	org := "planetscale"
	resp := &ps.LintExternalKeyspaceResponse{
		CanConnect: true,
		LintErrors: []*ps.ExternalKeyspaceLintError{
			{LintError: "NO_PRIMARY_KEY", TableName: "orders", ErrorDescription: "orders is missing a primary key"},
		},
	}

	svc := &mock.KeyspacesService{
		LintExternalFn: func(ctx context.Context, req *ps.LintExternalKeyspaceRequest) (*ps.LintExternalKeyspaceResponse, error) {
			return resp, nil
		},
		CreateExternalFn: func(ctx context.Context, req *ps.CreateExternalKeyspaceRequest) (*ps.Keyspace, error) {
			c.Fatalf("create should not be called during dry-run")
			return nil, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := CreateExternalCmd(ch)
	cmd.SetArgs([]string{
		"planetscale", "main", "commerce",
		"--host", "db.example.com",
		"--source-database", "commerce",
		"--username", "import",
		"--password", "secret",
		"--ssl-mode", "required",
		"--dry-run",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CreateExternalFnInvoked, qt.IsFalse)
	c.Assert(buf.String(), qt.Contains, "orders is missing a primary key")
	c.Assert(buf.String(), qt.Not(qt.Contains), "is compatible with keyspace")
}
