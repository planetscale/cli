package vtctld

import (
	"bytes"
	"context"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestListTablets(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.VtctldService{
		ListTabletsFn: func(ctx context.Context, req *ps.ListBranchTabletsRequest) ([]*ps.TabletGroup, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)

			// No filter flags set.
			c.Assert(req.Keyspace, qt.Equals, "")
			c.Assert(req.Shard, qt.Equals, "")
			c.Assert(req.TabletType, qt.Equals, "")
			c.Assert(req.TabletAliases, qt.HasLen, 0)

			return []*ps.TabletGroup{}, nil
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

	cmd := ListTabletsCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListTabletsFnInvoked, qt.IsTrue)
}

func TestListTablets_Filters(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.VtctldService{
		ListTabletsFn: func(ctx context.Context, req *ps.ListBranchTabletsRequest) ([]*ps.TabletGroup, error) {
			c.Assert(req.Keyspace, qt.Equals, "commerce")
			c.Assert(req.Shard, qt.Equals, "-80")
			c.Assert(req.TabletType, qt.Equals, "replica")
			c.Assert(req.TabletAliases, qt.DeepEquals, []string{"zone1-0000000100", "zone1-0000000101"})

			return []*ps.TabletGroup{}, nil
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

	cmd := ListTabletsCmd(ch)
	cmd.SetArgs([]string{
		db, branch,
		"--keyspace", "commerce",
		"--shard=-80",
		"--tablet-type", "replica",
		"--tablet-alias", "zone1-0000000100,zone1-0000000101",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListTabletsFnInvoked, qt.IsTrue)
}
