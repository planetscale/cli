package planetscale

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

const testPostgresBranch = "postgres-test-branch"

func TestPostgresBranches_Create(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"postgres-test-branch","name":"postgres-test-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z", "region": {"slug": "us-west", "display_name": "US West"}}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "postgres-test-db"

	branch, err := client.PostgresBranches.Create(ctx, &CreatePostgresBranchRequest{
		Organization: org,
		Database:     name,
		Region:       "us-west",
		Name:         testPostgresBranch,
		ParentBranch: "main",
	})

	want := &PostgresBranch{
		ID:   "postgres-test-branch",
		Name: testPostgresBranch,
		Region: Region{
			Slug: "us-west",
			Name: "US West",
		},
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(branch, qt.DeepEquals, want)
}

func TestPostgresBranches_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"data":[{"id":"postgres-test-branch","name":"postgres-test-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "postgres-test-db"

	branches, err := client.PostgresBranches.List(ctx, &ListPostgresBranchesRequest{
		Organization: org,
		Database:     name,
	})

	want := []*PostgresBranch{{
		ID:        "postgres-test-branch",
		Name:      testPostgresBranch,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}}

	c.Assert(err, qt.IsNil)
	c.Assert(branches, qt.DeepEquals, want)
}

func TestPostgresBranches_ListWithPagination(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Query().Get("limit"), qt.Equals, "10")
		c.Assert(r.URL.Query().Get("starting_after"), qt.Equals, "test-branch")
		w.WriteHeader(200)
		out := `{"data":[{"id":"postgres-test-branch","name":"postgres-test-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "postgres-test-db"

	branches, err := client.PostgresBranches.List(ctx, &ListPostgresBranchesRequest{
		Organization: org,
		Database:     name,
	}, WithLimit(10), WithStartingAfter("test-branch"))

	want := []*PostgresBranch{{
		ID:        "postgres-test-branch",
		Name:      testPostgresBranch,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}}

	c.Assert(err, qt.IsNil)
	c.Assert(branches, qt.DeepEquals, want)
}

func TestPostgresBranches_ListWithDefaultPerPage(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "100")
		w.WriteHeader(200)
		out := `{"data":[{"id":"postgres-test-branch","name":"postgres-test-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "postgres-test-db"

	branches, err := client.PostgresBranches.List(ctx, &ListPostgresBranchesRequest{
		Organization: org,
		Database:     name,
	})

	want := []*PostgresBranch{{
		ID:        "postgres-test-branch",
		Name:      testPostgresBranch,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}}

	c.Assert(err, qt.IsNil)
	c.Assert(branches, qt.DeepEquals, want)
}

func TestPostgresBranches_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"postgres-test-branch","name":"postgres-test-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z","replicas":1}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "postgres-test-db"

	branch, err := client.PostgresBranches.Get(ctx, &GetPostgresBranchRequest{
		Organization: org,
		Database:     name,
		Branch:       testPostgresBranch,
	})

	want := &PostgresBranch{
		ID:        "postgres-test-branch",
		Name:      testPostgresBranch,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		Replicas:  1,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(branch, qt.DeepEquals, want)
}

func TestPostgresBranches_Delete(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/postgres-test-db/branches/postgres-test-branch")
		c.Assert(r.URL.Query().Get("delete_descendants"), qt.Equals, "")
		w.WriteHeader(204)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	err = client.PostgresBranches.Delete(ctx, &DeletePostgresBranchRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Branch:       testPostgresBranch,
	})

	c.Assert(err, qt.IsNil)
}

func TestPostgresBranches_DeleteWithDescendants(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/postgres-test-db/branches/postgres-test-branch")
		c.Assert(r.URL.Query().Get("delete_descendants"), qt.Equals, "true")
		w.WriteHeader(204)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	err = client.PostgresBranches.Delete(ctx, &DeletePostgresBranchRequest{
		Organization:      "my-org",
		Database:          "postgres-test-db",
		Branch:            testPostgresBranch,
		DeleteDescendants: true,
	})

	c.Assert(err, qt.IsNil)
}

func TestPostgresBranches_Schema(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"data": [{"name": "test_schema", "raw": "CREATE TABLE test...", "html": "<div>CREATE TABLE test...</div>"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	schemas, err := client.PostgresBranches.Schema(ctx, &PostgresBranchSchemaRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Branch:       testPostgresBranch,
	})

	want := []*PostgresBranchSchema{
		{
			Name: "test_schema",
			Raw:  "CREATE TABLE test...",
			HTML: "<div>CREATE TABLE test...</div>",
		},
	}

	c.Assert(err, qt.IsNil)
	c.Assert(schemas, qt.DeepEquals, want)
}

func TestPostgresBranches_ListClusterSKUs(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `[
		{
			"name": "PS_10",
			"type": "ClusterSizeSku",
			"display_name": "PS-10",
			"cpu": "1/8",
			"provider_instance_type": null,
			"storage": null,
			"ram": 1,
			"enabled": true,
			"provider": null,
			"rate": null,
			"replica_rate": null,
			"default_vtgate": "VTG_5",
			"default_vtgate_rate": null,
			"sort_order": 1
		}
	]`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	skus, err := client.PostgresBranches.ListClusterSKUs(ctx, &ListBranchClusterSKUsRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Branch:       testPostgresBranch,
	})

	want := []*ClusterSKU{
		{
			Name:          "PS_10",
			DisplayName:   "PS-10",
			CPU:           "1/8",
			Memory:        1,
			Enabled:       true,
			DefaultVTGate: "VTG_5",
			SortOrder:     1,
		},
	}

	c.Assert(err, qt.IsNil)
	c.Assert(skus, qt.DeepEquals, want)
}

func TestPostgresBranches_Resize(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/postgres-test-db/branches/postgres-test-branch/changes")

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["cluster_size"], qt.Equals, "PS_10_GCP_X86")
		// Replicas is unset, so it must be omitted from the request body.
		_, hasReplicas := body["replicas"]
		c.Assert(hasReplicas, qt.IsFalse)

		w.WriteHeader(200)
		out := `{"id":"resize-1","state":"queued","cluster_name":"PS_10_GCP_X86","cluster_display_name":"PS-10","replicas":0,"previous_cluster_name":"PS_5_GCP_X86","previous_cluster_display_name":"PS-5","previous_replicas":0,"created_at":"2026-06-24T10:19:23.000Z","updated_at":"2026-06-24T10:19:23.000Z"}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	change, err := client.PostgresBranches.Resize(ctx, &ResizePostgresBranchRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Branch:       testPostgresBranch,
		ClusterSize:  "PS_10_GCP_X86",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(change.ID, qt.Equals, "resize-1")
	c.Assert(change.State, qt.Equals, "queued")
	c.Assert(change.ClusterName, qt.Equals, "PS_10_GCP_X86")
	c.Assert(change.PreviousClusterName, qt.Equals, "PS_5_GCP_X86")
}

func TestPostgresBranches_ResizeNoOp(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/postgres-test-db/branches/postgres-test-branch/changes")
		// The requested configuration already matches: 204 No Content.
		w.WriteHeader(204)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	change, err := client.PostgresBranches.Resize(ctx, &ResizePostgresBranchRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Branch:       testPostgresBranch,
		ClusterSize:  "PS_5_GCP_X86",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(change, qt.IsNil)
}

func TestPostgresBranches_ResizeWithParameters(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/postgres-test-db/branches/postgres-test-branch/changes")

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)

		// Cluster size is unset, so it must be omitted from the request body.
		_, hasClusterSize := body["cluster_size"]
		c.Assert(hasClusterSize, qt.IsFalse)

		parameters, ok := body["parameters"].(map[string]any)
		c.Assert(ok, qt.IsTrue, qt.Commentf("parameters should be a nested object"))
		pgconf, ok := parameters["pgconf"].(map[string]any)
		c.Assert(ok, qt.IsTrue)
		c.Assert(pgconf["max_connections"], qt.Equals, "200")

		w.WriteHeader(200)
		out := `{"id":"change-1","state":"queued","parameters":{"pgconf":{"max_connections":"200"}},"previous_parameters":{"pgconf":{"max_connections":"100"}},"created_at":"2026-06-24T10:19:23.000Z","updated_at":"2026-06-24T10:19:23.000Z"}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	change, err := client.PostgresBranches.Resize(context.Background(), &ResizePostgresBranchRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Branch:       testPostgresBranch,
		Parameters: map[string]map[string]string{
			"pgconf": {"max_connections": "200"},
		},
	})

	c.Assert(err, qt.IsNil)
	c.Assert(change.ID, qt.Equals, "change-1")
	c.Assert(change.Parameters, qt.DeepEquals, map[string]map[string]any{"pgconf": {"max_connections": "200"}})
	c.Assert(change.PreviousParameters, qt.DeepEquals, map[string]map[string]any{"pgconf": {"max_connections": "100"}})
}

