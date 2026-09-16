package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestNekiChangesService(t *testing.T) {
	c := qt.New(t)
	const basePath = "/v1/organizations/acme/databases/app/branches/main/neki-changes"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == basePath:
			c.Assert(r.URL.Query()["state[]"], qt.DeepEquals, []string{"pending", "applying"})
			c.Assert(r.URL.Query()["target_types[]"], qt.DeepEquals, []string{"NekiAdmin", "NekiRouter"})
			c.Assert(r.URL.Query().Get("target_id"), qt.Equals, "admin-1")
			c.Assert(r.URL.Query().Get("period"), qt.Equals, "24h")
			c.Assert(r.URL.Query().Get("completed_at"), qt.Equals, "2026-08-04")
			c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
			c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "25")
			_, _ = w.Write([]byte(`{"data":[{"type":"NekiAdminChangeRequest","id":"change-1","state":"pending","target_id":"admin-1","target_type":"NekiAdmin","target_name":"Admin","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z","admin_size":"NKA_1"}]}`))

		case r.Method == http.MethodGet && r.URL.Path == basePath+"/change-1":
			_, _ = w.Write([]byte(`{"type":"NekiAdminChangeRequest","id":"change-1","state":"pending","target_id":"admin-1","target_type":"NekiAdmin","target_name":"Admin","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z","admin_size":"NKA_1"}`))

		case r.Method == http.MethodDelete && r.URL.Path == basePath+"/change-1":
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	ctx := context.Background()

	changes, err := client.NekiChanges.List(ctx, &ListNekiChangesRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		States:       []string{"pending", "applying"},
		TargetTypes:  []string{"NekiAdmin", "NekiRouter"},
		TargetID:     "admin-1",
		Period:       "24h",
		CompletedAt:  "2026-08-04",
		Page:         2,
		PerPage:      25,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(changes, qt.HasLen, 1)
	c.Assert(changes[0].ID, qt.Equals, "change-1")
	c.Assert(changes[0].TargetType, qt.Equals, "NekiAdmin")
	c.Assert(changes[0].TargetName, qt.Equals, "Admin")

	var listed map[string]any
	c.Assert(json.Unmarshal(mustMarshal(c, changes[0]), &listed), qt.IsNil)
	c.Assert(listed["admin_size"], qt.Equals, "NKA_1")

	change, err := client.NekiChanges.Get(ctx, &GetNekiChangeRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Change:       "change-1",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(change.ID, qt.Equals, "change-1")
	c.Assert(change.State, qt.Equals, "pending")

	err = client.NekiChanges.Cancel(ctx, &CancelNekiChangeRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Change:       "change-1",
	})
	c.Assert(err, qt.IsNil)
}

func mustMarshal(c *qt.C, v any) []byte {
	c.Helper()
	data, err := json.Marshal(v)
	c.Assert(err, qt.IsNil)
	return data
}
