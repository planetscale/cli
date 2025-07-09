package backup

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

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
	cmd.SetArgs([]string{db, branch, backup, "--cluster-size", "PS-20"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}
