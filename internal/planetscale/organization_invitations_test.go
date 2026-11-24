package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

const organizationInvitationJSON = `{"id":"inv-1","email":"ada@example.com","role":"billing-admin","sender":{"id":"user-2","display_name":"Grace","name":"Grace","email":"grace@example.com","avatar_url":"","two_factor_auth_configured":true,"created_at":"2026-08-25T10:00:00Z","updated_at":"2026-08-25T10:00:00Z"},"recipient":null,"expired":false,"expires_at":"2026-09-01T10:00:00Z","created_at":"2026-08-25T10:00:00Z","avatar_url":""}`

func TestOrganizations_ListInvitations(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/invitations")
		c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "25")
		_, err := w.Write([]byte(`{"data":[` + organizationInvitationJSON + `]}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	invitations, err := client.Organizations.ListInvitations(context.Background(), &ListOrganizationInvitationsRequest{
		Organization: "my-org",
	}, WithPage(2), WithPerPage(25))
	c.Assert(err, qt.IsNil)
	c.Assert(invitations, qt.HasLen, 1)
	c.Assert(invitations[0].ID, qt.Equals, "inv-1")
	c.Assert(invitations[0].Role, qt.Equals, "billing-admin")
	c.Assert(invitations[0].Email, qt.Equals, "ada@example.com")
}

func TestOrganizations_CreateInvitation(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/invitations")
		var body CreateOrganizationInvitationRequest
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body.Email, qt.Equals, "ada@example.com")
		c.Assert(body.Role, qt.Equals, "billing-admin")
		_, err := w.Write([]byte(organizationInvitationJSON))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	invitation, err := client.Organizations.CreateInvitation(context.Background(), &CreateOrganizationInvitationRequest{
		Organization: "my-org",
		Email:        "ada@example.com",
		Role:         "billing-admin",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(invitation.Role, qt.Equals, "billing-admin")
}

func TestOrganizations_CancelInvitation(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/invitations/inv-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	err = client.Organizations.CancelInvitation(context.Background(), &CancelOrganizationInvitationRequest{
		Organization: "my-org",
		ID:           "inv-1",
	})
	c.Assert(err, qt.IsNil)
}
