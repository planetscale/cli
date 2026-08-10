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

func TestDatabase_UpdateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	defaultBranch := "production"

	res := &ps.Database{
		Name:                       db,
		Kind:                       "mysql",
		DefaultBranch:              defaultBranch,
		RequireApprovalForDeploy:   true,
		InsightsRawQueries:         false,
		ProductionBranchWebConsole: true,
	}

	svc := &mock.DatabaseService{
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateDatabaseSettingsRequest) (*ps.Database, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.DefaultBranch, qt.IsNotNil)
			c.Assert(*req.DefaultBranch, qt.Equals, defaultBranch)
			c.Assert(req.RequireApprovalForDeploy, qt.IsNotNil)
			c.Assert(*req.RequireApprovalForDeploy, qt.IsTrue)
			c.Assert(req.InsightsRawQueries, qt.IsNotNil)
			c.Assert(*req.InsightsRawQueries, qt.IsFalse)
			c.Assert(req.RestrictBranchRegion, qt.IsNil)
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

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{
		db,
		"--default-branch", defaultBranch,
		"--require-approval-for-deploy=true",
		"--insights-raw-queries=false",
	})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDatabase_UpdateCmd_NewName(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "old-name"
	newName := "new-name"

	res := &ps.Database{
		Name:          newName,
		Kind:          "postgresql",
		DefaultBranch: "main",
	}

	svc := &mock.DatabaseService{
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateDatabaseSettingsRequest) (*ps.Database, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.NewName, qt.IsNotNil)
			c.Assert(*req.NewName, qt.Equals, newName)
			c.Assert(req.DefaultBranch, qt.IsNil)
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

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{db, "--new-name", newName})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDatabase_UpdateCmd_VitessFlags(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	framework := "rails"
	tableName := "schema_migrations"
	autoMigrations := true

	res := &ps.Database{
		Name:                db,
		Kind:                "mysql",
		AllowDataBranching:  true,
		ForeignKeysEnabled:  true,
		AutomaticMigrations: &autoMigrations,
		MigrationFramework:  &framework,
		MigrationTableName:  &tableName,
	}

	svc := &mock.DatabaseService{
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateDatabaseSettingsRequest) (*ps.Database, error) {
			c.Assert(req.AllowDataBranching, qt.IsNotNil)
			c.Assert(*req.AllowDataBranching, qt.IsTrue)
			c.Assert(req.AllowForeignKeyConstraints, qt.IsNotNil)
			c.Assert(*req.AllowForeignKeyConstraints, qt.IsTrue)
			c.Assert(req.AutomaticMigrations, qt.IsNotNil)
			c.Assert(*req.AutomaticMigrations, qt.IsTrue)
			c.Assert(req.MigrationFramework, qt.IsNotNil)
			c.Assert(*req.MigrationFramework, qt.Equals, framework)
			c.Assert(req.MigrationTableName, qt.IsNotNil)
			c.Assert(*req.MigrationTableName, qt.Equals, tableName)
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

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{
		db,
		"--allow-data-branching=true",
		"--allow-foreign-key-constraints=true",
		"--automatic-migrations=true",
		"--migration-framework", framework,
		"--migration-table-name", tableName,
	})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDatabase_UpdateCmd_NoFlags(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.DatabaseService{}

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

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"planetscale"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "at least one settings flag must be provided")
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsFalse)
}
