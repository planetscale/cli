package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestMoveTables_Create(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["workflow"], qt.Equals, "my-workflow")
		c.Assert(body["target_keyspace"], qt.Equals, "target")
		c.Assert(body["source_keyspace"], qt.Equals, "source")

		_, hasAllTables := body["all_tables"]
		_, hasAutoStart := body["auto_start"]
		_, hasStopAfterCopy := body["stop_after_copy"]
		_, hasDeferSecondaryKeys := body["defer_secondary_keys"]
		_, hasAtomicCopy := body["atomic_copy"]
		_, hasCells := body["cells"]
		_, hasTabletTypes := body["tablet_types"]
		_, hasExcludeTables := body["exclude_tables"]
		_, hasTenantID := body["tenant_id"]
		_, hasGlobalKeyspace := body["global_keyspace"]
		c.Assert(hasAllTables, qt.IsFalse)
		c.Assert(hasAutoStart, qt.IsFalse)
		c.Assert(hasStopAfterCopy, qt.IsFalse)
		c.Assert(hasDeferSecondaryKeys, qt.IsFalse)
		c.Assert(hasAtomicCopy, qt.IsFalse)
		c.Assert(hasCells, qt.IsFalse)
		c.Assert(hasTabletTypes, qt.IsFalse)
		c.Assert(hasExcludeTables, qt.IsFalse)
		c.Assert(hasTenantID, qt.IsFalse)
		c.Assert(hasGlobalKeyspace, qt.IsFalse)

		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`{"id":"create-op"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	ref, err := client.MoveTables.Create(ctx, &MoveTablesCreateRequest{
		Organization:   "my-org",
		Database:       "my-db",
		Branch:         "my-branch",
		Workflow:       "my-workflow",
		TargetKeyspace: "target",
		SourceKeyspace: "source",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(ref, qt.DeepEquals, &VtctldOperationReference{ID: "create-op"})
}

func TestMoveTables_CreateWithTenantID(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["tenant_id"], qt.Equals, "tenant-123")

		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`{"id":"create-op"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	ref, err := client.MoveTables.Create(ctx, &MoveTablesCreateRequest{
		Organization:   "my-org",
		Database:       "my-db",
		Branch:         "my-branch",
		Workflow:       "my-workflow",
		TargetKeyspace: "target",
		SourceKeyspace: "source",
		TenantID:       "tenant-123",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(ref, qt.DeepEquals, &VtctldOperationReference{ID: "create-op"})
}

func TestMoveTables_CreateWithGlobalKeyspace(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["global_keyspace"], qt.Equals, "global")
		c.Assert(body["sharded_auto_increment_handling"], qt.Equals, "REPLACE")

		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`{"id":"create-op"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	ref, err := client.MoveTables.Create(ctx, &MoveTablesCreateRequest{
		Organization:                 "my-org",
		Database:                     "my-db",
		Branch:                       "my-branch",
		Workflow:                     "my-workflow",
		TargetKeyspace:               "target",
		SourceKeyspace:               "source",
		ShardedAutoIncrementHandling: "REPLACE",
		GlobalKeyspace:               "global",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(ref, qt.DeepEquals, &VtctldOperationReference{ID: "create-op"})
}

func TestMoveTables_CreateWithExplicitFalseValues(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)

		allTables, hasAllTables := body["all_tables"]
		autoStart, hasAutoStart := body["auto_start"]
		stopAfterCopy, hasStopAfterCopy := body["stop_after_copy"]
		deferSecondaryKeys, hasDeferSecondaryKeys := body["defer_secondary_keys"]
		atomicCopy, hasAtomicCopy := body["atomic_copy"]
		c.Assert(hasAllTables, qt.IsTrue)
		c.Assert(hasAutoStart, qt.IsTrue)
		c.Assert(hasStopAfterCopy, qt.IsTrue)
		c.Assert(hasDeferSecondaryKeys, qt.IsTrue)
		c.Assert(hasAtomicCopy, qt.IsTrue)
		c.Assert(allTables, qt.Equals, false)
		c.Assert(autoStart, qt.Equals, false)
		c.Assert(stopAfterCopy, qt.Equals, false)
		c.Assert(deferSecondaryKeys, qt.Equals, false)
		c.Assert(atomicCopy, qt.Equals, false)

		cells, hasCells := body["cells"]
		excludeTables, hasExcludeTables := body["exclude_tables"]
		tabletTypes, hasTabletTypes := body["tablet_types"]
		c.Assert(hasCells, qt.IsTrue)
		c.Assert(hasExcludeTables, qt.IsTrue)
		c.Assert(hasTabletTypes, qt.IsTrue)
		c.Assert(cells, qt.DeepEquals, []interface{}{"cell1", "cell2"})
		c.Assert(excludeTables, qt.DeepEquals, []interface{}{"internal_logs"})
		c.Assert(tabletTypes, qt.DeepEquals, []interface{}{"REPLICA", "RDONLY"})

		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`{"id":"create-op"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	falseValue := false
	ctx := context.Background()
	ref, err := client.MoveTables.Create(ctx, &MoveTablesCreateRequest{
		Organization:       "my-org",
		Database:           "my-db",
		Branch:             "my-branch",
		Workflow:           "my-workflow",
		TargetKeyspace:     "target",
		SourceKeyspace:     "source",
		Tables:             []string{"sales"},
		AllTables:          &falseValue,
		AutoStart:          &falseValue,
		StopAfterCopy:      &falseValue,
		DeferSecondaryKeys: &falseValue,
		AtomicCopy:         &falseValue,
		Cells:              []string{"cell1", "cell2"},
		ExcludeTables:      []string{"internal_logs"},
		TabletTypes:        []string{"REPLICA", "RDONLY"},
	})
	c.Assert(err, qt.IsNil)
	c.Assert(ref, qt.DeepEquals, &VtctldOperationReference{ID: "create-op"})
}

func TestMoveTables_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows")
		c.Assert(r.URL.Query().Get("target_keyspace"), qt.Equals, "target")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":{"workflows":[{"name":"my-workflow"}]}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.MoveTables.List(ctx, &MoveTablesListRequest{
		Organization:   "my-org",
		Database:       "my-db",
		Branch:         "my-branch",
		TargetKeyspace: "target",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"workflows":[{"name":"my-workflow"}]}`)
}

func TestMoveTables_ListSupportsLegacyResponseWithoutTargetKeyspace(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows")
		c.Assert(r.URL.Query().Get("target_keyspace"), qt.Equals, "")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`[]`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.MoveTables.List(ctx, &MoveTablesListRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `[]`)
}

func TestMoveTables_Show(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows/my-workflow")
		c.Assert(r.URL.Query().Get("target_keyspace"), qt.Equals, "target")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":{"result":"ok"}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.MoveTables.Show(ctx, &MoveTablesShowRequest{
		Organization:   "my-org",
		Database:       "my-db",
		Branch:         "my-branch",
		Workflow:       "my-workflow",
		TargetKeyspace: "target",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"result":"ok"}`)
}

