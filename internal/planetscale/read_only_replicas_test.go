package planetscale

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func readOnlyReplicaJSON() string {
	return `{
		"id": "replica-123",
		"name": "analytics",
		"state": "ready",
		"replicas": 2,
		"cluster_name": "ps-abc",
		"cluster_display_name": "PS-10",
		"access_host_url": "access.example.com",
		"private_access_host_url": "private.example.com",
		"private_connection_service_name": "svc",
		"created_at": "2025-01-15T10:30:00Z",
		"updated_at": "2025-01-15T10:31:00Z",
		"ready_at": "2025-01-15T10:31:00Z",
		"ready": true,
		"region": {"slug": "us-east"},
		"parameters": []
	}`
}

func TestReadOnlyReplicas_List(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/read-only-replicas")
		_, err := io.WriteString(w, "["+readOnlyReplicaJSON()+"]")
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	replicas, err := client.ReadOnlyReplicas.List(context.Background(), &ListReadOnlyReplicasRequest{
		Organization: testOrg, Database: testDatabase, Branch: "main",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(replicas, qt.HasLen, 1)
	c.Assert(replicas[0].Name, qt.Equals, "analytics")
	c.Assert(replicas[0].Region.Slug, qt.Equals, "us-east")
}

func TestReadOnlyReplicas_Create(t *testing.T) {
	c := qt.New(t)
	replicas := 3
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/read-only-replicas")
		body, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(string(body), qt.Equals, "{\"name\":\"analytics\",\"region\":\"us-east\",\"replicas\":3,\"cluster_size\":\"PS-10\"}\n")
		_, err = io.WriteString(w, readOnlyReplicaJSON())
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	replica, err := client.ReadOnlyReplicas.Create(context.Background(), &CreateReadOnlyReplicaRequest{
		Organization: testOrg, Database: testDatabase, Branch: "main",
		Name: "analytics", Region: "us-east", Replicas: &replicas, ClusterSize: "PS-10",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(replica.ID, qt.Equals, "replica-123")
}

func TestReadOnlyReplicas_Get(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/read-only-replicas/analytics")
		_, err := io.WriteString(w, readOnlyReplicaJSON())
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	replica, err := client.ReadOnlyReplicas.Get(context.Background(), &GetReadOnlyReplicaRequest{
		Organization: testOrg, Database: testDatabase, Branch: "main", Name: "analytics",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(replica.Name, qt.Equals, "analytics")
}

func TestReadOnlyReplicas_Delete(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/read-only-replicas/analytics")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	err = client.ReadOnlyReplicas.Delete(context.Background(), &DeleteReadOnlyReplicaRequest{
		Organization: testOrg, Database: testDatabase, Branch: "main", Name: "analytics",
	})
	c.Assert(err, qt.IsNil)
}

func TestReadOnlyReplicas_ListChanges(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/main/read-only-replica-changes?page=2&period=1d")
		_, err := io.WriteString(w, `{"data":[{"id":"change-123","state":"completed","replica":{"id":"replica-123","name":"analytics","created_at":"2025-01-15T10:30:00Z","updated_at":"2025-01-15T10:31:00Z"}}]}`)
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	changes, err := client.ReadOnlyReplicas.ListChanges(context.Background(), &ListReadOnlyReplicaChangesRequest{
		Organization: testOrg, Database: testDatabase, Branch: "main", Period: "1d",
	}, WithPage(2))
	c.Assert(err, qt.IsNil)
	c.Assert(changes, qt.HasLen, 1)
	c.Assert(changes[0].ID, qt.Equals, "change-123")
	c.Assert(changes[0].Replica.Name, qt.Equals, "analytics")
}
