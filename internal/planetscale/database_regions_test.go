package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestDatabases_ListRegions(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/regions")
		c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "50")

		_, err := w.Write([]byte(`{
			"data": [{
				"id": "reg123",
				"slug": "us-east",
				"provider": "AWS",
				"display_name": "US East",
				"location": "Northern Virginia",
				"enabled": true,
				"current_default": true,
				"public_ip_addresses": ["192.0.2.1"],
				"mysql_supported": true,
				"postgresql_supported": true
			}]
		}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	regions, err := client.Databases.ListRegions(context.Background(), &ListDatabaseRegionsRequest{
		Organization: "my-org",
		Database:     "my-db",
	}, WithPage(2), WithPerPage(50))
	c.Assert(err, qt.IsNil)
	c.Assert(regions, qt.DeepEquals, []*Region{{
		ID:                  "reg123",
		Slug:                "us-east",
		Provider:            "AWS",
		Name:                "US East",
		Location:            "Northern Virginia",
		Enabled:             true,
		IsDefault:           true,
		PublicIPAddresses:   []string{"192.0.2.1"},
		MySQLSupported:      true,
		PostgreSQLSupported: true,
	}})
}
