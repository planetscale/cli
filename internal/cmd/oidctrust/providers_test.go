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

func TestOIDCTrust_ProvidersCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"

	providers := []*ps.OIDCTrustProvider{
		{
			Name:           "github",
			DisplayName:    "GitHub Actions",
			Issuer:         "https://token.actions.githubusercontent.com",
			RequiredClaims: []string{"repository", "ref"},
		},
	}

	svc := &mock.OIDCTrustsService{
		ListProvidersFn: func(ctx context.Context, req *ps.ListOIDCTrustProvidersRequest) ([]*ps.OIDCTrustProvider, error) {
			c.Assert(req.Organization, qt.Equals, org)
			return providers, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				OIDCTrusts: svc,
			}, nil
		},
	}

	want := []map[string]any{
		{
			"name":            "github",
			"display_name":    "GitHub Actions",
			"issuer":          "https://token.actions.githubusercontent.com",
			"required_claims": []string{"repository", "ref"},
			"claims":          nil,
		},
	}

	for _, args := range [][]string{{"list"}, {}} {
		buf.Reset()
		cmd := ProvidersCmd(ch)
		cmd.SetArgs(args)
		c.Assert(cmd.Execute(), qt.IsNil, qt.Commentf("args: %v", args))
		c.Assert(buf.String(), qt.JSONEquals, want, qt.Commentf("args: %v", args))
	}
	c.Assert(svc.ListProvidersFnInvoked, qt.IsTrue)
}

func TestOIDCTrust_ProvidersShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OIDCTrustsService{
		ListProvidersFn: func(ctx context.Context, req *ps.ListOIDCTrustProvidersRequest) ([]*ps.OIDCTrustProvider, error) {
			return []*ps.OIDCTrustProvider{
				{Name: "github"},
				{
					Name:           "cursor",
					DisplayName:    "Cursor",
					Issuer:         "https://cursor.com",
					RequiredClaims: []string{"team_id"},
					Claims: []ps.OIDCTrustProviderClaim{
						{Name: "team_id", Required: true, Description: "Cursor team"},
						{Name: "email", Description: "User email"},
					},
				},
			}, nil
		},
	}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OIDCTrusts: svc}, nil
		},
	}

	cmd := ProvidersCmd(ch)
	cmd.SetArgs([]string{"show", "cursor"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.JSONEquals, map[string]any{
		"name":            "cursor",
		"display_name":    "Cursor",
		"issuer":          "https://cursor.com",
		"required_claims": []string{"team_id"},
		"claims": []map[string]any{
			{"name": "team_id", "required": true, "description": "Cursor team"},
			{"name": "email", "required": false, "description": "User email"},
		},
	})

	cmd = ProvidersCmd(ch)
	cmd.SetArgs([]string{"show", "nope"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, `OIDC trust provider .*nope.* is not available in organization .*planetscale.*`)
}
