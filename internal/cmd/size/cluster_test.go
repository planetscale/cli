package size

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/testutil"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestSizeCluster_ListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"

	orig := []*ps.ClusterSKU{
		{Name: "PS-10", Enabled: true, Rate: testutil.Pointer[int64](39)},
	}
	svc := &mock.OrganizationsService{
		ListClusterSKUsFn: func(ctx context.Context, req *ps.ListOrganizationClusterSKUsRequest, opts ...ps.ListOption) ([]*ps.ClusterSKU, error) {
			c.Assert(req.Organization, qt.Equals, org)
			return orig, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Organizations: svc,
			}, nil
		},
	}

	cmd := ListCmd(ch)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListClusterSKUsFnInvoked, qt.IsTrue)

	res := []*ClusterSKU{
		{orig: orig[0]},
	}

	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestSizeCluster_ListCmd_PostgreSQL(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"

	orig := []*ps.ClusterSKU{
		{Name: "PS-10", Enabled: true, Rate: testutil.Pointer[int64](39)},
	}
	svc := &mock.OrganizationsService{
		ListClusterSKUsFn: func(ctx context.Context, req *ps.ListOrganizationClusterSKUsRequest, opts ...ps.ListOption) ([]*ps.ClusterSKU, error) {
			c.Assert(req.Organization, qt.Equals, org)
			return orig, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Organizations: svc,
			}, nil
		},
	}

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{"--engine", "postgresql"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListClusterSKUsFnInvoked, qt.IsTrue)

	res := []*ClusterSKU{
		{orig: orig[0]},
	}

	c.Assert(buf.String(), qt.JSONEquals, res)
}
