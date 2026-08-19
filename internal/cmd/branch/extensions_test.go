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

func TestBranch_ExtensionsCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "postgres-db"
	branch := "main"

	pgSvc := &mock.PostgresBranchesService{
		ListExtensionsFn: func(ctx context.Context, req *ps.ListPostgresExtensionsRequest) ([]*ps.PostgresExtension, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			return []*ps.PostgresExtension{
				{Name: "vector", Loader: "shared_preload_libraries", Available: true, URL: "https://github.com/pgvector/pgvector"},
				{Name: "pg_stat_statements", Loader: "shared_preload_libraries", Available: false, UnavailableReason: "container_upgrade_required"},
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PostgresBranches: pgSvc}, nil
		},
	}

	cmd := ExtensionsCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(pgSvc.ListExtensionsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "vector")
	c.Assert(buf.String(), qt.Contains, "pg_stat_statements")
}

func TestBranch_ExtensionsCmd_ListSubcommand(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	pgSvc := &mock.PostgresBranchesService{
		ListExtensionsFn: func(ctx context.Context, req *ps.ListPostgresExtensionsRequest) ([]*ps.PostgresExtension, error) {
			return []*ps.PostgresExtension{
				{Name: "vector", Available: true},
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PostgresBranches: pgSvc}, nil
		},
	}

	cmd := ExtensionsCmd(ch)
	cmd.SetArgs([]string{"list", "postgres-db", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(pgSvc.ListExtensionsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "vector")
}
