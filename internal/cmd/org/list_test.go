package org

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

func TestOrganization_ListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"

	svc := &mock.OrganizationsService{
		ListFn: func(ctx context.Context) ([]*ps.Organization, error) {
			return []*ps.Organization{
				{Name: "foo"},
				{Name: "bar"},
			}, nil
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
	c.Assert(svc.ListFnInvoked, qt.IsTrue)

	orgs := []*organization{
		{Name: "foo"},
		{Name: "bar"},
	}
	c.Assert(buf.String(), qt.JSONEquals, orgs)
}
