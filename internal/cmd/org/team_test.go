package org

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func testTeam() *ps.OrganizationTeam {
	description := "Platform team"
	return &ps.OrganizationTeam{
		ID:          "team-1",
		DisplayName: "Platform",
		Name:        "Platform",
		Slug:        "platform",
		Description: &description,
		Managed:     false,
	}
}

func testTeamMember() *ps.OrganizationTeamMembership {
	return &ps.OrganizationTeamMembership{
		ID: "membership-1",
		User: ps.OrganizationTeamUser{
			ID:          "user-1",
			DisplayName: "Ada Lovelace",
			Name:        "Ada",
			Email:       "ada@example.com",
		},
		Passwords: []json.RawMessage{json.RawMessage(`{"id":"password-1","name":"production"}`)},
	}
}

func teamHelper(svc *mock.OrganizationsService, format printer.Format, buf *bytes.Buffer) *cmdutil.Helper {
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

func getTestTeam(_ context.Context, req *ps.GetOrganizationTeamRequest) (*ps.OrganizationTeam, error) {
	if req.Team == "platform" || req.Team == "team-1" {
		return testTeam(), nil
	}
	return nil, &ps.Error{Code: ps.ErrNotFound}
}

func TestOrg_TeamListCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		ListTeamsFn: func(ctx context.Context, req *ps.ListOrganizationTeamsRequest, opts ...ps.ListOption) ([]*ps.OrganizationTeam, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Query, qt.Equals, "plat")
			listOpts := &ps.ListOptions{URLValues: &url.Values{}}
			for _, opt := range opts {
				c.Assert(opt(listOpts), qt.IsNil)
			}
			c.Assert(listOpts.URLValues.Get("page"), qt.Equals, "2")
			c.Assert(listOpts.URLValues.Get("per_page"), qt.Equals, "25")
			return []*ps.OrganizationTeam{testTeam()}, nil
		},
	}
	cmd := TeamListCmd(teamHelper(svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"--query", "plat", "--page", "2", "--per-page", "25"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.Contains, `"slug": "platform"`)
}

func TestOrg_TeamListCmd_EmptyPage(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		ListTeamsFn: func(ctx context.Context, req *ps.ListOrganizationTeamsRequest, opts ...ps.ListOption) ([]*ps.OrganizationTeam, error) {
			return nil, nil
		},
	}
	cmd := TeamListCmd(teamHelper(svc, printer.Human, &buf))
	cmd.SetArgs([]string{"--page", "4"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "No teams found on this page.")
	c.Assert(buf.String(), qt.Not(qt.Contains), "No teams in")
}

func TestOrg_TeamShowCmd_ResolvesName(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		GetTeamFn: getTestTeam,
		ListTeamsFn: func(ctx context.Context, req *ps.ListOrganizationTeamsRequest, opts ...ps.ListOption) ([]*ps.OrganizationTeam, error) {
			c.Assert(req.Query == "Platform Team" || req.Query == "Platform" || req.Query == "", qt.IsTrue)
			return []*ps.OrganizationTeam{testTeam()}, nil
		},
	}
	cmd := TeamShowCmd(teamHelper(svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"Platform Team"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, `team .* does not exist in organization .*`)

	cmd = TeamShowCmd(teamHelper(svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"Platform"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListTeamsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"id": "team-1"`)
}

func TestOrg_TeamCreateAndUpdateCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		GetTeamFn: getTestTeam,
		CreateTeamFn: func(ctx context.Context, req *ps.CreateOrganizationTeamRequest) (*ps.OrganizationTeam, error) {
			c.Assert(req.Name, qt.Equals, "Platform")
			c.Assert(req.Description, qt.Equals, "Owns the platform")
			return testTeam(), nil
		},
		UpdateTeamFn: func(ctx context.Context, req *ps.UpdateOrganizationTeamRequest) (*ps.OrganizationTeam, error) {
			c.Assert(req.Team, qt.Equals, "platform")
			c.Assert(req.Name, qt.IsNil)
			c.Assert(req.Description, qt.IsNotNil)
			c.Assert(*req.Description, qt.Equals, "")
			return testTeam(), nil
		},
	}
	ch := teamHelper(svc, printer.JSON, &buf)
	create := TeamCreateCmd(ch)
	create.SetArgs([]string{"--name", "Platform", "--description", "Owns the platform"})
	c.Assert(create.Execute(), qt.IsNil)
	c.Assert(svc.CreateTeamFnInvoked, qt.IsTrue)

	update := TeamUpdateCmd(ch)
	update.SetArgs([]string{"platform", "--description", ""})
	c.Assert(update.Execute(), qt.IsNil)
	c.Assert(svc.UpdateTeamFnInvoked, qt.IsTrue)
}

func TestOrg_TeamDeleteCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		GetTeamFn: getTestTeam,
		DeleteTeamFn: func(ctx context.Context, req *ps.DeleteOrganizationTeamRequest) error {
			c.Assert(req.Team, qt.Equals, "platform")
			return nil
		},
	}
	cmd := TeamDeleteCmd(teamHelper(svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"team-1", "--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.DeleteTeamFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"result": "team deleted"`)
}

func TestOrg_TeamMemberListCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		GetTeamFn: getTestTeam,
		ListTeamMembersFn: func(ctx context.Context, req *ps.ListOrganizationTeamMembersRequest, opts ...ps.ListOption) ([]*ps.OrganizationTeamMembership, error) {
			c.Assert(req.Team, qt.Equals, "platform")
			listOpts := &ps.ListOptions{URLValues: &url.Values{}}
			for _, opt := range opts {
				c.Assert(opt(listOpts), qt.IsNil)
			}
			c.Assert(listOpts.URLValues.Get("page"), qt.Equals, "3")
			c.Assert(listOpts.URLValues.Get("per_page"), qt.Equals, "50")
			return []*ps.OrganizationTeamMembership{testTeamMember()}, nil
		},
	}
	cmd := TeamMemberListCmd(teamHelper(svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"platform", "--page", "3", "--per-page", "50"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.Contains, `"email": "ada@example.com"`)
	c.Assert(buf.String(), qt.Not(qt.Contains), `"passwords"`)
	c.Assert(buf.String(), qt.Not(qt.Contains), `"password-1"`)
}

