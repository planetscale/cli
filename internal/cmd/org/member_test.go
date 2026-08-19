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

func testMember() *ps.OrganizationMembership {
	return &ps.OrganizationMembership{
		ID:   "mem-1",
		Role: "member",
		User: ps.OrganizationMemberUser{
			ID:          "user-1",
			Name:        "Ada",
			DisplayName: "Ada Lovelace",
			Email:       "ada@example.com",
		},
	}
}

func TestOrg_MemberListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationsService{
		ListMembersFn: func(ctx context.Context, req *ps.ListOrganizationMembersRequest, opts ...ps.ListOption) ([]*ps.OrganizationMembership, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Query, qt.Equals, "ada")
			return []*ps.OrganizationMembership{testMember()}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Organizations: svc}, nil
		},
	}

	cmd := MemberListCmd(ch)
	cmd.SetArgs([]string{"--query", "ada"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListMembersFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "ada@example.com")
}

func TestOrg_MemberShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationsService{
		GetMemberFn: func(ctx context.Context, req *ps.GetOrganizationMemberRequest) (*ps.OrganizationMembership, error) {
			c.Assert(req.UserID, qt.Equals, "user-1")
			return testMember(), nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Organizations: svc}, nil
		},
	}

	cmd := MemberShowCmd(ch)
	cmd.SetArgs([]string{"user-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetMemberFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "user-1")
}

func TestOrg_MemberShowCmd_ResolveEmail(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationsService{
		GetMemberFn: func(ctx context.Context, req *ps.GetOrganizationMemberRequest) (*ps.OrganizationMembership, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
		ListMembersFn: func(ctx context.Context, req *ps.ListOrganizationMembersRequest, opts ...ps.ListOption) ([]*ps.OrganizationMembership, error) {
			c.Assert(req.Query, qt.Equals, "ada@example.com")
			return []*ps.OrganizationMembership{testMember()}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Organizations: svc}, nil
		},
	}

	cmd := MemberShowCmd(ch)
	cmd.SetArgs([]string{"ada@example.com"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListMembersFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "ada@example.com")
}

func TestOrg_MemberUpdateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationsService{
		ListMembersFn: func(ctx context.Context, req *ps.ListOrganizationMembersRequest, opts ...ps.ListOption) ([]*ps.OrganizationMembership, error) {
			return []*ps.OrganizationMembership{testMember()}, nil
		},
		UpdateMemberFn: func(ctx context.Context, req *ps.UpdateOrganizationMemberRequest) (*ps.OrganizationMembership, error) {
			c.Assert(req.UserID, qt.Equals, "user-1")
			c.Assert(req.Role, qt.Equals, "admin")
			m := testMember()
			m.Role = "admin"
			return m, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Organizations: svc}, nil
		},
	}

	cmd := MemberUpdateCmd(ch)
	cmd.SetArgs([]string{"user-1", "--role", "admin"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateMemberFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"role": "admin"`)
}

func TestOrg_MemberRemoveCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationsService{
		ListMembersFn: func(ctx context.Context, req *ps.ListOrganizationMembersRequest, opts ...ps.ListOption) ([]*ps.OrganizationMembership, error) {
			return []*ps.OrganizationMembership{testMember()}, nil
		},
		RemoveMemberFn: func(ctx context.Context, req *ps.RemoveOrganizationMemberRequest) error {
			c.Assert(req.UserID, qt.Equals, "user-1")
			c.Assert(req.DeletePasswords, qt.IsTrue)
			return nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Organizations: svc}, nil
		},
	}

	cmd := MemberRemoveCmd(ch)
	cmd.SetArgs([]string{"user-1", "--force", "--delete-passwords"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.RemoveMemberFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "member removed")
}
