package keyspace

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

func TestKeyspace_ReadOnlyRegionsCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	regions := []*ps.ReadOnlyRegionKeyspace{{
		Region:             "us-west",
		ClusterName:        "PS_20",
		ClusterDisplayName: "PS-20",
		Replicas:           2,
	}}
	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "analytics")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.Keyspace, qt.Equals, "events")
			c.Assert(req.Full, qt.IsTrue)
			return &ps.Keyspace{ReadOnlyRegions: regions}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := ReadOnlyRegionsCmd(ch)
	cmd.SetArgs([]string{"analytics", "main", "events"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.JSONEquals, regions)
}