func TestOrg_TeamMemberListCmd_EmptyPage(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		GetTeamFn: getTestTeam,
		ListTeamMembersFn: func(ctx context.Context, req *ps.ListOrganizationTeamMembersRequest, opts ...ps.ListOption) ([]*ps.OrganizationTeamMembership, error) {
			return nil, nil
		},
	}
	cmd := TeamMemberListCmd(teamHelper(svc, printer.Human, &buf))
	cmd.SetArgs([]string{"platform", "--page", "4"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "No team members found on this page.")
	c.Assert(buf.String(), qt.Not(qt.Contains), "No members in team")
}

func TestOrg_TeamMemberAddCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		GetTeamFn: getTestTeam,
		ListMembersFn: func(ctx context.Context, req *ps.ListOrganizationMembersRequest, opts ...ps.ListOption) ([]*ps.OrganizationMembership, error) {
			return []*ps.OrganizationMembership{testMember()}, nil
		},
		AddTeamMemberFn: func(ctx context.Context, req *ps.AddOrganizationTeamMemberRequest) (*ps.OrganizationTeamMembership, error) {
			c.Assert(req.Team, qt.Equals, "platform")
			c.Assert(req.UserID, qt.Equals, "user-1")
			return testTeamMember(), nil
		},
	}
	cmd := TeamMemberAddCmd(teamHelper(svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"platform", "ada@example.com"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.AddTeamMemberFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"id": "membership-1"`)
}

func TestOrg_TeamMemberRemoveCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.OrganizationsService{
		GetTeamFn: getTestTeam,
		ListTeamMembersFn: func(ctx context.Context, req *ps.ListOrganizationTeamMembersRequest, opts ...ps.ListOption) ([]*ps.OrganizationTeamMembership, error) {
			return []*ps.OrganizationTeamMembership{testTeamMember()}, nil
		},
		RemoveTeamMemberFn: func(ctx context.Context, req *ps.RemoveOrganizationTeamMemberRequest) error {
			c.Assert(req.Team, qt.Equals, "platform")
			c.Assert(req.ID, qt.Equals, "membership-1")
			c.Assert(req.DeletePasswords, qt.IsTrue)
			return nil
		},
	}
	cmd := TeamMemberRemoveCmd(teamHelper(svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"platform", "ada@example.com", "--force", "--delete-passwords"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.RemoveTeamMemberFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"result": "team member removed"`)
}

func TestOrg_TeamCmdIncludesSubcommands(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	cmd := TeamCmd(teamHelper(&mock.OrganizationsService{}, printer.Human, &buf))
	for _, name := range []string{"list", "show", "create", "update", "delete", "member"} {
		found, _, err := cmd.Find([]string{name})
		c.Assert(err, qt.IsNil)
		c.Assert(found.Name(), qt.Equals, name)
	}
}
