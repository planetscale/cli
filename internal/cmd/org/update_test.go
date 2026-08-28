package org

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

func TestOrganization_UpdateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateOrganizationRequest) (*ps.Organization, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.BillingEmail, qt.IsNotNil)
			c.Assert(*req.BillingEmail, qt.Equals, "billing@example.com")
			c.Assert(req.IDPManagedRoles, qt.IsNotNil)
			c.Assert(*req.IDPManagedRoles, qt.IsFalse)
			c.Assert(req.InvoiceBudgetAlerts, qt.IsNotNil)
			c.Assert(*req.InvoiceBudgetAlerts, qt.IsTrue)
			c.Assert(req.InvoiceBudgetAmount, qt.IsNotNil)
			c.Assert(*req.InvoiceBudgetAmount, qt.Equals, int64(2500))
			return &ps.Organization{
				Name:                req.Organization,
				BillingEmail:        *req.BillingEmail,
				IDPManagedRoles:     *req.IDPManagedRoles,
				InvoiceBudgetAlerts: true,
				InvoiceBudgetAmount: "2500",
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Organizations: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{
		"--org", "planetscale",
		"--billing-email", "billing@example.com",
		"--idp-managed-roles=false",
		"--spend-alert-amount", "2500",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &ps.Organization{
		Name:                "planetscale",
		BillingEmail:        "billing@example.com",
		IDPManagedRoles:     false,
		InvoiceBudgetAlerts: true,
		InvoiceBudgetAmount: "2500",
	})
}

func TestOrganization_UpdateCmdDisablesSpendAlert(t *testing.T) {
	c := qt.New(t)

	svc := &mock.OrganizationsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateOrganizationRequest) (*ps.Organization, error) {
			c.Assert(req.InvoiceBudgetAlerts, qt.IsNotNil)
			c.Assert(*req.InvoiceBudgetAlerts, qt.IsFalse)
			c.Assert(req.InvoiceBudgetAmount, qt.IsNil)
			return &ps.Organization{Name: req.Organization, InvoiceBudgetAlerts: false, InvoiceBudgetAmount: "0.0"}, nil
		},
	}

	format := printer.JSON
	p := printer.NewPrinter(&format)
	var buf bytes.Buffer
	p.SetResourceOutput(&buf)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Organizations: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"--org", "planetscale", "--spend-alert=false"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestOrganization_UpdateCmdEnablesSpendAlertFromCurrentAmount(t *testing.T) {
	c := qt.New(t)

	svc := &mock.OrganizationsService{
		GetFn: func(ctx context.Context, req *ps.GetOrganizationRequest) (*ps.Organization, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			return &ps.Organization{
				Name:                         req.Organization,
				InvoiceBudgetAmount:          "0.0",
				SuggestedInvoiceBudgetAmount: "1200",
			}, nil
		},
		UpdateFn: func(ctx context.Context, req *ps.UpdateOrganizationRequest) (*ps.Organization, error) {
			c.Assert(*req.InvoiceBudgetAlerts, qt.IsTrue)
			c.Assert(*req.InvoiceBudgetAmount, qt.Equals, int64(1200))
			return &ps.Organization{
				Name:                req.Organization,
				InvoiceBudgetAlerts: true,
				InvoiceBudgetAmount: "1200",
			}, nil
		},
	}

	format := printer.JSON
	p := printer.NewPrinter(&format)
	var buf bytes.Buffer
	p.SetResourceOutput(&buf)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Organizations: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"--org", "planetscale", "--spend-alert=true"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestOrganization_UpdateCmdRequiresUpdateFlag(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  &config.Config{},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"--org", "planetscale"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, "at least one of --billing-email, --idp-managed-roles, --idp-sso-managed-roles, --spend-alert, or --spend-alert-amount must be provided")
}

func TestOrganization_UpdateCmdRejectsAmountWithDisabledSpendAlert(t *testing.T) {
	c := qt.New(t)

	svc := &mock.OrganizationsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateOrganizationRequest) (*ps.Organization, error) {
			c.Fatal("Organizations.Update should not be called")
			return nil, nil
		},
	}

	format := printer.JSON
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  &config.Config{},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Organizations: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"--org", "planetscale", "--spend-alert=false", "--spend-alert-amount", "2500"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, "--spend-alert-amount cannot be combined with --spend-alert=false")
}

func TestOrganization_UpdateCmdRejectsOutOfRangeSpendAlertAmount(t *testing.T) {
	c := qt.New(t)

	svc := &mock.OrganizationsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateOrganizationRequest) (*ps.Organization, error) {
			c.Fatal("Organizations.Update should not be called")
			return nil, nil
		},
	}

	format := printer.JSON
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  &config.Config{},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Organizations: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"--org", "planetscale", "--spend-alert-amount", "0"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, "--spend-alert-amount must be between 1 and 99999999999")
}

func TestOrganization_UpdateCmdRequiresOrganization(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  &config.Config{},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"--billing-email", "billing@example.com"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, `required flag\(s\) "org" not set`)
}

func TestOrganization_UpdateCmdHuman(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateOrganizationRequest) (*ps.Organization, error) {
			return &ps.Organization{
				Name:                req.Organization,
				BillingEmail:        "billing@example.com",
				IDPManagedRoles:     true,
				InvoiceBudgetAlerts: true,
				InvoiceBudgetAmount: "2500",
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Organizations: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"--org", "planetscale", "--idp-managed-roles=true"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "BILLING EMAIL")
	c.Assert(buf.String(), qt.Contains, "IDP MANAGED ROLES")
	c.Assert(buf.String(), qt.Contains, "SPEND ALERT")
	c.Assert(buf.String(), qt.Contains, "billing@example.com")
}

func TestOrganization_UpdateCmdIDPSSOManagedRoles(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.OrganizationsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateOrganizationRequest) (*ps.Organization, error) {
			c.Assert(req.IDPSSOManagedRoles, qt.IsNotNil)
			c.Assert(*req.IDPSSOManagedRoles, qt.IsTrue)
			return &ps.Organization{Name: req.Organization, IDPSSOManagedRoles: true}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Organizations: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"--org", "planetscale", "--idp-sso-managed-roles=true"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}