func TestPostgresBranches_ListChanges(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/postgres-test-db/branches/postgres-test-branch/changes")

		w.WriteHeader(200)
		out := `{"data":[{"id":"change-2","state":"queued","created_at":"2026-06-24T10:19:23.000Z","updated_at":"2026-06-24T10:19:23.000Z"},{"id":"change-1","state":"completed","created_at":"2026-06-23T10:19:23.000Z","updated_at":"2026-06-23T10:19:23.000Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	changes, err := client.PostgresBranches.ListChanges(context.Background(), &ListPostgresBranchChangesRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Branch:       testPostgresBranch,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(changes, qt.HasLen, 2)
	c.Assert(changes[0].ID, qt.Equals, "change-2")
	c.Assert(changes[0].Finished(), qt.IsFalse)
	c.Assert(changes[1].ID, qt.Equals, "change-1")
	c.Assert(changes[1].Finished(), qt.IsTrue)
}

func TestPostgresBranches_GetChange(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/postgres-test-db/branches/postgres-test-branch/changes/change-1")

		w.WriteHeader(200)
		out := `{"id":"change-1","state":"resizing","created_at":"2026-06-24T10:19:23.000Z","updated_at":"2026-06-24T10:19:23.000Z"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	change, err := client.PostgresBranches.GetChange(context.Background(), &GetPostgresBranchChangeRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Branch:       testPostgresBranch,
		ID:           "change-1",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(change.ID, qt.Equals, "change-1")
	c.Assert(change.State, qt.Equals, "resizing")
	c.Assert(change.Finished(), qt.IsFalse)
}

func TestPostgresBranches_CancelChanges(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/postgres-test-db/branches/postgres-test-branch/changes")
		w.WriteHeader(204)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	err = client.PostgresBranches.CancelChanges(context.Background(), &CancelPostgresBranchChangesRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Branch:       testPostgresBranch,
	})

	c.Assert(err, qt.IsNil)
}

func TestPostgresBranches_ListParameters(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/postgres-test-db/branches/postgres-test-branch/parameters")
		c.Assert(r.URL.Query().Has("extension"), qt.IsFalse)
		c.Assert(r.URL.Query().Has("internal"), qt.IsFalse)

		w.WriteHeader(200)
		// The parameters endpoint returns a bare array, not a {"data": [...]} envelope.
		out := `[{"id":"param-1","name":"max_connections","display_name":"Max connections","namespace":"pgconf","parameter_type":"integer","default_value":"100","value":"200","restart":true,"immutable":false,"min":25,"max":5000}]`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	parameters, err := client.PostgresBranches.ListParameters(context.Background(), &ListPostgresParametersRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Branch:       testPostgresBranch,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(parameters, qt.HasLen, 1)
	c.Assert(parameters[0].Name, qt.Equals, "max_connections")
	c.Assert(parameters[0].Namespace, qt.Equals, "pgconf")
	c.Assert(parameters[0].Value, qt.Equals, "200")
	c.Assert(parameters[0].Restart, qt.IsTrue)
	c.Assert(parameters[0].Min, qt.Equals, float64(25))
	c.Assert(parameters[0].Max, qt.Equals, float64(5000))
}

func TestPostgresBranches_ListExtensions(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/postgres-test-db/branches/postgres-test-branch/extensions")
		w.WriteHeader(200)
		out := `[{"type":"PostgresClusterExtension","name":"vector","description":"<p>vector</p>","internal":false,"loader":"shared_preload_libraries","url":"https://github.com/pgvector/pgvector","available":true,"unavailable_reason":"","parameters":[]}]`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	extensions, err := client.PostgresBranches.ListExtensions(context.Background(), &ListPostgresExtensionsRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Branch:       testPostgresBranch,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(extensions, qt.HasLen, 1)
	c.Assert(extensions[0].Name, qt.Equals, "vector")
	c.Assert(extensions[0].Loader, qt.Equals, "shared_preload_libraries")
	c.Assert(extensions[0].Available, qt.IsTrue)
	c.Assert(extensions[0].URL, qt.Equals, "https://github.com/pgvector/pgvector")
}

func TestPostgresBranches_ListParametersWithFilters(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Query().Get("extension"), qt.Equals, "false")
		c.Assert(r.URL.Query().Get("internal"), qt.Equals, "true")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`[]`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	extension := false
	internal := true
	parameters, err := client.PostgresBranches.ListParameters(context.Background(), &ListPostgresParametersRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Branch:       testPostgresBranch,
		Extension:    &extension,
		Internal:     &internal,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(parameters, qt.HasLen, 0)
}

func TestPostgresBranches_ResizeParameterValidationError(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		out := `{"errors":[{"namespace":"pgconf","name":"max_connections","errors":["Value must be less than or equal to 100"]}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	_, err = client.PostgresBranches.Resize(context.Background(), &ResizePostgresBranchRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Branch:       testPostgresBranch,
		Parameters: map[string]map[string]string{
			"pgconf": {"max_connections": "200"},
		},
	})

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "pgconf.max_connections: Value must be less than or equal to 100")

	var psErr *Error
	c.Assert(errors.As(err, &psErr), qt.IsTrue)
	c.Assert(psErr.Code, qt.Equals, ErrInvalid)
}

func TestPostgresBranches_CreateWithStorage(t *testing.T) {
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

		out := `{"id":"postgres-test-branch","name":"postgres-test-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z", "region": {"slug": "us-west", "display_name": "US West"}}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	minStorage := int64(10737418240)
	maxStorage := int64(107374182400)

	branch, err := client.PostgresBranches.Create(ctx, &CreatePostgresBranchRequest{
		Organization: "my-org",
		Database:     "postgres-test-db",
		Region:       "us-west",
		Name:         testPostgresBranch,
		ParentBranch: "main",
		Storage: &StorageConfig{
			MinimumStorageBytes: &minStorage,
			MaximumStorageBytes: &maxStorage,
		},
	})

	want := &PostgresBranch{
		ID:   "postgres-test-branch",
		Name: testPostgresBranch,
		Region: Region{
			Slug: "us-west",
			Name: "US West",
		},
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(branch, qt.DeepEquals, want)
}
