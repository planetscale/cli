package vtctld

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestSetShardTabletControl(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.VtctldService{
		SetShardTabletControlFn: func(ctx context.Context, req *ps.VtctldSetShardTabletControlRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, "commerce")
			c.Assert(req.Shard, qt.Equals, "-")
			c.Assert(req.TabletType, qt.Equals, "rdonly")
			c.Assert(req.DeniedTables, qt.DeepEquals, []string{"customers"})
			c.Assert(req.Remove, qt.Not(qt.IsNil))
			c.Assert(*req.Remove, qt.Equals, true)
			return json.RawMessage(`{}`), nil
		},
	}

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Vtctld: svc,
			}, nil
		},
	}

	cmd := SetShardTabletControlCmd(ch)
	cmd.SetArgs([]string{db, branch})
	cmd.Flags().Set("keyspace", "commerce")
	cmd.Flags().Set("shard", "-")
	cmd.Flags().Set("tablet-type", "rdonly")
	cmd.Flags().Set("denied-tables", "customers")
	cmd.Flags().Set("remove", "true")

	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.SetShardTabletControlFnInvoked, qt.IsTrue)
}
