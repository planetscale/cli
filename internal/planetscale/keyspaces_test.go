package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestKeyspaces_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"type":"list","current_page":1,"next_page":null,"next_page_url":null,"prev_page":null,"prev_page_url":null,"data":[{"id":"thisisanid","type":"Keyspace","name":"planetscale","shards":2,"sharded":true,"created_at":"2022-01-14T15:39:28.394Z","updated_at":"2021-12-20T21:11:07.697Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	keyspaces, err := client.Keyspaces.List(ctx, &ListKeyspacesRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
	})

	wantID := "thisisanid"

	c.Assert(err, qt.IsNil)
	c.Assert(len(keyspaces), qt.Equals, 1)
	c.Assert(keyspaces[0].ID, qt.Equals, wantID)
	c.Assert(keyspaces[0].Sharded, qt.Equals, true)
	c.Assert(keyspaces[0].Shards, qt.Equals, 2)
}

func TestKeyspaces_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"type":"Keyspace","id":"thisisanid","name":"planetscale","shards":2,"sharded":true,"created_at":"2022-01-14T15:39:28.394Z","updated_at":"2021-12-20T21:11:07.697Z"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	keyspace, err := client.Keyspaces.Get(ctx, &GetKeyspaceRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
		Keyspace:     "qux",
	})

	wantID := "thisisanid"

	c.Assert(err, qt.IsNil)
	c.Assert(keyspace.ID, qt.Equals, wantID)
	c.Assert(keyspace.Sharded, qt.Equals, true)
	c.Assert(keyspace.Shards, qt.Equals, 2)
}

func TestKeyspaces_GetFull(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Query().Get("full"), qt.Equals, "true")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"id": "thisisanid",
			"name": "main",
			"read_only_regions": [{
				"region": "us-west",
				"cluster_name": "PS_20",
				"cluster_display_name": "PS-20",
				"replicas": 2
			}]
		}`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	keyspace, err := client.Keyspaces.Get(context.Background(), &GetKeyspaceRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
		Keyspace:     "main",
		Full:         true,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(keyspace.ReadOnlyRegions, qt.DeepEquals, []*ReadOnlyRegionKeyspace{{
		Region:             "us-west",
		ClusterName:        "PS_20",
		ClusterDisplayName: "PS-20",
		Replicas:           2,
	}})
}

func TestKeyspaces_Create(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		out := `{"type":"Keyspace","id":"thisisanid","name":"planetscale","shards":2,"sharded":true,"created_at":"2022-01-14T15:39:28.394Z","updated_at":"2021-12-20T21:11:07.697Z"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
		c.Assert(r.Method, qt.Equals, http.MethodPost)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	keyspace, err := client.Keyspaces.Create(ctx, &CreateKeyspaceRequest{
		Organization:  "foo",
		Database:      "bar",
		Branch:        "baz",
		Name:          "qux",
		ClusterSize:   "small",
		ExtraReplicas: 3,
		Shards:        2,
	})

	wantID := "thisisanid"

	c.Assert(err, qt.IsNil)
	c.Assert(keyspace.ID, qt.Equals, wantID)
	c.Assert(keyspace.Sharded, qt.Equals, true)
	c.Assert(keyspace.Shards, qt.Equals, 2)
}

func TestKeyspaces_VSchema(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"raw":"{\"sharded\":true,\"tables\":{}}","html":"<div>\"sharded\":true,\"tables\":{}</div>"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	vSchema, err := client.Keyspaces.VSchema(ctx, &GetKeyspaceVSchemaRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
		Keyspace:     "qux",
	})

	wantRaw := "{\"sharded\":true,\"tables\":{}}"
	wantHTML := "<div>\"sharded\":true,\"tables\":{}</div>"

	c.Assert(err, qt.IsNil)
	c.Assert(vSchema.Raw, qt.Equals, wantRaw)
	c.Assert(vSchema.HTML, qt.Equals, wantHTML)
}

func TestKeyspaces_UpdateVSchema(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"raw":"{\"sharded\":true,\"tables\":{}}","html":"<div>\"sharded\":true,\"tables\":{}</div>"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	vSchema, err := client.Keyspaces.UpdateVSchema(ctx, &UpdateKeyspaceVSchemaRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
		Keyspace:     "qux",
		VSchema:      "{\"sharded\":true,\"tables\":{}}",
	})

	wantRaw := "{\"sharded\":true,\"tables\":{}}"
	wantHTML := "<div>\"sharded\":true,\"tables\":{}</div>"

	c.Assert(err, qt.IsNil)
	c.Assert(vSchema.Raw, qt.Equals, wantRaw)
	c.Assert(vSchema.HTML, qt.Equals, wantHTML)
}

func TestKeyspaces_Resize(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"thisisanid","type":"KeyspaceResizeRequest","state":"pending","started_at":"2024-06-25T18:03:09.459Z","completed_at":"2024-06-25T18:04:06.228Z","created_at":"2024-06-25T18:03:09.439Z","updated_at":"2024-06-25T18:04:06.238Z","actor":{"id":"actorid","type":"User","display_name":"Test User"},"cluster_name":"PS_10","extra_replicas":1,"previous_cluster_name":"PS_10","replicas":3,"previous_replicas":5}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
		c.Assert(r.Method, qt.Equals, http.MethodPut)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	size := "PS_10"
	replicas := uint(3)

	krr, err := client.Keyspaces.Resize(ctx, &ResizeKeyspaceRequest{
		Organization:  "foo",
		Database:      "bar",
		Branch:        "baz",
		Keyspace:      "qux",
		ClusterSize:   &size,
		ExtraReplicas: &replicas,
	})

	wantID := "thisisanid"

	c.Assert(err, qt.IsNil)
	c.Assert(krr.ID, qt.Equals, wantID)
	c.Assert(krr.ExtraReplicas, qt.Equals, uint(1))
	c.Assert(krr.Replicas, qt.Equals, uint(3))
	c.Assert(krr.PreviousReplicas, qt.Equals, uint(5))
	c.Assert(krr.ClusterSize, qt.Equals, "PS_10")
	c.Assert(krr.PreviousClusterSize, qt.Equals, "PS_10")
}

