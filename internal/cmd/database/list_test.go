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

func TestDatabase_ListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"

	dbs := []*ps.Database{
		{Name: "foo"},
		{Name: "bar"},
	}

	svc := &mock.DatabaseService{
		ListFn: func(ctx context.Context, req *ps.ListDatabasesRequest) ([]*ps.Database, error) {
			c.Assert(req.Organization, qt.Equals, org)
			return dbs, nil
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

	cmd := ListCmd(ch)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, dbs)
}
