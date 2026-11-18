package token

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func TestServiceToken_UpdateCmdSetsCIDRs(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	id := "token-id"
	cidrs := []string{"10.0.0.0/8", "192.168.1.1/32"}
	secret := "must-not-be-printed"

	svc := &mock.ServiceTokenService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateServiceTokenRequest) (*ps.ServiceToken, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.ID, qt.Equals, id)
			c.Assert(req.CIDRs, qt.IsNotNil)
			c.Assert(*req.CIDRs, qt.DeepEquals, cidrs)
			return &ps.ServiceToken{ID: id, Token: secret, CIDRs: cidrs}, nil
		},
	}

	cmd := UpdateCmd(serviceTokenUpdateHelper(p, org, svc))
	cmd.SetArgs([]string{id, "--cidrs", "10.0.0.0/8,192.168.1.1/32"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]any{
		"id":                         id,
		"name":                       nil,
		"created_at":                 "0001-01-01T00:00:00Z",
		"last_used_at":               nil,
		"last_used_ip":               nil,
		"last_used_asn_organization": nil,
		"expires_at":                 nil,
		"cidrs":                      cidrs,
	})
	c.Assert(buf.String(), qt.Not(qt.Contains), secret)
}

func TestServiceToken_UpdateCmdClearsCIDRs(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.ServiceTokenService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateServiceTokenRequest) (*ps.ServiceToken, error) {
			c.Assert(req.CIDRs, qt.IsNotNil)
			c.Assert(*req.CIDRs, qt.HasLen, 0)
			return &ps.ServiceToken{ID: "token-id"}, nil
		},
	}

	cmd := UpdateCmd(serviceTokenUpdateHelper(p, "planetscale", svc))
	cmd.SetArgs([]string{"token-id", "--cidrs", ""})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestServiceToken_UpdateCmdRequiresCIDRs(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.ServiceTokenService{}

	cmd := UpdateCmd(serviceTokenUpdateHelper(p, "planetscale", svc))
	cmd.SetArgs([]string{"token-id"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "at least one of --name, --ttl, or --cidrs must be provided")
	c.Assert(svc.UpdateFnInvoked, qt.IsFalse)
}

func serviceTokenUpdateHelper(p *printer.Printer, org string, svc *mock.ServiceTokenService) *cmdutil.Helper {
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{ServiceTokens: svc}, nil
		},
	}
}
