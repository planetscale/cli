package org

import (
	"bytes"
	"context"
	"net/url"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func testCustomRole() *ps.OrganizationCustomRole {
	description := "Can view billing"
	return &ps.OrganizationCustomRole{
		ID:          "role-1",
		Name:        "Billing admin",
		Slug:        "billing-admin",
		Description: &description,
		Permissions: []ps.OrganizationCustomRolePermission{{Action: "show_organization"}},
	}
}

func customRoleHelper(svc *mock.OrganizationsService, format printer.Format, buf *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)
	p.SetHumanOutput(buf)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Organizations: svc}, nil
		},
	}
}

func TestOrg_CustomRoleListCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		ListCustomRolesFn: func(ctx context.Context, req *ps.ListOrganizationCustomRolesRequest, opts ...ps.ListOption) ([]*ps.OrganizationCustomRole, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			listOpts := &ps.ListOptions{URLValues: &url.Values{}}
			for _, opt := range opts {
				c.Assert(opt(listOpts), qt.IsNil)
			}
			c.Assert(listOpts.URLValues.Get("page"), qt.Equals, "2")
			c.Assert(listOpts.URLValues.Get("per_page"), qt.Equals, "25")
			return []*ps.OrganizationCustomRole{testCustomRole()}, nil
		},
	}
	cmd := CustomRoleListCmd(customRoleHelper(svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"--page", "2", "--per-page", "25"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListCustomRolesFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"slug": "billing-admin"`)
}

func TestOrg_CustomRoleListCmd_EmptyPage(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		ListCustomRolesFn: func(ctx context.Context, req *ps.ListOrganizationCustomRolesRequest, opts ...ps.ListOption) ([]*ps.OrganizationCustomRole, error) {
			return nil, nil
		},
	}
	cmd := CustomRoleListCmd(customRoleHelper(svc, printer.Human, &buf))
	cmd.SetArgs([]string{"--page", "4"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "No custom roles found on this page.")
	c.Assert(buf.String(), qt.Not(qt.Contains), "No custom roles in")
}

func TestOrg_CustomRoleShowCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		ListCustomRolesFn: func(ctx context.Context, req *ps.ListOrganizationCustomRolesRequest, opts ...ps.ListOption) ([]*ps.OrganizationCustomRole, error) {
			return []*ps.OrganizationCustomRole{testCustomRole()}, nil
		},
	}
	cmd := CustomRoleShowCmd(customRoleHelper(svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"billing-admin"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListCustomRolesFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"slug": "billing-admin"`)
	c.Assert(buf.String(), qt.Contains, `"name": "Billing admin"`)
}

func TestOrg_CustomRoleShowCmd_NotFound(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		ListCustomRolesFn: func(ctx context.Context, req *ps.ListOrganizationCustomRolesRequest, opts ...ps.ListOption) ([]*ps.OrganizationCustomRole, error) {
			return []*ps.OrganizationCustomRole{testCustomRole()}, nil
		},
	}
	cmd := CustomRoleShowCmd(customRoleHelper(svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"missing"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, `custom role .* does not exist in organization .*`)
}
