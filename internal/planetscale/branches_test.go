package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

const testBranch = "planetscale-go-test-db-branch"

func TestDatabaseBranches_Create(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"planetscale-go-test-db-branch","type":"database_branch","name":"planetscale-go-test-db-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z", "region": {"slug": "us-west", "display_name": "US West"}}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"

	db, err := client.DatabaseBranches.Create(ctx, &CreateDatabaseBranchRequest{
		Organization: org,
		Database:     name,
		Region:       "us-west",
		Name:         testBranch,
	})

	want := &DatabaseBranch{
		ID:   "planetscale-go-test-db-branch",
		Name: testBranch,
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

func TestDatabaseBranches_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"data":[{"id":"planetscale-go-test-db-branch","type":"database_branch","name":"planetscale-go-test-db-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"

	db, err := client.DatabaseBranches.List(ctx, &ListDatabaseBranchesRequest{
		Organization: org,
		Database:     name,
	})

	want := []*DatabaseBranch{{
		ID:        "planetscale-go-test-db-branch",
		Name:      testBranch,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabaseBranches_ListEmpty(t *testing.T) {
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
	name := "planetscale-go-test-db"

	db, err := client.DatabaseBranches.List(ctx, &ListDatabaseBranchesRequest{
		Organization: org,
		Database:     name,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.HasLen, 0)
}

func TestDatabaseBranches_ListWithPagination(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Query().Get("limit"), qt.Equals, "10")
		c.Assert(r.URL.Query().Get("starting_after"), qt.Equals, "test-branch")
		w.WriteHeader(200)
		out := `{"data":[{"id":"planetscale-go-test-db-branch","type":"database_branch","name":"planetscale-go-test-db-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"

	db, err := client.DatabaseBranches.List(ctx, &ListDatabaseBranchesRequest{
		Organization: org,
		Database:     name,
	}, WithLimit(10), WithStartingAfter("test-branch"))

	want := []*DatabaseBranch{{
		ID:        "planetscale-go-test-db-branch",
		Name:      testBranch,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabaseBranches_ListWithDefaultPerPage(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "100")
		w.WriteHeader(200)
		out := `{"data":[{"id":"planetscale-go-test-db-branch","type":"database_branch","name":"planetscale-go-test-db-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"

	db, err := client.DatabaseBranches.List(ctx, &ListDatabaseBranchesRequest{
		Organization: org,
		Database:     name,
	})

	want := []*DatabaseBranch{{
		ID:        "planetscale-go-test-db-branch",
		Name:      testBranch,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabaseBranches_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"planetscale-go-test-db-branch","type":"database_branch","name":"planetscale-go-test-db-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"

	db, err := client.DatabaseBranches.Get(ctx, &GetDatabaseBranchRequest{
		Organization: org,
		Database:     name,
		Branch:       testBranch,
	})

	want := &DatabaseBranch{
		ID:        "planetscale-go-test-db-branch",
		Name:      testBranch,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestBranches_Diff(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"data":[{"name": "foo"}, {"name": "bar"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	diffs, err := client.DatabaseBranches.Diff(ctx, &DiffBranchRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
	})

	want := []*Diff{
		{Name: "foo"},
		{Name: "bar"},
	}

	c.Assert(err, qt.IsNil)
	c.Assert(diffs, qt.DeepEquals, want)
}

func TestBranches_Schema(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"data":[{"name": "foo"}, {"name": "bar"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	schemas, err := client.DatabaseBranches.Schema(ctx, &BranchSchemaRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
	})

	want := []*Diff{
		{Name: "foo"},
		{Name: "bar"},
	}

	c.Assert(err, qt.IsNil)
	c.Assert(schemas, qt.DeepEquals, want)
}

func TestBranches_RefreshSchema(t *testing.T) {
	c := qt.New(t)

	wantURL := "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/planetscale-go-test-db-branch/refresh-schema"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		c.Assert(r.URL.String(), qt.DeepEquals, wantURL)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	err = client.DatabaseBranches.RefreshSchema(ctx, &RefreshSchemaRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       testBranch,
	})
	c.Assert(err, qt.IsNil)
}

func TestBranches_Demote(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"planetscale-go-test-db-branch","type":"database_branch","name":"planetscale-go-test-db-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z","production": false}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "my-test-db"
	branch := "main"

	b, err := client.DatabaseBranches.Demote(ctx, &DemoteRequest{
		Organization: org,
		Database:     name,
		Branch:       branch,
	})

	want := &DatabaseBranch{
		ID:         "planetscale-go-test-db-branch",
		Name:       testBranch,
		Production: false,
		CreatedAt:  time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt:  time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(b, qt.DeepEquals, want)
}

func TestDatabaseBranches_Promote(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"planetscale-go-test-db-branch","type":"database_branch","name":"planetscale-go-test-db-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z","production": true}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"

	db, err := client.DatabaseBranches.Promote(ctx, &PromoteRequest{
		Organization: org,
		Database:     name,
		Branch:       testBranch,
	})

	want := &DatabaseBranch{
		ID:         "planetscale-go-test-db-branch",
		Name:       testBranch,
		Production: true,
		CreatedAt:  time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt:  time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabaseBranches_EnableSafeMigrations(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"planetscale-go-test-db-branch","type":"database_branch","name":"planetscale-go-test-db-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z","production": true,"safe_migrations":true}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"

	db, err := client.DatabaseBranches.EnableSafeMigrations(ctx, &EnableSafeMigrationsRequest{
		Organization: org,
		Database:     name,
		Branch:       testBranch,
	})

	want := &DatabaseBranch{
		ID:             "planetscale-go-test-db-branch",
		Name:           testBranch,
		Production:     true,
		SafeMigrations: true,
		CreatedAt:      time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt:      time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabaseBranches_DisableSafeMigrations(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"planetscale-go-test-db-branch","type":"database_branch","name":"planetscale-go-test-db-branch","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z","production": true,"safe_migrations":false}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"

	db, err := client.DatabaseBranches.DisableSafeMigrations(ctx, &DisableSafeMigrationsRequest{
		Organization: org,
		Database:     name,
		Branch:       testBranch,
	})

	want := &DatabaseBranch{
		ID:             "planetscale-go-test-db-branch",
		Name:           testBranch,
		Production:     true,
		SafeMigrations: false,
		CreatedAt:      time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt:      time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(db, qt.DeepEquals, want)
}

func TestDatabaseBranches_LintSchema(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `
{
			"type": "list",
			"current_page": 1,
			"next_page": null,
			"next_page_url": null,
			"prev_page": null,
			"prev_page_url": null,
			"data": [
			{
			"type": "SchemaLintError",
			"lint_error": "NO_UNIQUE_KEY",
			"subject_type": "table_error",
			"keyspace_name": "test-database",
			"table_name": "test",
			"error_description": "table \"test\" has no unique key: all tables must have at least one unique, not-null key.",
			"docs_url": "https://planetscale.com/docs/learn/change-single-unique-key",
			"column_name": "",
			"foreign_key_column_names": [],
			"auto_increment_column_names": [],
			"charset_name": "",
			"engine_name": "",
			"vindex_name": null,
			"json_path": null,
			"check_constraint_name": "",
			"enum_value": "",
			"partitioning_type": "",
			"partition_name": "",
			"url_hash": "81e9ed9a459b5824393c4fa735753c49d47861bbd43742e2baa0ab7013158d2b"
			}
			]
			}
		`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	name := "planetscale-go-test-db"

	lintErrors, err := client.DatabaseBranches.LintSchema(ctx, &LintSchemaRequest{
		Organization: org,
		Database:     name,
		Branch:       testBranch,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(len(lintErrors), qt.Equals, 1)

	want := &SchemaLintError{
		LintError:        "NO_UNIQUE_KEY",
		SubjectType:      "table_error",
		Keyspace:         "test-database",
		Table:            "test",
		ErrorDescription: "table \"test\" has no unique key: all tables must have at least one unique, not-null key.",
		DocsURL:          "https://planetscale.com/docs/learn/change-single-unique-key",
	}
	lintErr := lintErrors[0]

	c.Assert(lintErr, qt.DeepEquals, want)
}

func TestDatabaseBranches_Delete(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/planetscale-go-test-db-branch")
		c.Assert(r.URL.Query().Get("delete_descendants"), qt.Equals, "")
		w.WriteHeader(204)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	err = client.DatabaseBranches.Delete(ctx, &DeleteDatabaseBranchRequest{
		Organization: "my-org",
		Database:     "planetscale-go-test-db",
		Branch:       testBranch,
	})

	c.Assert(err, qt.IsNil)
}

func TestDatabaseBranches_DeleteWithDescendants(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/planetscale-go-test-db-branch")
		c.Assert(r.URL.Query().Get("delete_descendants"), qt.Equals, "true")
		w.WriteHeader(204)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	err = client.DatabaseBranches.Delete(ctx, &DeleteDatabaseBranchRequest{
		Organization:      "my-org",
		Database:          "planetscale-go-test-db",
		Branch:            testBranch,
		DeleteDescendants: true,
	})

	c.Assert(err, qt.IsNil)
}

func TestDatabaseBranches_ListClusterSKUs(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-cool-org/databases/my-cool-db/branches/main/cluster-size-skus")
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

	orgs, err := client.DatabaseBranches.ListClusterSKUs(ctx, &ListBranchClusterSKUsRequest{
		Organization: "my-cool-org",
		Database:     "my-cool-db",
		Branch:       "main",
	})

	c.Assert(err, qt.IsNil)
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

	c.Assert(orgs, qt.DeepEquals, want)
}

func TestDatabaseBranches_ListClusterSKUsWithRates(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-cool-org/databases/my-cool-db/branches/main/cluster-size-skus?rates=true")
		out := `[
		{
			"name": "PS_10",
			"type": "ClusterSizeSku",
			"display_name": "PS-10",
			"cpu": "1/8",
			"provider_instance_type": null,
			"storage": 100,
			"ram": 1,
			"sort_order": 1,
			"enabled": true,
			"provider": null,
			"rate": 39,
			"replica_rate": 13,
			"default_vtgate": "VTG_5",
			"default_vtgate_rate": null
		}
	]`

		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	orgs, err := client.DatabaseBranches.ListClusterSKUs(ctx, &ListBranchClusterSKUsRequest{
		Organization: "my-cool-org",
		Database:     "my-cool-db",
		Branch:       "main",
	}, WithRates())

	c.Assert(err, qt.IsNil)
	want := []*ClusterSKU{
		{
			Name:          "PS_10",
			DisplayName:   "PS-10",
			CPU:           "1/8",
			Memory:        1,
			Enabled:       true,
			Storage:       Pointer[int64](100),
			Rate:          Pointer[int64](39),
			ReplicaRate:   Pointer[int64](13),
			DefaultVTGate: "VTG_5",
			SortOrder:     1,
		},
	}

	c.Assert(orgs, qt.DeepEquals, want)
}

func TestDatabaseBranches_Resize(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"resize-id","type":"BranchResizeRequest","state":"pending","vtgate_size":"vg.c1.xlarge","previous_vtgate_size":"vg.c1.nano","vtgate_name":"VTG_320","previous_vtgate_name":"VTG_5","vtgate_display_name":"VTG-320","previous_vtgate_display_name":"VTG-5","vtgate_count":2,"previous_vtgate_count":1,"vtgate_autoscaling":false,"previous_vtgate_autoscaling":false,"created_at":"2024-06-25T18:03:09.439Z","updated_at":"2024-06-25T18:03:09.439Z","started_at":"2024-06-25T18:03:09.459Z","completed_at":null}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
		c.Assert(r.Method, qt.Equals, http.MethodPut)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/foo/databases/bar/branches/baz/resizes")
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	count := 2
	resize, err := client.DatabaseBranches.Resize(ctx, &ResizeBranchRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
		VTGateSize:   "VTG_320",
		VTGateCount:  &count,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(resize.ID, qt.Equals, "resize-id")
	c.Assert(resize.State, qt.Equals, "pending")
	c.Assert(resize.VTGateName, qt.Equals, "VTG_320")
	c.Assert(resize.PreviousVTGateName, qt.Equals, "VTG_5")
	c.Assert(resize.VTGateCount, qt.Equals, 2)
	c.Assert(resize.PreviousVTGateCount, qt.Equals, 1)
}

func TestDatabaseBranches_ListResizes(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"data":[{"id":"resize-id","type":"BranchResizeRequest","state":"completed","vtgate_size":"vg.c1.xlarge","previous_vtgate_size":"vg.c1.nano","vtgate_name":"VTG_320","previous_vtgate_name":"VTG_5","vtgate_display_name":"VTG-320","previous_vtgate_display_name":"VTG-5","vtgate_count":1,"previous_vtgate_count":1,"vtgate_autoscaling":false,"previous_vtgate_autoscaling":false,"created_at":"2024-06-25T18:03:09.439Z","updated_at":"2024-06-25T18:04:06.238Z","started_at":"2024-06-25T18:03:09.459Z","completed_at":"2024-06-25T18:04:06.228Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/foo/databases/bar/branches/baz/resizes")
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	resizes, err := client.DatabaseBranches.ListResizes(ctx, &ListBranchResizesRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(resizes, qt.HasLen, 1)
	c.Assert(resizes[0].ID, qt.Equals, "resize-id")
	c.Assert(resizes[0].State, qt.Equals, "completed")
}

func TestDatabaseBranches_CancelResize(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/foo/databases/bar/branches/baz/resizes")
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	err = client.DatabaseBranches.CancelResize(ctx, &CancelBranchResizeRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
	})

	c.Assert(err, qt.IsNil)
}

func TestDatabaseBranches_ResizeStatus(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"data":[{"id":"latest","type":"BranchResizeRequest","state":"queued","vtgate_size":"vg.c1.2xlarge","previous_vtgate_size":"vg.c1.xlarge","vtgate_name":"VTG_640","previous_vtgate_name":"VTG_320","vtgate_display_name":"VTG-640","previous_vtgate_display_name":"VTG-320","vtgate_count":1,"previous_vtgate_count":1,"vtgate_autoscaling":false,"previous_vtgate_autoscaling":false,"created_at":"2024-06-25T18:03:09.439Z","updated_at":"2024-06-25T18:03:09.439Z","started_at":null,"completed_at":null}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	resize, err := client.DatabaseBranches.ResizeStatus(ctx, &BranchResizeStatusRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(resize.ID, qt.Equals, "latest")
	c.Assert(resize.State, qt.Equals, "queued")
	c.Assert(resize.VTGateName, qt.Equals, "VTG_640")
}

func TestDatabaseBranches_ResizeStatus_NotFound(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":[]}`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	resize, err := client.DatabaseBranches.ResizeStatus(ctx, &BranchResizeStatusRequest{
		Organization: "foo",
		Database:     "bar",
		Branch:       "baz",
	})

	wantError := &Error{
		msg:  "Not Found",
		Code: ErrNotFound,
	}

	c.Assert(resize, qt.IsNil)
	c.Assert(err.Error(), qt.Equals, wantError.Error())
}