func TestMoveTables_Status(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows/my-workflow/status")
		c.Assert(r.URL.Query().Get("target_keyspace"), qt.Equals, "target")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":{"result":"ok"}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.MoveTables.Status(ctx, &MoveTablesStatusRequest{
		Organization:   "my-org",
		Database:       "my-db",
		Branch:         "my-branch",
		Workflow:       "my-workflow",
		TargetKeyspace: "target",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"result":"ok"}`)
}

func TestMoveTables_Start(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows/my-workflow/start")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["target_keyspace"], qt.Equals, "target")

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"data":{"summary":"Streams started"}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.MoveTables.Start(ctx, &MoveTablesStartRequest{
		Organization:   "my-org",
		Database:       "my-db",
		Branch:         "my-branch",
		Workflow:       "my-workflow",
		TargetKeyspace: "target",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"summary":"Streams started"}`)
}

func TestMoveTables_SwitchTraffic(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows/my-workflow/switch-traffic")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["target_keyspace"], qt.Equals, "target")

		_, hasDryRun := body["dry_run"]
		_, hasInitializeTargetSequences := body["initialize_target_sequences"]
		_, hasMaxReplicationLagAllowed := body["max_replication_lag_allowed"]
		c.Assert(hasDryRun, qt.IsFalse)
		c.Assert(hasInitializeTargetSequences, qt.IsFalse)
		c.Assert(hasMaxReplicationLagAllowed, qt.IsFalse)

		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`{"id":"switch-op"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	ref, err := client.MoveTables.SwitchTraffic(ctx, &MoveTablesSwitchTrafficRequest{
		Organization:   "my-org",
		Database:       "my-db",
		Branch:         "my-branch",
		Workflow:       "my-workflow",
		TargetKeyspace: "target",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(ref, qt.DeepEquals, &VtctldOperationReference{ID: "switch-op"})
}

func TestMoveTables_SwitchTrafficWithExplicitFalseValues(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows/my-workflow/switch-traffic")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)

		dryRun, hasDryRun := body["dry_run"]
		initializeTargetSequences, hasInitializeTargetSequences := body["initialize_target_sequences"]
		maxReplicationLagAllowed, hasMaxReplicationLagAllowed := body["max_replication_lag_allowed"]
		c.Assert(hasDryRun, qt.IsTrue)
		c.Assert(hasInitializeTargetSequences, qt.IsTrue)
		c.Assert(hasMaxReplicationLagAllowed, qt.IsTrue)
		c.Assert(dryRun, qt.Equals, false)
		c.Assert(initializeTargetSequences, qt.Equals, false)
		c.Assert(maxReplicationLagAllowed, qt.Equals, float64(30))

		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`{"id":"switch-op"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	falseValue := false
	ctx := context.Background()
	maxLag := int64(30)
	ref, err := client.MoveTables.SwitchTraffic(ctx, &MoveTablesSwitchTrafficRequest{
		Organization:              "my-org",
		Database:                  "my-db",
		Branch:                    "my-branch",
		Workflow:                  "my-workflow",
		TargetKeyspace:            "target",
		MaxReplicationLagAllowed:  &maxLag,
		DryRun:                    &falseValue,
		InitializeTargetSequences: &falseValue,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(ref, qt.DeepEquals, &VtctldOperationReference{ID: "switch-op"})
}

func TestMoveTables_ReverseTraffic(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows/my-workflow/reverse-traffic")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["target_keyspace"], qt.Equals, "target")

		_, hasDryRun := body["dry_run"]
		_, hasTabletTypes := body["tablet_types"]
		_, hasMaxReplicationLagAllowed := body["max_replication_lag_allowed"]
		c.Assert(hasDryRun, qt.IsFalse)
		c.Assert(hasTabletTypes, qt.IsFalse)
		c.Assert(hasMaxReplicationLagAllowed, qt.IsFalse)

		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`{"id":"reverse-op"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	ref, err := client.MoveTables.ReverseTraffic(ctx, &MoveTablesReverseTrafficRequest{
		Organization:   "my-org",
		Database:       "my-db",
		Branch:         "my-branch",
		Workflow:       "my-workflow",
		TargetKeyspace: "target",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(ref, qt.DeepEquals, &VtctldOperationReference{ID: "reverse-op"})
}

func TestMoveTables_ReverseTrafficWithExplicitFalseValues(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows/my-workflow/reverse-traffic")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)

		dryRun, hasDryRun := body["dry_run"]
		c.Assert(hasDryRun, qt.IsTrue)
		c.Assert(dryRun, qt.Equals, false)

		tabletTypes, hasTabletTypes := body["tablet_types"]
		maxReplicationLagAllowed, hasMaxReplicationLagAllowed := body["max_replication_lag_allowed"]
		c.Assert(hasTabletTypes, qt.IsTrue)
		c.Assert(hasMaxReplicationLagAllowed, qt.IsTrue)
		c.Assert(tabletTypes, qt.DeepEquals, []interface{}{"REPLICA", "RDONLY"})
		c.Assert(maxReplicationLagAllowed, qt.Equals, float64(60))

		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`{"id":"reverse-op"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	falseValue := false
	ctx := context.Background()
	maxLag := int64(60)
	ref, err := client.MoveTables.ReverseTraffic(ctx, &MoveTablesReverseTrafficRequest{
		Organization:             "my-org",
		Database:                 "my-db",
		Branch:                   "my-branch",
		Workflow:                 "my-workflow",
		TargetKeyspace:           "target",
		TabletTypes:              []string{"REPLICA", "RDONLY"},
		MaxReplicationLagAllowed: &maxLag,
		DryRun:                   &falseValue,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(ref, qt.DeepEquals, &VtctldOperationReference{ID: "reverse-op"})
}

func TestMoveTables_Cancel(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows/my-workflow/cancel")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["target_keyspace"], qt.Equals, "target")

		_, hasKeepData := body["keep_data"]
		_, hasKeepRoutingRules := body["keep_routing_rules"]
		c.Assert(hasKeepData, qt.IsFalse)
		c.Assert(hasKeepRoutingRules, qt.IsFalse)

		w.WriteHeader(202)
		_, err = w.Write([]byte(`{"id":"cancel-op-123"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	op, err := client.MoveTables.Cancel(ctx, &MoveTablesCancelRequest{
		Organization:   "my-org",
		Database:       "my-db",
		Branch:         "my-branch",
		Workflow:       "my-workflow",
		TargetKeyspace: "target",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(op.ID, qt.Equals, "cancel-op-123")
}

func TestMoveTables_CancelWithExplicitFalseValues(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows/my-workflow/cancel")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)

		keepData, hasKeepData := body["keep_data"]
		keepRoutingRules, hasKeepRoutingRules := body["keep_routing_rules"]
		c.Assert(hasKeepData, qt.IsTrue)
		c.Assert(hasKeepRoutingRules, qt.IsTrue)
		c.Assert(keepData, qt.Equals, false)
		c.Assert(keepRoutingRules, qt.Equals, false)

		w.WriteHeader(202)
		_, err = w.Write([]byte(`{"id":"cancel-op-456"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	falseValue := false
	ctx := context.Background()
	op, err := client.MoveTables.Cancel(ctx, &MoveTablesCancelRequest{
		Organization:     "my-org",
		Database:         "my-db",
		Branch:           "my-branch",
		Workflow:         "my-workflow",
		TargetKeyspace:   "target",
		KeepData:         &falseValue,
		KeepRoutingRules: &falseValue,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(op.ID, qt.Equals, "cancel-op-456")
}

func TestMoveTables_Complete(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows/my-workflow/complete")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["target_keyspace"], qt.Equals, "target")

		_, hasKeepData := body["keep_data"]
		_, hasKeepRoutingRules := body["keep_routing_rules"]
		_, hasRenameTables := body["rename_tables"]
		_, hasDryRun := body["dry_run"]
		c.Assert(hasKeepData, qt.IsFalse)
		c.Assert(hasKeepRoutingRules, qt.IsFalse)
		c.Assert(hasRenameTables, qt.IsFalse)
		c.Assert(hasDryRun, qt.IsFalse)

		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`{"id":"complete-op"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	ref, err := client.MoveTables.Complete(ctx, &MoveTablesCompleteRequest{
		Organization:   "my-org",
		Database:       "my-db",
		Branch:         "my-branch",
		Workflow:       "my-workflow",
		TargetKeyspace: "target",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(ref, qt.DeepEquals, &VtctldOperationReference{ID: "complete-op"})
}

func TestMoveTables_CompleteWithExplicitFalseValues(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/move-tables/workflows/my-workflow/complete")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)

		keepData, hasKeepData := body["keep_data"]
		keepRoutingRules, hasKeepRoutingRules := body["keep_routing_rules"]
		renameTables, hasRenameTables := body["rename_tables"]
		dryRun, hasDryRun := body["dry_run"]
		c.Assert(hasKeepData, qt.IsTrue)
		c.Assert(hasKeepRoutingRules, qt.IsTrue)
		c.Assert(hasRenameTables, qt.IsTrue)
		c.Assert(hasDryRun, qt.IsTrue)
		c.Assert(keepData, qt.Equals, false)
		c.Assert(keepRoutingRules, qt.Equals, false)
		c.Assert(renameTables, qt.Equals, false)
		c.Assert(dryRun, qt.Equals, false)

		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`{"id":"complete-op"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	falseValue := false
	ctx := context.Background()
	ref, err := client.MoveTables.Complete(ctx, &MoveTablesCompleteRequest{
		Organization:     "my-org",
		Database:         "my-db",
		Branch:           "my-branch",
		Workflow:         "my-workflow",
		TargetKeyspace:   "target",
		KeepData:         &falseValue,
		KeepRoutingRules: &falseValue,
		RenameTables:     &falseValue,
		DryRun:           &falseValue,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(ref, qt.DeepEquals, &VtctldOperationReference{ID: "complete-op"})
}
