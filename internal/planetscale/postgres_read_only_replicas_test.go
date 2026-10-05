package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

const testDedicatedReadReplicaJSON = `{
	"id":"replica-1",
	"name":"analytics",
	"state":"ready",
	"replicas":2,
	"cluster_name":"PS_10_GCP_X86",
	"cluster_display_name":"PS-10",
	"access_host_url":"replica.example.com",
	"private_access_host_url":"",
	"private_connection_service_name":null,
	"created_at":"2026-08-28T10:19:23.000Z",
	"updated_at":"2026-08-28T10:20:23.000Z",
	"ready_at":"2026-08-28T10:20:23.000Z",
	"ready":true,
	"actor":{"id":"user-1","display_name":"Alice"},
	"region":{"id":"region-1","slug":"us-east","provider":"GCP","display_name":"US East"},
	"parameters":[]
}`

func TestPostgresDedicatedReadReplicas_List(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/read-only-replicas")
		_, err := w.Write([]byte("[" + testDedicatedReadReplicaJSON + "]"))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	replicas, err := client.PostgresDedicatedReadReplicas.List(context.Background(), &ListPostgresDedicatedReadReplicasRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(replicas, qt.HasLen, 1)
	c.Assert(replicas[0].ID, qt.Equals, "replica-1")
	c.Assert(replicas[0].Name, qt.Equals, "analytics")
	c.Assert(replicas[0].Region.Slug, qt.Equals, "us-east")
}

func TestPostgresDedicatedReadReplicas_Get(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/read-only-replicas/analytics")
		_, err := w.Write([]byte(testDedicatedReadReplicaJSON))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	replica, err := client.PostgresDedicatedReadReplicas.Get(context.Background(), &GetPostgresDedicatedReadReplicaRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
		Replica:      "analytics",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(replica.ID, qt.Equals, "replica-1")
	c.Assert(replica.Name, qt.Equals, "analytics")
}

func TestPostgresDedicatedReadReplicas_Create(t *testing.T) {
	c := qt.New(t)
	replicaCount := 2
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/read-only-replicas")

		var body map[string]any
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body, qt.DeepEquals, map[string]any{
			"name":         "analytics",
			"region":       "us-east",
			"replicas":     float64(2),
			"cluster_size": "PS_10_GCP_X86",
		})
		_, err := w.Write([]byte(testDedicatedReadReplicaJSON))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	replica, err := client.PostgresDedicatedReadReplicas.Create(context.Background(), &CreatePostgresDedicatedReadReplicaRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
		Name:         "analytics",
		Region:       "us-east",
		Replicas:     &replicaCount,
		ClusterSize:  "PS_10_GCP_X86",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(replica.ID, qt.Equals, "replica-1")
}

func TestPostgresDedicatedReadReplicas_Update(t *testing.T) {
	c := qt.New(t)
	replicaCount := 3
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/read-only-replicas/analytics")

		var body map[string]any
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body["replicas"], qt.Equals, float64(3))
		c.Assert(body["cluster_size"], qt.Equals, "PS_20_GCP_X86")
		c.Assert(body["parameters"], qt.DeepEquals, map[string]any{
			"pgconf": map[string]any{"max_connections": "300"},
		})
		_, err := w.Write([]byte(testDedicatedReadReplicaJSON))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	replica, err := client.PostgresDedicatedReadReplicas.Update(context.Background(), &UpdatePostgresDedicatedReadReplicaRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
		Replica:      "analytics",
		Replicas:     &replicaCount,
		ClusterSize:  "PS_20_GCP_X86",
		Parameters: map[string]map[string]string{
			"pgconf": {"max_connections": "300"},
		},
	})
	c.Assert(err, qt.IsNil)
	c.Assert(replica.ID, qt.Equals, "replica-1")
}

func TestPostgresDedicatedReadReplicas_Delete(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/read-only-replicas/analytics")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	err = client.PostgresDedicatedReadReplicas.Delete(context.Background(), &DeletePostgresDedicatedReadReplicaRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
		Replica:      "analytics",
	})
	c.Assert(err, qt.IsNil)
}

func TestPostgresReadOnlyReplicasBackwardCompatibility(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/read-only-replicas")
		_, err := w.Write([]byte("[" + testDedicatedReadReplicaJSON + "]"))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	c.Assert(client.PostgresReadOnlyReplicas, qt.Equals, client.PostgresDedicatedReadReplicas)

	replicas, err := client.PostgresReadOnlyReplicas.List(context.Background(), &ListPostgresReadOnlyReplicasRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(replicas, qt.HasLen, 1)

	replica := replicas[0]
	c.Assert(replica.Name, qt.Equals, "analytics")
}
