package org

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

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
		GetFn: func(ctx context.Context, req *ps.OrganizationSSORequest) (*ps.OrganizationSSO, error) {
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
	c.Assert(buf.String(), qt.Contains, "pscale org sso configure --org planetscale --format json")
}

func TestOrg_SSOEnableCmd(t *testing.T) {
	c := qt.New(t)

	oldOpen := openSSOBrowser
	c.Cleanup(func() { openSSOBrowser = oldOpen })
	openSSOBrowser = func(_ string, url string) error {
		t.Fatalf("unexpected browser open in JSON mode: %s", url)
		return nil
	}

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		EnableFn: func(ctx context.Context, req *ps.OrganizationSSORequest) (*ps.OrganizationSSO, error) {
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
	c.Assert(buf.String(), qt.Contains, "domain_verification_url")
}

func TestOrg_SSODisableCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		DisableFn: func(ctx context.Context, req *ps.OrganizationSSORequest) (*ps.OrganizationSSO, error) {
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
		ConfigureFn: func(ctx context.Context, req *ps.OrganizationSSORequest) (*ps.SSOPortal, error) {
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
	c.Assert(buf.String(), qt.Contains, "--idp-sso-managed-roles=true")
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
		EnableDirectoryFn: func(ctx context.Context, req *ps.OrganizationSSORequest) (*ps.SSOPortal, error) {
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
	c.Assert(buf.String(), qt.Contains, "--idp-managed-roles=true")
}

func TestOrg_SSODirectoryDisableCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		DisableDirectoryFn: func(ctx context.Context, req *ps.OrganizationSSORequest) (*ps.OrganizationSSO, error) {
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
		ListDomainsFn: func(ctx context.Context, req *ps.OrganizationSSORequest) ([]*ps.OrganizationDomain, error) {
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

func TestOrg_SSODomainVerifyCmd(t *testing.T) {
	c := qt.New(t)

	oldOpen := openSSOBrowser
	c.Cleanup(func() { openSSOBrowser = oldOpen })
	openSSOBrowser = func(_ string, url string) error { return nil }

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		VerifyDomainFn: func(ctx context.Context, req *ps.OrganizationSSORequest) (*ps.SSOPortal, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			return &ps.SSOPortal{PortalURL: "https://portal.example/verify"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSODomainVerifyCmd(ch)
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.VerifyDomainFnInvoked, qt.IsTrue)
	c.Assert(svc.ListDomainsFnInvoked, qt.IsFalse)
	c.Assert(buf.String(), qt.Contains, `"portal_url": "https://portal.example/verify"`)
	c.Assert(buf.String(), qt.Contains, "domain show <domain-id>")
}

func TestOrg_SSODomainShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		GetDomainFn: func(ctx context.Context, req *ps.OrganizationSSODomainRequest) (*ps.OrganizationDomain, error) {
			c.Assert(req.DomainID, qt.Equals, "dom_1")
			return &ps.OrganizationDomain{ID: "dom_1", Domain: "example.com", State: "pending"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSODomainShowCmd(ch)
	cmd.SetArgs([]string{"dom_1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetDomainFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"state": "pending"`)
	c.Assert(buf.String(), qt.Contains, "domain show dom_1")
}

func TestOrg_SSODomainVerifyCmdWait(t *testing.T) {
	c := qt.New(t)

	oldInterval := ssoDomainPollInterval
	ssoDomainPollInterval = time.Millisecond
	c.Cleanup(func() { ssoDomainPollInterval = oldInterval })

	oldOpen := openSSOBrowser
	c.Cleanup(func() { openSSOBrowser = oldOpen })
	openSSOBrowser = func(_ string, url string) error { return nil }

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	listCalls := 0
	svc := &mock.OrganizationSSOService{
		ListDomainsFn: func(ctx context.Context, req *ps.OrganizationSSORequest) ([]*ps.OrganizationDomain, error) {
			listCalls++
			if listCalls == 1 {
				return nil, nil
			}
			return []*ps.OrganizationDomain{{ID: "dom_1", Domain: "example.com", State: "pending"}}, nil
		},
		VerifyDomainFn: func(ctx context.Context, req *ps.OrganizationSSORequest) (*ps.SSOPortal, error) {
			return &ps.SSOPortal{PortalURL: "https://portal.example/verify"}, nil
		},
		GetDomainFn: func(ctx context.Context, req *ps.OrganizationSSODomainRequest) (*ps.OrganizationDomain, error) {
			c.Assert(req.DomainID, qt.Equals, "dom_1")
			return &ps.OrganizationDomain{ID: "dom_1", Domain: "example.com", State: "verified"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSODomainVerifyCmd(ch)
	cmd.SetArgs([]string{"--wait"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.VerifyDomainFnInvoked, qt.IsTrue)
	c.Assert(svc.GetDomainFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"state": "verified"`)
	c.Assert(buf.String(), qt.Contains, "example.com")
}

func TestOrg_SSODomainVerifyCmdWaitUsesLatestNewDomain(t *testing.T) {
	c := qt.New(t)

	oldInterval := ssoDomainPollInterval
	ssoDomainPollInterval = time.Millisecond
	c.Cleanup(func() { ssoDomainPollInterval = oldInterval })

	oldOpen := openSSOBrowser
	c.Cleanup(func() { openSSOBrowser = oldOpen })
	openSSOBrowser = func(_ string, url string) error { return nil }

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	old := &ps.OrganizationDomain{ID: "dom_old", Domain: "old.com", State: "pending"}
	newer := &ps.OrganizationDomain{
		ID:        "dom_new",
		Domain:    "new.com",
		State:     "pending",
		CreatedAt: time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC),
	}
	oldestNew := &ps.OrganizationDomain{
		ID:        "dom_mid",
		Domain:    "mid.com",
		State:     "pending",
		CreatedAt: time.Date(2026, 8, 28, 11, 0, 0, 0, time.UTC),
	}

	listCalls := 0
	svc := &mock.OrganizationSSOService{
		ListDomainsFn: func(ctx context.Context, req *ps.OrganizationSSORequest) ([]*ps.OrganizationDomain, error) {
			listCalls++
			if listCalls == 1 {
				return []*ps.OrganizationDomain{old}, nil
			}
			return []*ps.OrganizationDomain{old, oldestNew, newer}, nil
		},
		VerifyDomainFn: func(ctx context.Context, req *ps.OrganizationSSORequest) (*ps.SSOPortal, error) {
			return &ps.SSOPortal{PortalURL: "https://portal.example/verify"}, nil
		},
		GetDomainFn: func(ctx context.Context, req *ps.OrganizationSSODomainRequest) (*ps.OrganizationDomain, error) {
			c.Assert(req.DomainID, qt.Equals, "dom_new")
			return &ps.OrganizationDomain{ID: "dom_new", Domain: "new.com", State: "verified"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSODomainVerifyCmd(ch)
	cmd.SetArgs([]string{"--wait"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "new.com")
}

func TestOrg_SSODomainVerifyCmdWaitPollsShowForNewID(t *testing.T) {
	c := qt.New(t)

	oldInterval := ssoDomainPollInterval
	ssoDomainPollInterval = time.Millisecond
	c.Cleanup(func() { ssoDomainPollInterval = oldInterval })

	oldOpen := openSSOBrowser
	c.Cleanup(func() { openSSOBrowser = oldOpen })
	openSSOBrowser = func(_ string, url string) error { return nil }

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	listCalls := 0
	svc := &mock.OrganizationSSOService{
		ListDomainsFn: func(ctx context.Context, req *ps.OrganizationSSORequest) ([]*ps.OrganizationDomain, error) {
			listCalls++
			if listCalls == 1 {
				return nil, nil
			}
			return []*ps.OrganizationDomain{{ID: "dom_1", Domain: "example.com", State: "bogus"}}, nil
		},
		VerifyDomainFn: func(ctx context.Context, req *ps.OrganizationSSORequest) (*ps.SSOPortal, error) {
			return &ps.SSOPortal{PortalURL: "https://portal.example/verify"}, nil
		},
		GetDomainFn: func(ctx context.Context, req *ps.OrganizationSSODomainRequest) (*ps.OrganizationDomain, error) {
			c.Assert(req.DomainID, qt.Equals, "dom_1")
			return &ps.OrganizationDomain{ID: "dom_1", Domain: "example.com", State: "verified"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSODomainVerifyCmd(ch)
	cmd.SetArgs([]string{"--wait"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetDomainFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"state": "verified"`)
}

func TestOrg_SSODomainVerifyCmdWaitStopsOnPermissionError(t *testing.T) {
	c := qt.New(t)

	oldInterval := ssoDomainPollInterval
	ssoDomainPollInterval = time.Millisecond
	c.Cleanup(func() { ssoDomainPollInterval = oldInterval })

	oldOpen := openSSOBrowser
	c.Cleanup(func() { openSSOBrowser = oldOpen })
	openSSOBrowser = func(_ string, url string) error { return nil }

	format := printer.JSON
	p := printer.NewPrinter(&format)

	listCalls := 0
	svc := &mock.OrganizationSSOService{
		ListDomainsFn: func(ctx context.Context, req *ps.OrganizationSSORequest) ([]*ps.OrganizationDomain, error) {
			listCalls++
			if listCalls == 1 {
				return nil, nil
			}
			return nil, &ps.Error{Code: ps.ErrPermission}
		},
		VerifyDomainFn: func(ctx context.Context, req *ps.OrganizationSSORequest) (*ps.SSOPortal, error) {
			return &ps.SSOPortal{PortalURL: "https://portal.example/verify"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OrganizationSSO: svc}, nil
		},
	}

	cmd := SSODomainVerifyCmd(ch)
	cmd.SetArgs([]string{"--wait", "--wait-timeout", "2s"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, ".*")
}

func TestOrg_SSODomainDeleteCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationSSOService{
		DeleteDomainFn: func(ctx context.Context, req *ps.OrganizationSSODomainRequest) error {
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
	c.Assert(buf.String(), qt.Contains, "pscale org sso domain list --org planetscale --format json")
}

func TestSSOResourceNextSteps(t *testing.T) {
	c := qt.New(t)
	org := "acme"

	c.Assert(ssoResourceNextSteps(org, &ps.OrganizationSSO{}), qt.DeepEquals, []string{
		"pscale org sso enable --org acme --format json",
	})
	c.Assert(ssoResourceNextSteps(org, &ps.OrganizationSSO{Enabled: true}), qt.DeepEquals, []string{
		"pscale org sso domain verify --org acme --format json --wait",
	})
	c.Assert(ssoResourceNextSteps(org, &ps.OrganizationSSO{Enabled: true, HasVerifiedDomain: true}), qt.DeepEquals, []string{
		"pscale org sso configure --org acme --format json",
	})
	c.Assert(ssoResourceNextSteps(org, &ps.OrganizationSSO{Enabled: true, HasVerifiedDomain: true, Configured: true}), qt.DeepEquals, []string{
		"pscale org sso directory enable --org acme --format json",
		"pscale org update --org acme --format json --idp-sso-managed-roles=true",
	})
	c.Assert(ssoResourceNextSteps(org, &ps.OrganizationSSO{Enabled: true, HasVerifiedDomain: true, Configured: true, Directory: true}), qt.DeepEquals, []string{
		"pscale org update --org acme --format json --idp-managed-roles=true",
	})
}
