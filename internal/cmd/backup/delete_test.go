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

func TestBackup_DeleteCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"
	backup := "mybackup"

	svc := &mock.BackupsService{
		DeleteFn: func(ctx context.Context, req *ps.DeleteBackupRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Backup, qt.Equals, backup)

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
				Backups: svc,
			}, nil

		},
	}

	cmd := DeleteCmd(ch)
	cmd.SetArgs([]string{db, branch, backup, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)

	res := map[string]string{
		"result": "backup deleted",
		"backup": backup,
		"branch": branch,
	}
	c.Assert(buf.String(), qt.JSONEquals, res)
}
