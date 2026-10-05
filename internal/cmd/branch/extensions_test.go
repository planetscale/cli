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
				{Name: "vector", Loader: "shared_preload_libraries", Enabled: true, CanEnable: true, URL: "https://github.com/pgvector/pgvector"},
				{Name: "pg_stat_statements", Loader: "shared_preload_libraries", CanEnable: false},
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
				{Name: "vector", CanEnable: true},
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

func TestBranch_ExtensionsTogglePreservesOtherExtensions(t *testing.T) {
	for _, tc := range []struct {
		verb     string
		enabled  bool
		expected []string
	}{
		{verb: "enable", expected: []string{"pg_stat_statements", "pg_strict", "vector"}},
		{verb: "disable", enabled: true, expected: []string{"pg_stat_statements", "pg_strict"}},
	} {
		t.Run(tc.verb, func(t *testing.T) {
			c := qt.New(t)
			var buf bytes.Buffer
			format := printer.JSON
			p := printer.NewPrinter(&format)
			p.SetResourceOutput(&buf)
			pgSvc := &mock.PostgresBranchesService{
				ListExtensionsFn: func(_ context.Context, _ *ps.ListPostgresExtensionsRequest) ([]*ps.PostgresExtension, error) {
					return []*ps.PostgresExtension{
						{Name: "pg_stat_statements", Enabled: true},
						{Name: "vector", Enabled: tc.enabled, CanEnable: true},
						{Name: "pg_strict", Enabled: true, CanEnable: true},
						{Name: "pg_cron", CanEnable: true},
					}, nil
				},
				ResizeFn: func(_ context.Context, req *ps.ResizePostgresBranchRequest) (*ps.PostgresBranchClusterResizeRequest, error) {
					c.Assert(req.Organization, qt.Equals, "planetscale")
					c.Assert(req.Database, qt.Equals, "postgres-db")
					c.Assert(req.Branch, qt.Equals, "main")
					c.Assert(req.Parameters, qt.IsNil)
					c.Assert(req.Extensions, qt.IsNotNil)
					c.Assert(*req.Extensions, qt.DeepEquals, tc.expected)
					return &ps.PostgresBranchClusterResizeRequest{ID: "change-id", State: "queued"}, nil
				},
			}
			ch := &cmdutil.Helper{Printer: p, Config: &config.Config{Organization: "planetscale"}, Client: func() (*ps.Client, error) {
				return &ps.Client{PostgresBranches: pgSvc}, nil
			}}
			cmd := ExtensionsCmd(ch)
			cmd.SetArgs([]string{tc.verb, "postgres-db", "main", "vector"})
			c.Assert(cmd.Execute(), qt.IsNil)
			c.Assert(pgSvc.ResizeFnInvoked, qt.IsTrue)
			c.Assert(buf.String(), qt.Contains, "change-id")
		})
	}
}

func TestBranch_ExtensionsToggleRejectsWhenCannotEnable(t *testing.T) {
	for _, verb := range []string{"enable", "disable"} {
		t.Run(verb, func(t *testing.T) {
			c := qt.New(t)
			var buf bytes.Buffer
			format := printer.JSON
			p := printer.NewPrinter(&format)
			p.SetResourceOutput(&buf)
			pgSvc := &mock.PostgresBranchesService{
				ListExtensionsFn: func(_ context.Context, _ *ps.ListPostgresExtensionsRequest) ([]*ps.PostgresExtension, error) {
					return []*ps.PostgresExtension{{Name: "hstore", Enabled: verb == "disable"}}, nil
				},
			}
			ch := &cmdutil.Helper{Printer: p, Config: &config.Config{Organization: "planetscale"}, Client: func() (*ps.Client, error) {
				return &ps.Client{PostgresBranches: pgSvc}, nil
			}}
			cmd := ExtensionsCmd(ch)
			cmd.SetArgs([]string{verb, "postgres-db", "main", "hstore"})
			c.Assert(cmd.Execute(), qt.ErrorMatches, "extension hstore cannot be enabled or disabled")
			c.Assert(pgSvc.ResizeFnInvoked, qt.IsFalse)
		})
	}
}

func TestBranch_ExtensionsToggleSkipsAlreadyEnabled(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)
	pgSvc := &mock.PostgresBranchesService{
		ListExtensionsFn: func(_ context.Context, _ *ps.ListPostgresExtensionsRequest) ([]*ps.PostgresExtension, error) {
			return []*ps.PostgresExtension{{Name: "vector", Enabled: true, CanEnable: true}}, nil
		},
	}
	ch := &cmdutil.Helper{Printer: p, Config: &config.Config{Organization: "planetscale"}, Client: func() (*ps.Client, error) {
		return &ps.Client{PostgresBranches: pgSvc}, nil
	}}
	cmd := ExtensionsCmd(ch)
	cmd.SetArgs([]string{"enable", "postgres-db", "main", "vector"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(pgSvc.ResizeFnInvoked, qt.IsFalse)
	c.Assert(buf.String(), qt.Contains, "no_change")
}
