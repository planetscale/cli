package backup

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

func TestBackup_RestoreShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "app"
	branch := "main"
	backup := "bak_123"

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return &ps.Database{Kind: ps.DatabaseEngineNeki}, nil
		},
	}
	backupSvc := &mock.BackupsService{
		GetFn: func(ctx context.Context, req *ps.GetBackupRequest) (*ps.Backup, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Backup, qt.Equals, backup)
			return &ps.Backup{Name: backup}, nil
		},
	}
	profileSvc := &mock.NekiShardConfigurationProfilesService{
		ListFn: func(ctx context.Context, req *ps.ListNekiShardConfigurationProfilesRequest) ([]*ps.NekiShardConfigurationProfile, error) {
			c.Assert(req, qt.DeepEquals, &ps.ListNekiShardConfigurationProfilesRequest{
				Organization: org,
				Database:     db,
				Branch:       branch,
			})
			return []*ps.NekiShardConfigurationProfile{
				{Name: "default", Default: true, ClusterSize: "PS_40", Replicas: 2},
				{Name: "analytics", ClusterSize: "PS_10", Replicas: 0},
			}, nil
		},
	}
	routerSvc := &mock.NekiRoutersService{
		ListFn: func(ctx context.Context, req *ps.ListNekiRoutersRequest) ([]*ps.NekiRouter, error) {
			c.Assert(req, qt.DeepEquals, &ps.ListNekiRoutersRequest{
				Organization: org,
				Database:     db,
				Branch:       branch,
			})
			return []*ps.NekiRouter{
				{
					Name:            "default",
					Default:         true,
					SKU:             &ps.NekiRouterSKU{Name: "NKR_20", DisplayName: "NKR-20"},
					ReplicasPerCell: 1,
				},
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases:                      dbSvc,
				Backups:                        backupSvc,
				NekiShardConfigurationProfiles: profileSvc,
				NekiRouters:                    routerSvc,
			}, nil
		},
	}

	cmd := RestoreCmd(ch)
	cmd.SetArgs([]string{"show", db, branch, backup})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(backupSvc.GetFnInvoked, qt.IsTrue)
	c.Assert(profileSvc.ListFnInvoked, qt.IsTrue)
	c.Assert(routerSvc.ListFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]any{
		"database": db,
		"branch":   branch,
		"backup":   backup,
		"configuration_profiles": []map[string]any{
			{"name": "default", "default": true, "cluster_size": "PS_40", "replicas": float64(2)},
			{"name": "analytics", "default": false, "cluster_size": "PS_10", "replicas": float64(0)},
		},
		"routers": []map[string]any{
			{"name": "default", "default": true, "size": "NKR-20", "replicas_per_cell": float64(1)},
		},
	})
}

func TestBackup_RestoreShowCmd_RejectsNonNeki(t *testing.T) {
	c := qt.New(t)

	backupSvc := &mock.BackupsService{}
	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Kind: ps.DatabaseEnginePostgres}, nil
		},
	}
	format := printer.JSON
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: dbSvc, Backups: backupSvc}, nil
		},
	}

	cmd := RestoreCmd(ch)
	cmd.SetArgs([]string{"show", "app", "main", "bak_123"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, `.*only supported for Neki databases`)
	c.Assert(backupSvc.GetFnInvoked, qt.IsFalse)
}

func TestBackup_RestoreShowCmd_Human(t *testing.T) {
	c := qt.New(t)

	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases: &mock.DatabaseService{
					GetFn: func(context.Context, *ps.GetDatabaseRequest) (*ps.Database, error) {
						return &ps.Database{Kind: ps.DatabaseEngineNeki}, nil
					},
				},
				Backups: &mock.BackupsService{
					GetFn: func(context.Context, *ps.GetBackupRequest) (*ps.Backup, error) {
						return &ps.Backup{Name: "bak_123"}, nil
					},
				},
				NekiShardConfigurationProfiles: &mock.NekiShardConfigurationProfilesService{
					ListFn: func(context.Context, *ps.ListNekiShardConfigurationProfilesRequest) ([]*ps.NekiShardConfigurationProfile, error) {
						return []*ps.NekiShardConfigurationProfile{
							{Name: "default", Default: true, ClusterSize: "PS_40", Replicas: 2},
						}, nil
					},
				},
				NekiRouters: &mock.NekiRoutersService{
					ListFn: func(context.Context, *ps.ListNekiRoutersRequest) ([]*ps.NekiRouter, error) {
						return []*ps.NekiRouter{
							{Name: "default", Default: true, SKU: &ps.NekiRouterSKU{Name: "NKR_20"}, ReplicasPerCell: 1},
						}, nil
					},
				},
			}, nil
		},
	}

	cmd := RestoreCmd(ch)
	cmd.SetArgs([]string{"show", "app", "main", "bak_123"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "live source sizes")
	c.Assert(out.String(), qt.Contains, "CLUSTER SIZE")
	c.Assert(out.String(), qt.Contains, "PS_40")
	c.Assert(out.String(), qt.Contains, "REPLICAS PER CELL")
	c.Assert(out.String(), qt.Contains, "NKR_20")
}
