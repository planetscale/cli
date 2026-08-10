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

func TestDatabase_ShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"

	res := &ps.Database{Name: "foo", Kind: "mysql"}

	svc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)

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
			}, nil
		},
	}

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDatabase_ShowCmd_HumanVertical(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	autoMigrations := true

	res := &ps.Database{
		Name:                       db,
		Kind:                       "mysql",
		DefaultBranch:              "main",
		RequireApprovalForDeploy:   true,
		RestrictBranchRegion:       false,
		AllowDataBranching:         true,
		ForeignKeysEnabled:         false,
		AutomaticMigrations:        &autoMigrations,
		InsightsRawQueries:         true,
		InsightsEnabled:            true,
		ProductionBranchWebConsole: false,
	}

	svc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
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
			}, nil
		},
	}

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	out := buf.String()
	c.Assert(out, qt.Contains, "Name")
	c.Assert(out, qt.Contains, db)
	c.Assert(out, qt.Contains, "Default Branch")
	c.Assert(out, qt.Contains, "main")
	c.Assert(out, qt.Contains, "Require Approval For Deploy")
	c.Assert(out, qt.Contains, "Allow Data Branching")
	c.Assert(out, qt.Contains, "Automatic Migrations")
	c.Assert(out, qt.Contains, "Insights Raw Queries")
	c.Assert(strings.Contains(out, "|"), qt.IsFalse)
}

func TestDatabase_ShowCmd_HumanVerticalPostgresOmitsVitessSettings(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)

	res := &ps.Database{
		Name:                       "pg-db",
		Kind:                       "postgresql",
		DefaultBranch:              "main",
		RequireApprovalForDeploy:   true,
		AllowDataBranching:         true,
		ForeignKeysEnabled:         true,
		InsightsRawQueries:         false,
		InsightsEnabled:            true,
		ProductionBranchWebConsole: true,
	}

	svc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return res, nil
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

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{"pg-db"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	out := buf.String()
	c.Assert(out, qt.Contains, "postgresql")
	c.Assert(out, qt.Contains, "Default Branch")
	c.Assert(out, qt.Contains, "Insights Raw Queries")
	c.Assert(out, qt.Not(qt.Contains), "Require Approval For Deploy")
	c.Assert(out, qt.Not(qt.Contains), "Allow Data Branching")
	c.Assert(out, qt.Not(qt.Contains), "Foreign Keys Enabled")
	c.Assert(out, qt.Not(qt.Contains), "Automatic Migrations")
	c.Assert(out, qt.Not(qt.Contains), "Migration Framework")
}
