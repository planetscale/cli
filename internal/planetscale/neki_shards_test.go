package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestNekiShardsService(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/shards":
			c.Assert(r.URL.Query().Get("q"), qt.Equals, "report")
			c.Assert(r.URL.Query().Get("exclude_configuration_profile"), qt.Equals, "archive")
			c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
			c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "25")
			_, _ = w.Write([]byte(`{"data":[{"id":"shard-1","name":"0","display_name":"reports","configuration_profile":"default","ready":true,"authoritative":true,"created_at":"2026-08-04T12:00:00Z"}]}`))

		case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/shards/shard-1":
			_, _ = w.Write([]byte(`{"id":"shard-1","name":"0","display_name":"reports","configuration_profile":"default","ready":true,"authoritative":true,"created_at":"2026-08-04T12:00:00Z"}`))

		case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/configuration-profiles/analytics/shards":
			_, _ = w.Write([]byte(`{"data":[{"id":"shard-2","name":"1","display_name":null,"configuration_profile":"analytics","ready":false,"authoritative":false,"created_at":"2026-08-04T12:00:00Z"}]}`))

		case r.Method == http.MethodPost && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/configuration-profiles/analytics/shards/bulk":
			var body map[string]interface{}
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body, qt.DeepEquals, map[string]interface{}{"count": float64(2)})
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"created":2,"shard_ids":["shard-2","shard-3"]}`))

		case r.Method == http.MethodPatch && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/configuration-profiles/analytics/shards":
			var body map[string]interface{}
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body, qt.DeepEquals, map[string]interface{}{"shard_ids": []interface{}{"shard-1", "shard-2"}})
			_, _ = w.Write([]byte(`[{"id":"shard-1","status":"assigned"},{"id":"shard-2","status":"failed","error":{"code":"incompatible_architecture"}}]`))

		case r.Method == http.MethodPatch && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/shards/shard-1":
			var body map[string]interface{}
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body, qt.DeepEquals, map[string]interface{}{"display_name": nil})
			_, _ = w.Write([]byte(`{"id":"shard-1","name":"0","display_name":null,"configuration_profile":"default","ready":true,"created_at":"2026-08-04T12:00:00Z"}`))

		case r.Method == http.MethodDelete && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/shards/shard-1":
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	ctx := context.Background()

	shards, err := client.NekiShards.List(ctx, &ListNekiShardsRequest{
		Organization:                "acme",
		Database:                    "app",
		Branch:                      "main",
		ExcludeConfigurationProfile: "archive",
		Query:                       "report",
		Page:                        2,
		PerPage:                     25,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(shards, qt.HasLen, 1)
	c.Assert(shards[0].ID, qt.Equals, "shard-1")
	c.Assert(shards[0].Ready, qt.IsTrue)
	c.Assert(shards[0].Authoritative, qt.IsTrue)

	shard, err := client.NekiShards.Get(ctx, &GetNekiShardRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Shard:        "shard-1",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(shard.ID, qt.Equals, "shard-1")
	c.Assert(shard.Authoritative, qt.IsTrue)

	shards, err = client.NekiShards.List(ctx, &ListNekiShardsRequest{
		Organization:         "acme",
		Database:             "app",
		Branch:               "main",
		ConfigurationProfile: "analytics",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(shards[0].ConfigurationProfile, qt.Equals, "analytics")
	c.Assert(shards[0].Ready, qt.IsFalse)
	c.Assert(shards[0].Authoritative, qt.IsFalse)

	created, err := client.NekiShards.Create(ctx, &CreateNekiShardsRequest{
		Organization:         "acme",
		Database:             "app",
		Branch:               "main",
		ConfigurationProfile: "analytics",
		Count:                2,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(created.ShardIDs, qt.DeepEquals, []string{"shard-2", "shard-3"})

	assignments, err := client.NekiShards.Assign(ctx, &AssignNekiShardsRequest{
		Organization:         "acme",
		Database:             "app",
		Branch:               "main",
		ConfigurationProfile: "analytics",
		ShardIDs:             []string{"shard-1", "shard-2"},
	})
	c.Assert(err, qt.IsNil)
	c.Assert(assignments, qt.HasLen, 2)
	c.Assert(assignments[1].Error["code"], qt.Equals, "incompatible_architecture")

	updated, err := client.NekiShards.Update(ctx, &UpdateNekiShardRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Shard:        "shard-1",
		DisplayName:  nil,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(updated.DisplayName, qt.IsNil)
	c.Assert(updated.Ready, qt.IsTrue)

	err = client.NekiShards.Delete(ctx, &DeleteNekiShardRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Shard:        "shard-1",
	})
	c.Assert(err, qt.IsNil)
}
