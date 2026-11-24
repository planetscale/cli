package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

const organizationCustomRoleJSON = `{"id":"role-1","name":"Billing admin","slug":"billing-admin","description":"Can view billing","permissions":[{"action":"show_organization"}],"created_at":"2026-08-25T10:00:00Z","updated_at":"2026-08-25T10:00:00Z"}`

func TestOrganizations_ListCustomRoles(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/custom-roles")
		c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "25")
		_, err := w.Write([]byte(`{"data":[` + organizationCustomRoleJSON + `]}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	roles, err := client.Organizations.ListCustomRoles(context.Background(), &ListOrganizationCustomRolesRequest{
		Organization: "my-org",
	}, WithPage(2), WithPerPage(25))
	c.Assert(err, qt.IsNil)
	c.Assert(roles, qt.HasLen, 1)
	c.Assert(roles[0].ID, qt.Equals, "role-1")
	c.Assert(roles[0].Name, qt.Equals, "Billing admin")
	c.Assert(roles[0].Slug, qt.Equals, "billing-admin")
	c.Assert(*roles[0].Description, qt.Equals, "Can view billing")
	c.Assert(roles[0].Permissions, qt.DeepEquals, []OrganizationCustomRolePermission{{Action: "show_organization"}})
	c.Assert(roles[0].CreatedAt, qt.Equals, time.Date(2026, time.August, 25, 10, 0, 0, 0, time.UTC))
}

func TestOrganizations_CreateCustomRole(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/custom-roles")
		var body struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Permissions []string `json:"permissions"`
		}
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body.Name, qt.Equals, "Billing admin")
		c.Assert(body.Description, qt.Equals, "Can view billing")
		c.Assert(body.Permissions, qt.DeepEquals, []string{"show_organization", "show_invoice"})
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(organizationCustomRoleJSON))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	role, err := client.Organizations.CreateCustomRole(context.Background(), &CreateOrganizationCustomRoleRequest{
		Organization: "my-org",
		Name:         "Billing admin",
		Description:  "Can view billing",
		Permissions:  []string{"show_organization", "show_invoice"},
	})
	c.Assert(err, qt.IsNil)
	c.Assert(role.Slug, qt.Equals, "billing-admin")
}

func TestOrganizations_UpdateCustomRole(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/custom-roles/billing-admin")
		var body map[string]any
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body["name"], qt.Equals, "Billing operator")
		c.Assert(body["permissions"], qt.DeepEquals, []any{"show_organization", "show_invoice"})
		_, hasDescription := body["description"]
		c.Assert(hasDescription, qt.IsFalse)
		_, err := w.Write([]byte(organizationCustomRoleJSON))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	name := "Billing operator"
	permissions := []string{"show_organization", "show_invoice"}
	role, err := client.Organizations.UpdateCustomRole(context.Background(), &UpdateOrganizationCustomRoleRequest{
		Organization: "my-org",
		Slug:         "billing-admin",
		Name:         &name,
		Permissions:  &permissions,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(role.Slug, qt.Equals, "billing-admin")
}

func TestOrganizations_DeleteCustomRole(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/custom-roles/billing-admin")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	err = client.Organizations.DeleteCustomRole(context.Background(), &DeleteOrganizationCustomRoleRequest{
		Organization: "my-org",
		Slug:         "billing-admin",
	})
	c.Assert(err, qt.IsNil)
}
