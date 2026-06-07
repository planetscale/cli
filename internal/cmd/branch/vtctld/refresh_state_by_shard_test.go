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

func TestRefreshStateByShard(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.VtctldService{
		RefreshStateByShardFn: func(ctx context.Context, req *ps.VtctldRefreshStateByShardRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, "commerce")
			c.Assert(req.Shard, qt.Equals, "-")
			c.Assert(req.Cells, qt.DeepEquals, []string{"zone1"})
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

	cmd := RefreshStateByShardCmd(ch)
	cmd.SetArgs([]string{db, branch})
	cmd.Flags().Set("keyspace", "commerce")
	cmd.Flags().Set("shard", "-")
	cmd.Flags().Set("cells", "zone1")

	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.RefreshStateByShardFnInvoked, qt.IsTrue)
}
