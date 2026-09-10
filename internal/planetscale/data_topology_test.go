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

func TestDatabaseBranches_DataTopology(t *testing.T) {
	c := qt.New(t)

	wantPath := "/v1/organizations/my-org/databases/my-db/branches/main/data-topology"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, wantPath)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"id":"topology-id","type":"NekiDataTopology","data_topology":{"authoritative_shard_group":"default"},"synced_at":"2026-08-03T12:00:00Z"}`))
		c.Assert(err, qt.IsNil)
	}))
	t.Cleanup(ts.Close)

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	dataTopology, err := client.DatabaseBranches.DataTopology(context.Background(), &BranchDataTopologyRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "main",
	})
	c.Assert(err, qt.IsNil)

	syncedAt := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	c.Assert(dataTopology, qt.DeepEquals, &DataTopology{
		ID:           "topology-id",
		Type:         "NekiDataTopology",
		DataTopology: json.RawMessage(`{"authoritative_shard_group":"default"}`),
		SyncedAt:     &syncedAt,
	})
}

func TestDatabaseBranches_UpdateDataTopology(t *testing.T) {
	c := qt.New(t)

	topology := json.RawMessage(`{"authoritative_shard_group":"default","shard_groups":[]}`)
	wantPath := "/v1/organizations/my-org/databases/my-db/branches/main/data-topology"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPut)
		c.Assert(r.URL.Path, qt.Equals, wantPath)

		var body struct {
			DataTopology json.RawMessage `json:"data_topology"`
		}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(string(body.DataTopology), qt.JSONEquals, map[string]any{
			"authoritative_shard_group": "default",
			"shard_groups":              []any{},
		})

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"id":"topology-id","type":"NekiDataTopology","data_topology":{"authoritative_shard_group":"default","shard_groups":[]},"synced_at":null}`))
		c.Assert(err, qt.IsNil)
	}))
	t.Cleanup(ts.Close)

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	dataTopology, err := client.DatabaseBranches.UpdateDataTopology(context.Background(), &UpdateBranchDataTopologyRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "main",
		DataTopology: topology,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(dataTopology.DataTopology), qt.JSONEquals, map[string]any{
		"authoritative_shard_group": "default",
		"shard_groups":              []any{},
	})
}
