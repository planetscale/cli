package keyspace

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestKeyspace_ResizeCancelCmd(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON

	p := printer.NewPrinter(&format)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "sharded"

	svc := &mock.KeyspacesService{
		CancelResizeFn: func(ctx context.Context, req *ps.CancelKeyspaceResizeRequest) error {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)

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
				Keyspaces: svc,
			}, nil
		},
	}

	cmd := ResizeCancelCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CancelResizeFnInvoked, qt.IsTrue)
}
