package snapshot

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestSnapshot_ListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"
	resp := []*ps.SchemaSnapshot{
		{ID: "123"},
		{ID: "345"},
	}

	svc := &mock.SchemaSnapshotsService{
		ListFn: func(ctx context.Context, req *ps.ListSchemaSnapshotsRequest) ([]*ps.SchemaSnapshot, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)

			return resp, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				SchemaSnapshots: svc,
			}, nil

		},
	}

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)

	res := []*SchemaSnapshot{
		{orig: resp[0]},
		{orig: resp[1]},
	}
	c.Assert(buf.String(), qt.JSONEquals, res)
}
