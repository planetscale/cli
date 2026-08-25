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

func TestBackup_UpdateCmd_SetsProtected(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"
	backup := "mybackup"

	res := &ps.Backup{Name: "foo", Protected: true}

	svc := &mock.BackupsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateBackupRequest) (*ps.Backup, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Backup, qt.Equals, backup)
			c.Assert(req.Protected, qt.IsTrue)
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
				Backups: svc,
			}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{db, branch, backup, "--protected"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBackup_UpdateCmd_SetsProtectedFalse(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"
	backup := "mybackup"

	res := &ps.Backup{Name: "foo", Protected: false}

	svc := &mock.BackupsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateBackupRequest) (*ps.Backup, error) {
			c.Assert(req.Protected, qt.IsFalse)
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
				Backups: svc,
			}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{db, branch, backup, "--protected=false"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBackup_UpdateCmd_RequiresProtectedFlag(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.BackupsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateBackupRequest) (*ps.Backup, error) {
			c.Fatal("Backups.Update should not be called without --protected")
			return nil, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Backups: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main", "backup-id"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "--protected must be provided")
	c.Assert(svc.UpdateFnInvoked, qt.IsFalse)
}
