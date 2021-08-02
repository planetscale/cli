package region

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

func TestRegion_ListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"

	regions := []*ps.Region{
		{Name: "Test Region", Slug: "test-region", Enabled: true},
	}

	svc := &mock.OrganizationsService{
		ListRegionsFn: func(ctx context.Context, req *ps.ListOrganizationRegionsRequest) ([]*ps.Region, error) {
			c.Assert(req.Organization, qt.Equals, org)
			return regions, nil
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
	c.Assert(svc.ListRegionsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, regions)
}
