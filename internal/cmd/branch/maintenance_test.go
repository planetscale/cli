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

func TestBranch_MaintenanceRunCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: db, Kind: ps.DatabaseEnginePostgres}, nil
		},
	}
	svc := &mock.BranchMaintenanceService{
		RunFn: func(ctx context.Context, req *ps.RunBranchMaintenanceRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.UpdatePostgresMinorVersion, qt.IsFalse)

			return nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases:         dbSvc,
				BranchMaintenance: svc,
			}, nil
		},
	}

	cmd := MaintenanceRunCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.RunFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]string{
		"result": "maintenance started",
		"branch": branch,
	})
}

func TestBranch_MaintenanceRunCmd_UpdateMinorVersion(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: req.Database, Kind: ps.DatabaseEnginePostgres}, nil
		},
	}
	svc := &mock.BranchMaintenanceService{
		RunFn: func(ctx context.Context, req *ps.RunBranchMaintenanceRequest) error {
			c.Assert(req.UpdatePostgresMinorVersion, qt.IsTrue)
			return nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: "planetscale",
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases:         dbSvc,
				BranchMaintenance: svc,
			}, nil
		},
	}

	cmd := MaintenanceRunCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main", "--update-postgres-minor-version"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.RunFnInvoked, qt.IsTrue)
}

func TestBranch_MaintenanceRunCmd_VitessDatabase(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: req.Database, Kind: ps.DatabaseEngineMySQL}, nil
		},
	}
	svc := &mock.BranchMaintenanceService{
		RunFn: func(ctx context.Context, req *ps.RunBranchMaintenanceRequest) error {
			return nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: "planetscale",
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases:         dbSvc,
				BranchMaintenance: svc,
			}, nil
		},
	}

	cmd := MaintenanceRunCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*only available for PostgreSQL.*mysql.*`)
	c.Assert(svc.RunFnInvoked, qt.IsFalse)
}
