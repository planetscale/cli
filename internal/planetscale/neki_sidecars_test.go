package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestNekiSidecarsService(t *testing.T) {
	c := qt.New(t)
	const basePath = "/v1/organizations/acme/databases/app/branches/main"
	const sidecarsPath = basePath + "/sidecars"
	const sidecarPath = sidecarsPath + "/metal"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeSidecar := func() {
			_, _ = w.Write([]byte(`{"type":"neki_sidecar","id":"sidecar-1","configuration_profile":"metal","state":"ready","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z"}`))
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == sidecarsPath:
			_, _ = w.Write([]byte(`[{"type":"neki_sidecar","id":"sidecar-1","configuration_profile":"metal","state":"ready","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z"}]`))

		case r.Method == http.MethodGet && r.URL.Path == sidecarPath:
			writeSidecar()

		case r.Method == http.MethodPatch && r.URL.Path == sidecarPath:
			var body map[string]interface{}
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body, qt.DeepEquals, map[string]interface{}{
				"parameters": map[string]interface{}{"pgbouncer": map[string]interface{}{"default_pool_size": "20"}},
			})
			_, _ = w.Write([]byte(`{"type":"neki_sidecar","id":"sidecar-1","configuration_profile":"metal","state":"updating","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-05T12:00:00Z"}`))

		case r.Method == http.MethodGet && r.URL.Path == sidecarPath+"/parameters":
			_, _ = w.Write([]byte(`[{"name":"default_pool_size","display_name":"Default pool size","namespace":"pgbouncer","advanced":false,"category":null,"description":"Default pool size","parameter_type":"integer","default_value":"10","value":"20","required":false,"created_at":"2026-08-04T12:00:00Z","updated_at":null,"restart":false,"url":"https://example.test/parameter"}]`))

		case r.Method == http.MethodGet && r.URL.Path == sidecarPath+"/changes":
			c.Assert(r.URL.Query().Get("period"), qt.Equals, "24h")
			c.Assert(r.URL.Query().Get("completed_at"), qt.Equals, "2026-08-04")
			c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
			c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "25")
			_, _ = w.Write([]byte(`{"data":[{"id":"change-1","state":"pending","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z","parameters":{"pgbouncer":{"default_pool_size":"20"}},"previous_parameters":{"pgbouncer":{"default_pool_size":"10"}}}]}`))

		case r.Method == http.MethodGet && r.URL.Path == sidecarPath+"/changes/change-1":
			_, _ = w.Write([]byte(`{"id":"change-1","state":"pending","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z","parameters":{"pgbouncer":{"default_pool_size":"20"}},"previous_parameters":{"pgbouncer":{"default_pool_size":"10"}}}`))

		case r.Method == http.MethodDelete && r.URL.Path == sidecarPath+"/changes/change-1":
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodPost && r.URL.Path == sidecarsPath:
			http.Error(w, "sidecars cannot be created", http.StatusMethodNotAllowed)

		case r.Method == http.MethodDelete && r.URL.Path == sidecarPath:
			http.Error(w, "sidecars cannot be deleted", http.StatusMethodNotAllowed)

		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	ctx := context.Background()

	sidecars, err := client.NekiSidecars.List(ctx, &ListNekiSidecarsRequest{Organization: "acme", Database: "app", Branch: "main"})
	c.Assert(err, qt.IsNil)
	c.Assert(sidecars, qt.HasLen, 1)
	c.Assert(sidecars[0].ID, qt.Equals, "sidecar-1")
	c.Assert(sidecars[0].ConfigurationProfile, qt.Equals, "metal")
	c.Assert(sidecars[0].State, qt.Equals, "ready")

	sidecar, err := client.NekiSidecars.Get(ctx, &GetNekiSidecarRequest{Organization: "acme", Database: "app", Branch: "main", Sidecar: "metal"})
	c.Assert(err, qt.IsNil)
	c.Assert(sidecar.ID, qt.Equals, "sidecar-1")
	c.Assert(sidecar.ConfigurationProfile, qt.Equals, "metal")

	sidecar, err = client.NekiSidecars.Update(ctx, &UpdateNekiSidecarRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Sidecar:      "metal",
		Parameters:   map[string]map[string]string{"pgbouncer": {"default_pool_size": "20"}},
	})
	c.Assert(err, qt.IsNil)
	c.Assert(sidecar.State, qt.Equals, "updating")

	parameters, err := client.NekiSidecars.ListParameters(ctx, &ListNekiSidecarParametersRequest{Organization: "acme", Database: "app", Branch: "main", Sidecar: "metal"})
	c.Assert(err, qt.IsNil)
	c.Assert(parameters, qt.HasLen, 1)
	c.Assert(parameters[0].Name, qt.Equals, "default_pool_size")
	c.Assert(parameters[0].Namespace, qt.Equals, "pgbouncer")
	c.Assert(parameters[0].Value, qt.Equals, "20")

	changes, err := client.NekiSidecars.ListChanges(ctx, &ListNekiSidecarChangesRequest{Organization: "acme", Database: "app", Branch: "main", Sidecar: "metal", Period: "24h", CompletedAt: "2026-08-04", Page: 2, PerPage: 25})
	c.Assert(err, qt.IsNil)
	c.Assert(changes, qt.HasLen, 1)
	c.Assert(changes[0].ID, qt.Equals, "change-1")

	change, err := client.NekiSidecars.GetChange(ctx, &GetNekiSidecarChangeRequest{Organization: "acme", Database: "app", Branch: "main", Sidecar: "metal", Change: "change-1"})
	c.Assert(err, qt.IsNil)
	c.Assert(change.State, qt.Equals, "pending")

	err = client.NekiSidecars.CancelChange(ctx, &CancelNekiSidecarChangeRequest{Organization: "acme", Database: "app", Branch: "main", Sidecar: "metal", Change: "change-1"})
	c.Assert(err, qt.IsNil)
}
