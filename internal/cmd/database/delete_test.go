package database

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

func TestDatabase_DeleteCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"

	res := map[string]string{
		"result":   "database deleted",
		"database": db,
	}

	svc := &mock.DatabaseService{
		DeleteFn: func(ctx context.Context, req *ps.DeleteDatabaseRequest) (*ps.DatabaseDeletionRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return nil, nil
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

	cmd := DeleteCmd(ch)
	cmd.SetArgs([]string{db, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDatabase_DeleteCmdWithDeletionRequest(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"

	svc := &mock.DatabaseService{
		DeleteFn: func(ctx context.Context, req *ps.DeleteDatabaseRequest) (*ps.DatabaseDeletionRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return &ps.DatabaseDeletionRequest{
				ID: "test-planetscale-id",
			}, nil
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

	cmd := DeleteCmd(ch)
	cmd.SetArgs([]string{db, "--force"})
	err := cmd.Execute()

	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)
	c.Assert(err.Error(), qt.Equals, "A deletion request for database planetscale was successfully created. Database will be deleted after another database administrator also requests deletion.")
}
