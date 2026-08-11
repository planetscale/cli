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

func TestPostgresBouncers_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/bouncers")
		w.WriteHeader(200)
		out := `{
			"data":[{
				"id":"bouncer-1",
				"name":"read-pool",
				"sku":{"name":"PGB_10","display_name":"PS-10","cpu":"0.25","ram":268435456,"sort_order":1},
				"target":"replica",
				"replicas_per_cell":1,
				"created_at":"2021-01-14T10:19:23.000Z",
				"updated_at":"2021-01-14T10:19:23.000Z",
				"deleted_at":null,
				"actor":{"id":"user-1","display_name":"Alice","avatar_url":"https://example.com/a.png"},
				"branch":{"id":"branch-1","name":"main","created_at":"2021-01-01T00:00:00.000Z","updated_at":"2021-01-01T00:00:00.000Z","deleted_at":null},
				"parameters":[]
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	bouncers, err := client.PostgresBouncers.List(context.Background(), &ListPostgresBouncersRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(bouncers, qt.HasLen, 1)
	c.Assert(bouncers[0].Name, qt.Equals, "read-pool")
	c.Assert(bouncers[0].Target, qt.Equals, "replica")
	c.Assert(bouncers[0].SKU.Name, qt.Equals, "PGB_10")
	c.Assert(bouncers[0].Branch.Name, qt.Equals, "main")
}

func TestPostgresBouncers_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/bouncers/read-pool")
		w.WriteHeader(200)
		out := `{
			"id":"bouncer-1",
			"name":"read-pool",
			"sku":{"name":"PGB_10","display_name":"PS-10","cpu":"0.25","ram":268435456,"sort_order":1},
			"target":"replica",
			"replicas_per_cell":2,
			"created_at":"2021-01-14T10:19:23.000Z",
			"updated_at":"2021-01-14T10:19:23.000Z",
			"deleted_at":null,
			"actor":{"id":"user-1","display_name":"Alice","avatar_url":"https://example.com/a.png"},
			"branch":{"id":"branch-1","name":"main","created_at":"2021-01-01T00:00:00.000Z","updated_at":"2021-01-01T00:00:00.000Z","deleted_at":null},
			"parameters":[{"id":"p1","namespace":"pgbouncer","name":"default_pool_size","display_name":"Default pool size","category":"","description":"","immutable":false,"parameter_type":"integer","default_value":"20","value":"50","required":false,"restart":false,"created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	bouncer, err := client.PostgresBouncers.Get(context.Background(), &GetPostgresBouncerRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
		Bouncer:      "read-pool",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(bouncer.ReplicasPerCell, qt.Equals, 2)
	c.Assert(bouncer.Parameters, qt.HasLen, 1)
	c.Assert(bouncer.Parameters[0].Value, qt.Equals, "50")
}

func TestPostgresBouncers_Create(t *testing.T) {
	c := qt.New(t)

	replicas := 2
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/bouncers")

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["name"], qt.Equals, "read-pool")
		c.Assert(body["target"], qt.Equals, "replica")
		c.Assert(body["bouncer_size"], qt.Equals, "PGB_10")
		c.Assert(body["replicas_per_cell"], qt.Equals, float64(2))

		w.WriteHeader(200)
		out := `{
			"id":"bouncer-2",
			"name":"read-pool",
			"sku":{"name":"PGB_10","display_name":"PS-10","cpu":"0.25","ram":268435456,"sort_order":1},
			"target":"replica",
			"replicas_per_cell":2,
			"created_at":"2021-01-14T10:19:23.000Z",
			"updated_at":"2021-01-14T10:19:23.000Z",
			"deleted_at":null,
			"actor":{"id":"user-1","display_name":"Alice","avatar_url":"https://example.com/a.png"},
			"branch":{"id":"branch-1","name":"main","created_at":"2021-01-01T00:00:00.000Z","updated_at":"2021-01-01T00:00:00.000Z","deleted_at":null},
			"parameters":[]
		}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	bouncer, err := client.PostgresBouncers.Create(context.Background(), &CreatePostgresBouncerRequest{
		Organization:    testOrg,
		Database:        "my-db",
		Branch:          "main",
		Name:            "read-pool",
		Target:          "replica",
		BouncerSize:     "PGB_10",
		ReplicasPerCell: &replicas,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(bouncer.ID, qt.Equals, "bouncer-2")
	c.Assert(bouncer.CreatedAt.Equal(time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)), qt.IsTrue)
}

func TestPostgresBouncers_Delete(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/bouncers/read-pool")
		w.WriteHeader(204)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	err = client.PostgresBouncers.Delete(context.Background(), &DeletePostgresBouncerRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
		Bouncer:      "read-pool",
	})
	c.Assert(err, qt.IsNil)
}

func TestPostgresBouncers_ListResizes(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/bouncers/read-pool/resizes")
		w.WriteHeader(200)
		out := `{
			"data":[{
				"id":"resize-1",
				"state":"pending",
				"replicas_per_cell":2,
				"target":"replica",
				"parameters":{},
				"previous_replicas_per_cell":1,
				"previous_target":"replica",
				"previous_parameters":{},
				"started_at":null,
				"completed_at":null,
				"created_at":"2021-01-14T10:19:23.000Z",
				"updated_at":"2021-01-14T10:19:23.000Z",
				"actor":{"id":"user-1","display_name":"Alice","avatar_url":"https://example.com/a.png"},
				"bouncer":{"id":"bouncer-1","name":"read-pool","created_at":"2021-01-01T00:00:00.000Z","updated_at":"2021-01-01T00:00:00.000Z","deleted_at":null},
				"sku":{"name":"PGB_10","display_name":"PS-10","cpu":"0.25","ram":268435456,"sort_order":1},
				"previous_sku":{"name":"PGB_5","display_name":"PS-5","cpu":"0.125","ram":134217728,"sort_order":0}
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	resizes, err := client.PostgresBouncers.ListResizes(context.Background(), &ListPostgresBouncerResizesRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
		Bouncer:      "read-pool",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(resizes, qt.HasLen, 1)
	c.Assert(resizes[0].ID, qt.Equals, "resize-1")
	c.Assert(resizes[0].State, qt.Equals, PostgresBouncerResizeStatePending)
	c.Assert(resizes[0].SKU.Name, qt.Equals, "PGB_10")
	c.Assert(resizes[0].Finished(), qt.IsFalse)
}

func TestPostgresBouncers_Resize(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/bouncers/read-pool/resizes")
		var body map[string]any
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body["bouncer_size"], qt.Equals, "PGB_10")
		c.Assert(body["target"], qt.Equals, "replica")
		w.WriteHeader(200)
		out := `{
			"id":"resize-1",
			"state":"pending",
			"replicas_per_cell":1,
			"target":"replica",
			"parameters":{},
			"previous_replicas_per_cell":1,
			"previous_target":"replica",
			"previous_parameters":{},
			"started_at":null,
			"completed_at":null,
			"created_at":"2021-01-14T10:19:23.000Z",
			"updated_at":"2021-01-14T10:19:23.000Z",
			"actor":{"id":"user-1","display_name":"Alice","avatar_url":"https://example.com/a.png"},
			"bouncer":{"id":"bouncer-1","name":"read-pool","created_at":"2021-01-01T00:00:00.000Z","updated_at":"2021-01-01T00:00:00.000Z","deleted_at":null},
			"sku":{"name":"PGB_10","display_name":"PS-10","cpu":"0.25","ram":268435456,"sort_order":1},
			"previous_sku":{"name":"PGB_5","display_name":"PS-5","cpu":"0.125","ram":134217728,"sort_order":0}
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	resize, err := client.PostgresBouncers.Resize(context.Background(), &ResizePostgresBouncerRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
		Bouncer:      "read-pool",
		BouncerSize:  "PGB_10",
		Target:       "replica",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(resize.ID, qt.Equals, "resize-1")
	c.Assert(resize.State, qt.Equals, PostgresBouncerResizeStatePending)
}

func TestPostgresBouncers_CancelResizes(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/bouncers/read-pool/resizes")
		w.WriteHeader(204)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	err = client.PostgresBouncers.CancelResizes(context.Background(), &CancelPostgresBouncerResizesRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
		Bouncer:      "read-pool",
	})
	c.Assert(err, qt.IsNil)
}

func TestPostgresBouncerResizeRequest_Finished(t *testing.T) {
	c := qt.New(t)
	c.Assert((&PostgresBouncerResizeRequest{State: PostgresBouncerResizeStatePending}).Finished(), qt.IsFalse)
	c.Assert((&PostgresBouncerResizeRequest{State: PostgresBouncerResizeStateResizing}).Finished(), qt.IsFalse)
	c.Assert((&PostgresBouncerResizeRequest{State: PostgresBouncerResizeStateCompleted}).Finished(), qt.IsTrue)
	c.Assert((&PostgresBouncerResizeRequest{State: PostgresBouncerResizeStateCanceled}).Finished(), qt.IsTrue)
}

func TestPostgresBouncers_Resize_NoChange(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		w.WriteHeader(204)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	resize, err := client.PostgresBouncers.Resize(context.Background(), &ResizePostgresBouncerRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
		Bouncer:      "read-pool",
		BouncerSize:  "PGB_10",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(resize, qt.IsNil)
}
