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

	cmd := ProvidersCmd(ch)
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListProvidersFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, []map[string]any{
		{
			"name":            "github",
			"display_name":    "GitHub Actions",
			"issuer":          "https://token.actions.githubusercontent.com",
			"required_claims": []string{"repository", "ref"},
		},
	})
}
