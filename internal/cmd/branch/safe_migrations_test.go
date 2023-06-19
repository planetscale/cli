package branch

import (
	"bytes"
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestBranch_EnableSafeMigrationsCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "production"

	res := &ps.DatabaseBranch{
		Name:           branch,
		SafeMigrations: true,
		Production:     true,
	}

	svc := &mock.DatabaseBranchesService{
		EnableSafeMigrationsFn: func(ctx context.Context, req *ps.EnableSafeMigrationsRequest) (*ps.DatabaseBranch, error) {
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)

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

	cmd := EnableSafeMigrationsCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.EnableSafeMigrationsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBranch_EnableSafeMigrationsCmdWithLintingErrors(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "production"

	lintError := &ps.SchemaLintError{
		LintError:        "NO_UNIQUE_KEY",
		SubjectType:      "table_error",
		Keyspace:         "test-database",
		Table:            "test",
		ErrorDescription: "table \"test\" has no unique key: all tables must have at least one unique, not-null key.",
		DocsURL:          "https://planetscale.com/docs/learn/change-single-unique-key",
	}

	lintErrors := []*ps.SchemaLintError{lintError}

	svc := &mock.DatabaseBranchesService{
		EnableSafeMigrationsFn: func(ctx context.Context, req *ps.EnableSafeMigrationsRequest) (*ps.DatabaseBranch, error) {
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)

			return nil, &ps.Error{
				Code: ps.ErrRetry,
			}
		},
		LintSchemaFn: func(ctx context.Context, req *ps.LintSchemaRequest) ([]*ps.SchemaLintError, error) {
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)

			return lintErrors, nil
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

	cmd := EnableSafeMigrationsCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.EnableSafeMigrationsFnInvoked, qt.IsTrue)
	c.Assert(svc.LintSchemaFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, lintErrors)
}

func TestBranch_DisableSafeMigrationsCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "production"

	res := &ps.DatabaseBranch{
		Name:           branch,
		SafeMigrations: false,
		Production:     false,
	}

	svc := &mock.DatabaseBranchesService{
		DisableSafeMigrationsFn: func(ctx context.Context, req *ps.DisableSafeMigrationsRequest) (*ps.DatabaseBranch, error) {
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)

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

	cmd := DisableSafeMigrationsCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DisableSafeMigrationsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}
