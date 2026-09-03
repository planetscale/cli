package backup

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

func TestBackup_RestoreCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "restore-branch"
	backup := "mybackup"

	res := &ps.DatabaseBranch{Name: "restore-branch"}

	svc := &mock.DatabaseBranchesService{
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Name, qt.Equals, branch)
			c.Assert(req.BackupID, qt.Equals, backup)
			c.Assert(req.ClusterSize, qt.Equals, "PS-20")
			return res, nil
		},
	}

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			return &ps.Database{Kind: "mysql"}, nil
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
				Databases:        dbSvc,
			}, nil
		},
	}

	cmd := RestoreCmd(ch)
	cmd.SetArgs([]string{db, branch, backup, "--cluster-size", "PS-20"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBackup_RestoreCmd_PostgreSQL(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "restore-branch"
	backup := "mybackup"

	res := &ps.PostgresBranch{Name: "restore-branch"}

	svc := &mock.PostgresBranchesService{
		CreateFn: func(ctx context.Context, req *ps.CreatePostgresBranchRequest) (*ps.PostgresBranch, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Name, qt.Equals, branch)
			c.Assert(req.BackupID, qt.Equals, backup)
			c.Assert(req.ClusterName, qt.Equals, "PS-20")
			c.Assert(req.Replicas, qt.IsNotNil)
			c.Assert(*req.Replicas, qt.Equals, 2)
			return res, nil
		},
	}

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			return &ps.Database{Kind: "postgresql"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				PostgresBranches: svc,
				Databases:        dbSvc,
			}, nil
		},
	}

	cmd := RestoreCmd(ch)
	cmd.SetArgs([]string{db, branch, backup, "--cluster-size", "PS-20", "--replicas", "2"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBackup_RestoreCmdRejectsReplicasForMySQL(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)
	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Kind: ps.DatabaseEngineMySQL}, nil
		},
	}
	svc := &mock.DatabaseBranchesService{
		CreateFn: func(ctx context.Context, req *ps.CreateDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			c.Fatal("CreateFn should not be called for MySQL with --replicas")
			return nil, nil
		},
	}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: dbSvc, DatabaseBranches: svc}, nil
		},
	}

	cmd := RestoreCmd(ch)
	cmd.SetArgs([]string{"planetscale", "restored", "backup-id", "--cluster-size", "PS-20", "--replicas", "2"})

	c.Assert(cmd.Execute(), qt.ErrorMatches, ".*--replicas is only supported for PostgreSQL.*")
	c.Assert(svc.CreateFnInvoked, qt.IsFalse)
}
