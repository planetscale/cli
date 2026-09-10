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

func maintenanceTestHelper(out *bytes.Buffer, kind ps.DatabaseEngine, svc *mock.BranchMaintenanceService) *cmdutil.Helper {
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(out)

	dbSvc := &mock.DatabaseService{
		GetFn: func(_ context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: req.Database, Kind: kind}, nil
		},
	}

	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: dbSvc, BranchMaintenance: svc}, nil
		},
	}
}

func TestMaintenanceRunCmd(t *testing.T) {
	for _, kind := range []ps.DatabaseEngine{ps.DatabaseEngineNeki, ps.DatabaseEnginePostgres} {
		t.Run(string(kind), func(t *testing.T) {
			c := qt.New(t)
			var out bytes.Buffer
			svc := &mock.BranchMaintenanceService{
				RunFn: func(_ context.Context, req *ps.RunBranchMaintenanceRequest) error {
					c.Assert(req, qt.DeepEquals, &ps.RunBranchMaintenanceRequest{
						Organization: "acme",
						Database:     "app",
						Branch:       "main",
					})
					return nil
				},
			}

			cmd := MaintenanceRunCmd(maintenanceTestHelper(&out, kind, svc))
			cmd.SetArgs([]string{"app", "main"})

			c.Assert(cmd.Execute(), qt.IsNil)
			c.Assert(svc.RunFnInvoked, qt.IsTrue)
			c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
				"result":   "maintenance started",
				"database": "app",
				"branch":   "main",
			})
		})
	}
}

func TestMaintenanceRunCmdUpdatesPostgresMinorVersion(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.BranchMaintenanceService{
		RunFn: func(_ context.Context, req *ps.RunBranchMaintenanceRequest) error {
			c.Assert(req.UpdatePostgresMinorVersion, qt.IsTrue)
			return nil
		},
	}

	cmd := MaintenanceRunCmd(maintenanceTestHelper(&out, ps.DatabaseEnginePostgres, svc))
	cmd.SetArgs([]string{"app", "main", "--update-postgres-minor-version"})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.RunFnInvoked, qt.IsTrue)
}

func TestMaintenanceRunCmdRejectsVitessDatabases(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.BranchMaintenanceService{
		RunFn: func(_ context.Context, _ *ps.RunBranchMaintenanceRequest) error {
			return nil
		},
	}

	cmd := MaintenanceRunCmd(maintenanceTestHelper(&out, ps.DatabaseEngineMySQL, svc))
	cmd.SetArgs([]string{"app", "main"})

	c.Assert(cmd.Execute(), qt.ErrorMatches, `(?s).*only available for Postgres and Neki.*mysql.*`)
	c.Assert(svc.RunFnInvoked, qt.IsFalse)
}

func TestMaintenanceRunCmdRequiresBranch(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.BranchMaintenanceService{}

	cmd := MaintenanceRunCmd(maintenanceTestHelper(&out, ps.DatabaseEngineNeki, svc))
	cmd.SetArgs([]string{"app"})

	c.Assert(cmd.Execute(), qt.IsNotNil)
	c.Assert(svc.RunFnInvoked, qt.IsFalse)
}
