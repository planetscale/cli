package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestNekiAdminsService(t *testing.T) {
	c := qt.New(t)
	const basePath = "/v1/organizations/acme/databases/app/branches/main"
	const adminPath = basePath + "/admin"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAdmin := func() {
			_, _ = w.Write([]byte(`{"type":"NekiAdmin","id":"admin-1","admin_size":"NKA_0","sku":{"name":"NKA_0","display_name":"NKA-0"},"state":"ready","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z"}`))
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == adminPath:
			writeAdmin()

		case r.Method == http.MethodPatch && r.URL.Path == adminPath:
			var body map[string]interface{}
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body, qt.DeepEquals, map[string]interface{}{
				"admin_size": "NKA_1",
				"parameters": map[string]interface{}{"admin": map[string]interface{}{"recovery-poll-interval": "20s"}},
			})
			_, _ = w.Write([]byte(`{"type":"NekiAdmin","id":"admin-1","admin_size":"NKA_1","sku":{"name":"NKA_1","display_name":"NKA-1"},"state":"updating","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-05T12:00:00Z"}`))

		case r.Method == http.MethodGet && r.URL.Path == basePath+"/admin-size-skus":
			_, _ = w.Write([]byte(`[{"name":"NKA_0","display_name":"NKA-0","cpu":"0.5","ram":1073741824,"sort_order":1}]`))

		case r.Method == http.MethodGet && r.URL.Path == adminPath+"/parameters":
			_, _ = w.Write([]byte(`[{"name":"recovery-poll-interval","display_name":"Recovery poll interval","namespace":"admin","advanced":false,"category":null,"description":"How often the admin runs recovery analysis and repairs.","parameter_type":"time","default_value":"10s","value":"20s","required":false,"created_at":"2026-08-04T12:00:00Z","updated_at":null,"restart":false,"url":"https://example.test/parameter"}]`))

		case r.Method == http.MethodGet && r.URL.Path == adminPath+"/changes":
			c.Assert(r.URL.Query().Get("period"), qt.Equals, "24h")
			c.Assert(r.URL.Query().Get("completed_at"), qt.Equals, "2026-08-04")
			c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
			c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "25")
			_, _ = w.Write([]byte(`{"data":[{"id":"change-1","state":"pending","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z","admin_size":"NKA_1","admin_size_display_name":"NKA-1","previous_admin_size":"NKA_0","previous_admin_size_display_name":"NKA-0","parameters":{"admin":{"recovery-poll-interval":"20s"}},"previous_parameters":{"admin":{"recovery-poll-interval":"10s"}}}]}`))

		case r.Method == http.MethodGet && r.URL.Path == adminPath+"/changes/change-1":
			_, _ = w.Write([]byte(`{"id":"change-1","state":"pending","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z","admin_size":"NKA_1","admin_size_display_name":"NKA-1","previous_admin_size":"NKA_0","previous_admin_size_display_name":"NKA-0","parameters":{"admin":{"recovery-poll-interval":"20s"}},"previous_parameters":{"admin":{"recovery-poll-interval":"10s"}}}`))

		case r.Method == http.MethodDelete && r.URL.Path == adminPath+"/changes/change-1":
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodPost && r.URL.Path == adminPath:
			http.Error(w, "admin cannot be created", http.StatusMethodNotAllowed)

		case r.Method == http.MethodDelete && r.URL.Path == adminPath:
			http.Error(w, "admin cannot be deleted", http.StatusMethodNotAllowed)

		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	ctx := context.Background()

	admin, err := client.NekiAdmins.Get(ctx, &GetNekiAdminRequest{Organization: "acme", Database: "app", Branch: "main"})
	c.Assert(err, qt.IsNil)
	c.Assert(admin.ID, qt.Equals, "admin-1")
	c.Assert(admin.AdminSize, qt.Equals, "NKA_0")
	c.Assert(admin.SKU.DisplayName, qt.Equals, "NKA-0")
	c.Assert(admin.State, qt.Equals, "ready")

	size := "NKA_1"
	admin, err = client.NekiAdmins.Update(ctx, &UpdateNekiAdminRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		AdminSize:    &size,
		Parameters:   map[string]map[string]string{"admin": {"recovery-poll-interval": "20s"}},
	})
	c.Assert(err, qt.IsNil)
	c.Assert(admin.AdminSize, qt.Equals, "NKA_1")
	c.Assert(admin.SKU.DisplayName, qt.Equals, "NKA-1")
	c.Assert(admin.State, qt.Equals, "updating")

	skus, err := client.NekiAdmins.ListSizeSKUs(ctx, &ListNekiAdminSizeSKUsRequest{Organization: "acme", Database: "app", Branch: "main"})
	c.Assert(err, qt.IsNil)
	c.Assert(skus, qt.HasLen, 1)
	c.Assert(skus[0].Name, qt.Equals, "NKA_0")
	c.Assert(skus[0].DisplayName, qt.Equals, "NKA-0")

	parameters, err := client.NekiAdmins.ListParameters(ctx, &ListNekiAdminParametersRequest{Organization: "acme", Database: "app", Branch: "main"})
	c.Assert(err, qt.IsNil)
	c.Assert(parameters, qt.HasLen, 1)
	c.Assert(parameters[0].Name, qt.Equals, "recovery-poll-interval")
	c.Assert(parameters[0].Namespace, qt.Equals, "admin")
	c.Assert(parameters[0].Value, qt.Equals, "20s")

	changes, err := client.NekiAdmins.ListChanges(ctx, &ListNekiAdminChangesRequest{Organization: "acme", Database: "app", Branch: "main", Period: "24h", CompletedAt: "2026-08-04", Page: 2, PerPage: 25})
	c.Assert(err, qt.IsNil)
	c.Assert(changes, qt.HasLen, 1)
	c.Assert(changes[0].ID, qt.Equals, "change-1")
	c.Assert(changes[0].AdminSize, qt.Equals, "NKA_1")
	c.Assert(changes[0].AdminSizeDisplayName, qt.Equals, "NKA-1")
	c.Assert(changes[0].PreviousAdminSize, qt.Equals, "NKA_0")

	change, err := client.NekiAdmins.GetChange(ctx, &GetNekiAdminChangeRequest{Organization: "acme", Database: "app", Branch: "main", Change: "change-1"})
	c.Assert(err, qt.IsNil)
	c.Assert(change.State, qt.Equals, "pending")

	err = client.NekiAdmins.CancelChange(ctx, &CancelNekiAdminChangeRequest{Organization: "acme", Database: "app", Branch: "main", Change: "change-1"})
	c.Assert(err, qt.IsNil)
}
