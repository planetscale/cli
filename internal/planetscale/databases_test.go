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

const (
	testOrg      = "my-org"
	testDatabase = "planetscale-go-test-db"
)

func TestDatabases_Create(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"planetscale-go-test-db","type":"database","name":"planetscale-go-test-db","notes":"This is a test DB created from the planetscale-go API library","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z", "region": { "slug": "us-west", "display_name": "US West" },"state":"ready"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"
	notes := "This is a test DB created from the planetscale-go API library"

	db, err := client.Databases.Create(ctx, &CreateDatabaseRequest{
		Organization: org,
		Region:       "us-west",
		Name:         name,
		Notes:        notes,
	})

	want := &Database{
		Name:  name,
		Notes: notes,
		State: DatabaseReady,
		Region: Region{
			Slug: "us-west",
			Name: "US West",
		},
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabases_CreateCloudflareBilling(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["cloudflare_account_id"], qt.Equals, "cf_account_123")
		c.Assert(body["cloudflare_timestamp"], qt.Equals, "1710000000")
		c.Assert(body["cloudflare_signature"], qt.Equals, "abc123sig")

		out := `{"id":"planetscale-go-test-db","type":"database","name":"planetscale-go-test-db","notes":"","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z", "region": { "slug": "us-west", "display_name": "US West" },"state":"ready"}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	db, err := client.Databases.Create(ctx, &CreateDatabaseRequest{
		Organization:        "my-org",
		Region:              "us-west",
		Name:                "planetscale-go-test-db",
		CloudflareAccountID: "cf_account_123",
		CloudflareTimestamp: "1710000000",
		CloudflareSignature: "abc123sig",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(db.Name, qt.Equals, "planetscale-go-test-db")
}

func TestDatabases_CreatePostgres(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["kind"], qt.Equals, "postgresql")

		out := `{"id":"planetscale-go-test-db","type":"database","name":"planetscale-go-test-db","notes":"This is a test DB created from the planetscale-go API library","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z", "region": { "slug": "us-west", "display_name": "US West" },"state":"ready","kind":"postgresql"}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"
	notes := "This is a test DB created from the planetscale-go API library"

	db, err := client.Databases.Create(ctx, &CreateDatabaseRequest{
		Organization: org,
		Region:       "us-west",
		Name:         name,
		Notes:        notes,
		Kind:         DatabaseEnginePostgres,
	})

	want := &Database{
		Name:  name,
		Notes: notes,
		State: DatabaseReady,
		Region: Region{
			Slug: "us-west",
			Name: "US West",
		},
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		Kind:      "postgresql",
	}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabases_CreateWithReplicas(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["replicas"], qt.Equals, float64(3))

		out := `{"id":"planetscale-go-test-db","type":"database","name":"planetscale-go-test-db","notes":"This is a test DB created from the planetscale-go API library","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z", "region": { "slug": "us-west", "display_name": "US West" },"state":"ready"}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"
	notes := "This is a test DB created from the planetscale-go API library"
	replicas := 3

	db, err := client.Databases.Create(ctx, &CreateDatabaseRequest{
		Organization: org,
		Region:       "us-west",
		Name:         name,
		Notes:        notes,
		Replicas:     &replicas,
	})

	want := &Database{
		Name:  name,
		Notes: notes,
		State: DatabaseReady,
		Region: Region{
			Slug: "us-west",
			Name: "US West",
		},
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabases_CreateWithReplicasZero(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)

		// With omitempty and *int type, replicas field SHOULD be present when explicitly set to 0
		replicas, hasReplicas := body["replicas"]
		c.Assert(hasReplicas, qt.IsTrue, qt.Commentf("replicas field should be present when explicitly set to 0"))
		c.Assert(replicas, qt.Equals, float64(0))

		out := `{"id":"planetscale-go-test-db","type":"database","name":"planetscale-go-test-db","notes":"This is a test DB created from the planetscale-go API library","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z", "region": { "slug": "us-west", "display_name": "US West" },"state":"ready"}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"
	notes := "This is a test DB created from the planetscale-go API library"
	replicas := 0

	db, err := client.Databases.Create(ctx, &CreateDatabaseRequest{
		Organization: org,
		Region:       "us-west",
		Name:         name,
		Notes:        notes,
		Replicas:     &replicas,
	})

	want := &Database{
		Name:  name,
		Notes: notes,
		State: DatabaseReady,
		Region: Region{
			Slug: "us-west",
			Name: "US West",
		},
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabases_CreateWithStorage(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.Method, qt.Equals, http.MethodPost)

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)

		storage, ok := body["storage"].(map[string]any)
		c.Assert(ok, qt.IsTrue, qt.Commentf("storage field should be a nested object"))
		c.Assert(storage["minimum_storage_bytes"], qt.Equals, float64(10737418240))
		c.Assert(storage["maximum_storage_bytes"], qt.Equals, float64(107374182400))

		out := `{"id":"planetscale-go-test-db","type":"database","name":"planetscale-go-test-db","notes":"","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z", "region": { "slug": "us-west", "display_name": "US West" },"state":"ready","kind":"postgresql"}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	minStorage := int64(10737418240)
	maxStorage := int64(107374182400)

	db, err := client.Databases.Create(ctx, &CreateDatabaseRequest{
		Organization: testOrg,
		Region:       "us-west",
		Name:         testDatabase,
		Kind:         DatabaseEnginePostgres,
		Storage: &StorageConfig{
			MinimumStorageBytes: &minStorage,
			MaximumStorageBytes: &maxStorage,
		},
	})

	want := &Database{
		Name:  testDatabase,
		State: DatabaseReady,
		Kind:  DatabaseEnginePostgres,
		Region: Region{
			Slug: "us-west",
			Name: "US West",
		},
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabases_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"planetscale-go-test-db","type":"database","name":"planetscale-go-test-db","notes":"This is a test DB created from the planetscale-go API library","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"
	notes := "This is a test DB created from the planetscale-go API library"

	db, err := client.Databases.Get(ctx, &GetDatabaseRequest{
		Organization: org,
		Database:     name,
	})

	want := &Database{
		Name:      name,
		Notes:     notes,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabases_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"data":[{"id":"planetscale-go-test-db","type":"database", "name":"planetscale-go-test-db","notes":"This is a test DB created from the planetscale-go API library","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"
	notes := "This is a test DB created from the planetscale-go API library"

	db, err := client.Databases.List(ctx, &ListDatabasesRequest{
		Organization: org,
	})

	want := []*Database{{
		Name:      name,
		Notes:     notes,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabases_ListWithOptions(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"data":[{"id":"planetscale-go-test-db","type":"database", "name":"planetscale-go-test-db","notes":"This is a test DB created from the planetscale-go API library","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
		c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "100")
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"
	notes := "This is a test DB created from the planetscale-go API library"

	db, err := client.Databases.List(ctx, &ListDatabasesRequest{
		Organization: org,
	}, WithPage(2))

	want := []*Database{{
		Name:      name,
		Notes:     notes,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabases_DeleteNoContent(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		_, err := w.Write(nil)
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"

	dbr, err := client.Databases.Delete(ctx, &DeleteDatabaseRequest{
		Organization: org,
		Database:     "planetscale-go-test-db",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(dbr, qt.IsNil)
}

func TestDatabases_DeleteAccepted(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		out := `{"id": "planetscale-go-test-db"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"

	dbr, err := client.Databases.Delete(ctx, &DeleteDatabaseRequest{
		Organization: org,
		Database:     "planetscale-go-test-db",
	})

	want := &DatabaseDeletionRequest{
		ID: "planetscale-go-test-db",
	}

	c.Assert(err, qt.IsNil)
	c.Assert(dbr, qt.DeepEquals, want)
}

func TestDatabases_List_malformed_response(t *testing.T) {
	c := qt.New(t)

	malformedBody := `<html><head><title>400 Bad Request</title></head>
<body> <hr><center>nginx</center></body></html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, err := w.Write([]byte(malformedBody))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"

	_, err = client.Databases.List(ctx, &ListDatabasesRequest{
		Organization: org,
	})

	c.Assert(err, qt.Not(qt.IsNil))
	c.Assert(err.Error(), qt.Contains, "received HTTP 400 with a malformed error response body")
	c.Assert(err.Error(), qt.Contains, "400 Bad Request")
}

func TestDatabases_Empty(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"data":[]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"

	db, err := client.Databases.List(ctx, &ListDatabasesRequest{
		Organization: org,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.HasLen, 0)
}
