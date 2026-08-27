package org

import (
	"bytes"
	"context"
	"errors"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func testSSO() *ps.OrganizationSSO {
	url := "https://portal.example/verify"
	return &ps.OrganizationSSO{
		ID:                    "org_1",
		Type:                  "OrganizationSSO",
		Enabled:               true,
		Configured:            false,
		Directory:             false,
		HasVerifiedDomain:     true,
		DomainVerificationURL: &url,
		Domains: []*ps.OrganizationDomain{{
			ID:     "dom_1",
			Type:   "OrganizationDomain",
			Domain: "example.com",
			State:  "verified",
		}},
	}
}

func TestOrg_SSOShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		GetFn: func(ctx context.Context, req *ps.GetOrganizationSSORequest) (*ps.OrganizationSSO, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			return testSSO(), nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSOShowCmd(ch)
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"enabled": true`)
	c.Assert(buf.String(), qt.Contains, "example.com")
}

func TestOrg_SSOEnableCmd(t *testing.T) {
	c := qt.New(t)

	oldOpen := openSSOBrowser
	c.Cleanup(func() { openSSOBrowser = oldOpen })
	var opened string
	openSSOBrowser = func(_ string, url string) error {
		opened = url
		return nil
	}

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		EnableFn: func(ctx context.Context, req *ps.EnableOrganizationSSORequest) (*ps.OrganizationSSO, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			return testSSO(), nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSOEnableCmd(ch)
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.EnableFnInvoked, qt.IsTrue)
	c.Assert(opened, qt.Equals, "https://portal.example/verify")
	c.Assert(buf.String(), qt.Contains, "domain_verification_url")
}

func TestOrg_SSODisableCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		DisableFn: func(ctx context.Context, req *ps.DisableOrganizationSSORequest) (*ps.OrganizationSSO, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			sso := testSSO()
			sso.Enabled = false
			sso.DomainVerificationURL = nil
			return sso, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSODisableCmd(ch)
	cmd.SetArgs([]string{"--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.DisableFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"enabled": false`)
}

func TestOrg_SSODisableCmd_RequiresForce(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: &mock.OrganizationSSOService{}}, nil
		},
	}

	cmd := SSODisableCmd(ch)
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "--force")
}

func TestOrg_SSOConfigureCmd(t *testing.T) {
	c := qt.New(t)

	oldOpen := openSSOBrowser
	c.Cleanup(func() { openSSOBrowser = oldOpen })
	var opened string
	openSSOBrowser = func(_ string, url string) error {
		opened = url
		return nil
	}

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		ConfigureFn: func(ctx context.Context, req *ps.ConfigureOrganizationSSORequest) (*ps.SSOPortal, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			return &ps.SSOPortal{PortalURL: "https://portal.example/sso"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSOConfigureCmd(ch)
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ConfigureFnInvoked, qt.IsTrue)
	c.Assert(opened, qt.Equals, "https://portal.example/sso")
	c.Assert(buf.String(), qt.Contains, `"portal_url": "https://portal.example/sso"`)
	c.Assert(buf.String(), qt.Contains, `"browser_opened": true`)
}

func TestOrg_SSODirectoryEnableCmd(t *testing.T) {
	c := qt.New(t)

	oldOpen := openSSOBrowser
	c.Cleanup(func() { openSSOBrowser = oldOpen })
	openSSOBrowser = func(_ string, url string) error { return errors.New("no browser") }

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		EnableDirectoryFn: func(ctx context.Context, req *ps.EnableOrganizationSSODirectoryRequest) (*ps.SSOPortal, error) {
			return &ps.SSOPortal{PortalURL: "https://portal.example/dsync"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSODirectoryEnableCmd(ch)
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.EnableDirectoryFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"browser_opened": false`)
}

func TestOrg_SSODirectoryDisableCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		DisableDirectoryFn: func(ctx context.Context, req *ps.DisableOrganizationSSODirectoryRequest) (*ps.OrganizationSSO, error) {
			sso := testSSO()
			sso.Directory = false
			return sso, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSODirectoryDisableCmd(ch)
	cmd.SetArgs([]string{"--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.DisableDirectoryFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"directory": false`)
}

func TestOrg_SSODomainListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		ListDomainsFn: func(ctx context.Context, req *ps.ListOrganizationSSODomainsRequest) ([]*ps.OrganizationDomain, error) {
			return testSSO().Domains, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSODomainListCmd(ch)
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListDomainsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "example.com")
}

func TestOrg_SSODomainDeleteCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		DeleteDomainFn: func(ctx context.Context, req *ps.DeleteOrganizationSSODomainRequest) error {
			c.Assert(req.DomainID, qt.Equals, "dom_1")
			return nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSODomainDeleteCmd(ch)
	cmd.SetArgs([]string{"dom_1", "--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.DeleteDomainFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "domain deleted")
}