func TestKeyspaces_CancelResize(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	err = client.Keyspaces.CancelResize(ctx, &CancelKeyspaceResizeRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
		Keyspace:     "qux",
	})

	c.Assert(err, qt.IsNil)
}

func TestKeyspaces_ResizeStatus(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"type":"list","current_page":1,"next_page":null,"next_page_url":null,"prev_page":null,"prev_page_url":null,"data":[{"id":"thisisanid","type":"KeyspaceResizeRequest","state":"completed","started_at":"2024-06-25T18:03:09.459Z","completed_at":"2024-06-25T18:04:06.228Z","created_at":"2024-06-25T18:03:09.439Z","updated_at":"2024-06-25T18:04:06.238Z","actor":{"id":"thisisanid","type":"User","display_name":"Test User"},"cluster_name":"PS_10","extra_replicas":0,"previous_cluster_name":"PS_10","replicas":2,"previous_replicas":5}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	krr, err := client.Keyspaces.ResizeStatus(ctx, &KeyspaceResizeStatusRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
		Keyspace:     "qux",
	})

	wantID := "thisisanid"

	c.Assert(err, qt.IsNil)
	c.Assert(krr.ID, qt.Equals, wantID)
	c.Assert(krr.ExtraReplicas, qt.Equals, uint(0))
	c.Assert(krr.Replicas, qt.Equals, uint(2))
	c.Assert(krr.PreviousReplicas, qt.Equals, uint(5))
	c.Assert(krr.ClusterSize, qt.Equals, "PS_10")
	c.Assert(krr.PreviousClusterSize, qt.Equals, "PS_10")
}

func TestKeyspaces_ResizeStatusEmpty(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"type":"list","current_page":1,"next_page":null,"next_page_url":null,"prev_page":null,"prev_page_url":null,"data":[]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	krr, err := client.Keyspaces.ResizeStatus(ctx, &KeyspaceResizeStatusRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
		Keyspace:     "qux",
	})

	wantError := &Error{
		msg:  "Not Found",
		Code: ErrNotFound,
	}

	c.Assert(krr, qt.IsNil)
	c.Assert(err.Error(), qt.Equals, wantError.Error())
}

func TestKeyspaces_RolloutStatus(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"type":"BranchInfrastructureKeyspace","state":"complete","name":"qux","shards":[{"type":"BranchInfrastructureKeyspaceShard","state":"complete","last_rollout_started_at":"2025-01-17T18:27:25.027Z","last_rollout_finished_at":"2025-01-17T18:28:25.027Z","name":"-80"},{"type":"BranchInfrastructureKeyspaceShard","state":"complete","last_rollout_started_at":"2025-01-17T18:28:25.033Z","last_rollout_finished_at":"2025-01-17T18:29:25.033Z","name":"80-"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	krr, err := client.Keyspaces.RolloutStatus(ctx, &KeyspaceRolloutStatusRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
		Keyspace:     "qux",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(krr.Name, qt.Equals, "qux")
	c.Assert(krr.State, qt.Equals, "complete")
	c.Assert(len(krr.Shards), qt.Equals, int(2))
	c.Assert(krr.Shards[0].Name, qt.Equals, "-80")
	c.Assert(krr.Shards[0].State, qt.Equals, "complete")
	c.Assert(krr.Shards[1].Name, qt.Equals, "80-")
	c.Assert(krr.Shards[1].State, qt.Equals, "complete")
}

func TestKeyspaces_UpdateSettings(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"type":"Keyspace","id":"thisisanid","name":"planetscale","shards":2,"sharded":true,"created_at":"2022-01-14T15:39:28.394Z","updated_at":"2021-12-20T21:11:07.697Z","vreplication_flags":{"optimize_inserts":true,"allow_no_blob_binlog_row_image":true,"vplayer_batching":true},"replication_durability_constraints":{"strategy":"maximum"}}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	keyspace, err := client.Keyspaces.UpdateSettings(ctx, &UpdateKeyspaceSettingsRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
		Keyspace:     "qux",
		VReplicationFlags: &VReplicationFlags{
			OptimizeInserts:           true,
			AllowNoBlobBinlogRowImage: true,
			VPlayerBatching:           true,
		},
		ReplicationDurabilityConstraints: &ReplicationDurabilityConstraints{
			Strategy: "maximum",
		},
	})

	c.Assert(err, qt.IsNil)
	c.Assert(keyspace.ID, qt.Equals, "thisisanid")
	c.Assert(keyspace.Sharded, qt.Equals, true)
	c.Assert(keyspace.Shards, qt.Equals, 2)
	c.Assert(keyspace.VReplicationFlags.OptimizeInserts, qt.Equals, true)
	c.Assert(keyspace.VReplicationFlags.AllowNoBlobBinlogRowImage, qt.Equals, true)
	c.Assert(keyspace.VReplicationFlags.VPlayerBatching, qt.Equals, true)
	c.Assert(keyspace.ReplicationDurabilityConstraints.Strategy, qt.Equals, "maximum")
}
