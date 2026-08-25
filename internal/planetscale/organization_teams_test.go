package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

const organizationTeamJSON = `{"id":"team-1","display_name":"Platform","creator":{"id":"user-2","display_name":"Grace","avatar_url":""},"members":[],"databases":[],"analyst_databases":[],"name":"Platform","slug":"platform","created_at":"2026-08-25T10:00:00Z","updated_at":"2026-08-25T10:00:00Z","description":"Platform team","managed":false}`

const organizationTeamMemberJSON = `{"id":"membership-1","user":{"id":"user-1","display_name":"Ada Lovelace","name":"Ada","email":"ada@example.com","avatar_url":"","created_at":"2026-08-25T10:00:00Z","updated_at":"2026-08-25T10:00:00Z","two_factor_auth_configured":true,"default_organization":null},"actor":{"id":"user-2","display_name":"Grace","avatar_url":""},"created_at":"2026-08-25T10:00:00Z","updated_at":"2026-08-25T10:00:00Z","passwords":[]}`

func TestOrganizations_ListTeams(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/teams")
		c.Assert(r.URL.Query().Get("q"), qt.Equals, "plat")
		c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "25")
		_, err := w.Write([]byte(`{"data":[` + organizationTeamJSON + `]}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	teams, err := client.Organizations.ListTeams(context.Background(), &ListOrganizationTeamsRequest{
		Organization: "my-org",
		Query:        "plat",
	}, WithPage(2), WithPerPage(25))
	c.Assert(err, qt.IsNil)
	c.Assert(teams, qt.HasLen, 1)
	c.Assert(teams[0].Slug, qt.Equals, "platform")
	c.Assert(*teams[0].Description, qt.Equals, "Platform team")
}

func TestOrganizations_TeamMutations(t *testing.T) {
	c := qt.New(t)
	var requestNumber int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestNumber++
		switch requestNumber {
		case 1:
			c.Assert(r.Method, qt.Equals, http.MethodGet)
			c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/teams/platform")
		case 2:
			c.Assert(r.Method, qt.Equals, http.MethodPost)
			c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/teams")
			var body CreateOrganizationTeamRequest
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body.Name, qt.Equals, "Platform")
		case 3:
			c.Assert(r.Method, qt.Equals, http.MethodPatch)
			c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/teams/platform")
			var body map[string]string
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body, qt.DeepEquals, map[string]string{"description": ""})
		case 4:
			c.Assert(r.Method, qt.Equals, http.MethodDelete)
			c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/teams/platform")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, err := w.Write([]byte(organizationTeamJSON))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	team, err := client.Organizations.GetTeam(context.Background(), &GetOrganizationTeamRequest{Organization: "my-org", Team: "platform"})
	c.Assert(err, qt.IsNil)
	c.Assert(team.ID, qt.Equals, "team-1")
	_, err = client.Organizations.CreateTeam(context.Background(), &CreateOrganizationTeamRequest{Organization: "my-org", Name: "Platform"})
	c.Assert(err, qt.IsNil)
	description := ""
	_, err = client.Organizations.UpdateTeam(context.Background(), &UpdateOrganizationTeamRequest{Organization: "my-org", Team: "platform", Description: &description})
	c.Assert(err, qt.IsNil)
	err = client.Organizations.DeleteTeam(context.Background(), &DeleteOrganizationTeamRequest{Organization: "my-org", Team: "platform"})
	c.Assert(err, qt.IsNil)
}

func TestOrganizations_TeamMembers(t *testing.T) {
	c := qt.New(t)
	var requestNumber int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestNumber++
		switch requestNumber {
		case 1:
			c.Assert(r.Method, qt.Equals, http.MethodGet)
			c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/teams/platform/members")
			c.Assert(r.URL.Query().Get("page"), qt.Equals, "3")
			c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "50")
			_, err := w.Write([]byte(`{"data":[` + organizationTeamMemberJSON + `]}`))
			c.Assert(err, qt.IsNil)
		case 2:
			c.Assert(r.Method, qt.Equals, http.MethodPost)
			c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/teams/platform/members")
			var body map[string]string
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body["user_id"], qt.Equals, "user-1")
			_, err := w.Write([]byte(organizationTeamMemberJSON))
			c.Assert(err, qt.IsNil)
		case 3:
			c.Assert(r.Method, qt.Equals, http.MethodDelete)
			c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/teams/platform/members/membership-1")
			c.Assert(r.URL.Query().Get("delete_passwords"), qt.Equals, "true")
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	members, err := client.Organizations.ListTeamMembers(context.Background(), &ListOrganizationTeamMembersRequest{
		Organization: "my-org",
		Team:         "platform",
	}, WithPage(3), WithPerPage(50))
	c.Assert(err, qt.IsNil)
	c.Assert(members, qt.HasLen, 1)
	c.Assert(members[0].User.Email, qt.Equals, "ada@example.com")
	added, err := client.Organizations.AddTeamMember(context.Background(), &AddOrganizationTeamMemberRequest{
		Organization: "my-org",
		Team:         "platform",
		UserID:       "user-1",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(added.ID, qt.Equals, "membership-1")
	err = client.Organizations.RemoveTeamMember(context.Background(), &RemoveOrganizationTeamMemberRequest{
		Organization:    "my-org",
		Team:            "platform",
		ID:              "membership-1",
		DeletePasswords: true,
	})
	c.Assert(err, qt.IsNil)
}
