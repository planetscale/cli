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

func testInvitation() *ps.OrganizationInvitation {
	return &ps.OrganizationInvitation{
		ID:    "inv-1",
		Email: "ada@example.com",
		Role:  "billing-admin",
	}
}

func inviteHelper(svc *mock.OrganizationsService, format printer.Format, buf *bytes.Buffer) *cmdutil.Helper {
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

func TestOrg_InviteListCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		ListInvitationsFn: func(ctx context.Context, req *ps.ListOrganizationInvitationsRequest, opts ...ps.ListOption) ([]*ps.OrganizationInvitation, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			listOpts := &ps.ListOptions{URLValues: &url.Values{}}
			for _, opt := range opts {
				c.Assert(opt(listOpts), qt.IsNil)
			}
			c.Assert(listOpts.URLValues.Get("page"), qt.Equals, "2")
			c.Assert(listOpts.URLValues.Get("per_page"), qt.Equals, "25")
			return []*ps.OrganizationInvitation{testInvitation()}, nil
		},
	}
	cmd := InviteListCmd(inviteHelper(svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"--page", "2", "--per-page", "25"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListInvitationsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"role": "billing-admin"`)
}

func TestOrg_InviteListCmd_EmptyPage(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		ListInvitationsFn: func(ctx context.Context, req *ps.ListOrganizationInvitationsRequest, opts ...ps.ListOption) ([]*ps.OrganizationInvitation, error) {
			return nil, nil
		},
	}
	cmd := InviteListCmd(inviteHelper(svc, printer.Human, &buf))
	cmd.SetArgs([]string{"--page", "4"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "No invitations found on this page.")
	c.Assert(buf.String(), qt.Not(qt.Contains), "No pending invitations")
}

func TestOrg_InviteCreateCmd_CustomRoleSlug(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		CreateInvitationFn: func(ctx context.Context, req *ps.CreateOrganizationInvitationRequest) (*ps.OrganizationInvitation, error) {
			c.Assert(req.Email, qt.Equals, "ada@example.com")
			c.Assert(req.Role, qt.Equals, "billing-admin")
			return testInvitation(), nil
		},
	}
	cmd := InviteCreateCmd(inviteHelper(svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"ada@example.com", "--role", "billing-admin"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CreateInvitationFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"role": "billing-admin"`)
}

func TestOrg_InviteCancelCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		ListInvitationsFn: func(ctx context.Context, req *ps.ListOrganizationInvitationsRequest, opts ...ps.ListOption) ([]*ps.OrganizationInvitation, error) {
			return []*ps.OrganizationInvitation{testInvitation()}, nil
		},
		CancelInvitationFn: func(ctx context.Context, req *ps.CancelOrganizationInvitationRequest) error {
			c.Assert(req.ID, qt.Equals, "inv-1")
			return nil
		},
	}
	cmd := InviteCancelCmd(inviteHelper(svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"inv-1", "--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CancelInvitationFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"result": "invitation canceled"`)
}
