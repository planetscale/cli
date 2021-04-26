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

func TestSnapshot_DiffCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	id := "123456"
	org := "planetscale"

	res := []*ps.Diff{
		{Name: "foo"},
		{Name: "bar"},
	}

	svc := &mock.SchemaSnapshotsService{
		DiffFn: func(ctx context.Context, req *ps.DiffSchemaSnapshotRequest) ([]*ps.Diff, error) {
			c.Assert(req.ID, qt.Equals, id)
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
				SchemaSnapshots: svc,
			}, nil

		},
	}

	cmd := DiffCmd(ch)
	cmd.SetArgs([]string{id})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DiffFnInvoked, qt.IsTrue)

	c.Assert(buf.String(), qt.JSONEquals, res)
}
