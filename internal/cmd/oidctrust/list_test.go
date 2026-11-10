package oidctrust

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

func TestOIDCTrust_ListCmd_Empty(t *testing.T) {
	c := qt.New(t)

	svc := &mock.OIDCTrustsService{
		ListFn: func(ctx context.Context, req *ps.ListOIDCTrustsRequest, opts ...ps.ListOption) ([]*ps.OIDCTrust, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			return []*ps.OIDCTrust{}, nil
		},
	}

	run := func(format printer.Format) (string, string) {
		var human, resource bytes.Buffer
		p := printer.NewPrinter(&format)
		p.SetHumanOutput(&human)
		p.SetResourceOutput(&resource)
		ch := &cmdutil.Helper{
			Printer: p,
			Config:  &config.Config{Organization: "planetscale"},
			Client:  func() (*ps.Client, error) { return &ps.Client{OIDCTrusts: svc}, nil },
		}
		cmd := ListCmd(ch)
		cmd.SetArgs([]string{})
		c.Assert(cmd.Execute(), qt.IsNil)
		return human.String(), resource.String()
	}

	human, resource := run(printer.Human)
	c.Assert(human, qt.Contains, "No OIDC trusts exist in organization")
	c.Assert(resource, qt.Equals, "")

	_, resource = run(printer.JSON)
	c.Assert(resource, qt.JSONEquals, []any{})
}
