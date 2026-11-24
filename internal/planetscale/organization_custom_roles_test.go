package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestOrganizations_ListCustomRoles(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/custom-roles")
		c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "25")
		_, err := w.Write([]byte(`{"data":[{"id":"role-1","name":"Billing admin","slug":"billing-admin","description":"Can view billing","permissions":[{"action":"show_organization"}],"created_at":"2026-08-25T10:00:00Z","updated_at":"2026-08-25T10:00:00Z"}]}`))
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
