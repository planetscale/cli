package d1

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestCompleteRequiresVerifiedPhase(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	migrationID := "complete-unverified"
	if err := SavePlan(&PlanResult{
		MigrationID: migrationID,
		Org:         org,
		Database:    database,
		Branch:      branch,
		InputPath:   testFixture(t),
	}); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if err := SetMigrationPhase(org, database, branch, migrationID, PhaseImported); err != nil {
		t.Fatalf("SetMigrationPhase: %v", err)
	}

	if err := Complete(org, database, branch, migrationID, NotifyAPIConfig{}); err == nil {
		t.Fatal("expected error completing unverified migration")
	}
}

func TestMigrationPhaseTransitions(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	migrationID := "testphase123"

	plan := &PlanResult{
		MigrationID: migrationID,
		Org:         org,
		Database:    database,
		Branch:      branch,
		InputPath:   testFixture(t),
	}
	if err := SavePlan(plan); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	state, err := LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase != PhasePlanned {
		t.Fatalf("phase = %q, want %q", state.Phase, PhasePlanned)
	}

	opts := ImportOptions{
		Org:         org,
		Database:    database,
		Branch:      branch,
		MigrationID: migrationID,
		InputPath:   plan.InputPath,
		Method:      MethodPgloader,
	}
	if err := saveImportMigrationState(opts, PhaseImporting, ""); err != nil {
		t.Fatalf("saveImportMigrationState importing: %v", err)
	}
	state, err = LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState importing: %v", err)
	}
	if state.Phase != PhaseImporting {
		t.Fatalf("phase = %q, want %q", state.Phase, PhaseImporting)
	}

	if err := SetMigrationPhase(org, database, branch, migrationID, PhaseImported); err != nil {
		t.Fatalf("SetMigrationPhase imported: %v", err)
	}
	if err := SetMigrationPhase(org, database, branch, migrationID, PhaseVerified); err != nil {
		t.Fatalf("SetMigrationPhase verified: %v", err)
	}
	if err := Complete(org, database, branch, migrationID, NotifyAPIConfig{}); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	state, err = LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState complete: %v", err)
	}
	if state.Phase != PhaseComplete {
		t.Fatalf("phase = %q, want %q", state.Phase, PhaseComplete)
	}
}

func TestComplete_SucceedsWhenNotifyAPIFails(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client, err := ps.NewClient(
		ps.WithBaseURL(srv.URL),
		ps.WithAccessToken("token"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	org, database, branch := "acme", "mydb", "main"
	migrationID := "notifyfail789"
	if err := SavePlan(&PlanResult{
		MigrationID: migrationID,
		Org:         org,
		Database:    database,
		Branch:      branch,
		InputPath:   testFixture(t),
	}); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if err := SetMigrationPhase(org, database, branch, migrationID, PhaseVerified); err != nil {
		t.Fatalf("SetMigrationPhase verified: %v", err)
	}

	if err := Complete(org, database, branch, migrationID, NotifyAPIConfig{Client: client}); err != nil {
		t.Fatalf("Complete should succeed when notify API fails: %v", err)
	}

	state, err := LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase != PhaseComplete {
		t.Fatalf("phase = %q, want %q", state.Phase, PhaseComplete)
	}
}

func TestComplete_SendsCompletePayload(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("unmarshal body: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	client, err := ps.NewClient(
		ps.WithBaseURL(srv.URL),
		ps.WithAccessToken("token"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	org, database, branch := "acme", "mydb", "main"
	migrationID := "completepayload123"
	inputPath := testFixture(t)
	if err := SavePlan(&PlanResult{
		MigrationID: migrationID,
		Org:         org,
		Database:    database,
		Branch:      branch,
		InputPath:   inputPath,
	}); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	state, err := LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	state.Method = MethodPgloader
	state.LoadedTables = []string{"organizations", "users"}
	state.CreatedAt = time.Now().UTC().Add(-5 * time.Minute)
	if err := SaveState(state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if err := SetMigrationPhase(org, database, branch, migrationID, PhaseVerified); err != nil {
		t.Fatalf("SetMigrationPhase verified: %v", err)
	}

	if err := Complete(org, database, branch, migrationID, NotifyAPIConfig{Client: client}); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if body["event"] != NotifyEventComplete {
		t.Fatalf("event = %v, want %s", body["event"], NotifyEventComplete)
	}
	if body["method"] != MethodPgloader {
		t.Fatalf("method = %v, want %s", body["method"], MethodPgloader)
	}
	if body["table_count"].(float64) != 2 {
		t.Fatalf("table_count = %v, want 2", body["table_count"])
	}
	if body["export_bytes"].(float64) <= 0 {
		t.Fatalf("export_bytes = %v, want > 0", body["export_bytes"])
	}
	if body["duration_ms"].(float64) <= 0 {
		t.Fatalf("duration_ms = %v, want > 0", body["duration_ms"])
	}
	msg, _ := body["message"].(string)
	if msg == "" || !strings.Contains(msg, "re-baseline ORM migrations") {
		t.Fatalf("message = %v, want ORM re-baseline guidance", body["message"])
	}
}

func TestSaveImportMigrationStateFailed(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	migrationID := "testfailed456"
	if err := SavePlan(&PlanResult{
		MigrationID: migrationID,
		Org:         org,
		Database:    database,
		Branch:      branch,
		InputPath:   testFixture(t),
	}); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	opts := ImportOptions{
		Org:         org,
		Database:    database,
		Branch:      branch,
		MigrationID: migrationID,
		InputPath:   testFixture(t),
		Method:      MethodPgloader,
	}
	if err := saveImportMigrationState(opts, PhaseFailed, "/tmp/test.sqlite"); err != nil {
		t.Fatalf("saveImportMigrationState failed: %v", err)
	}

	state, err := LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase != PhaseFailed {
		t.Fatalf("phase = %q, want %q", state.Phase, PhaseFailed)
	}
	if state.SQLitePath != "/tmp/test.sqlite" {
		t.Fatalf("sqlite path = %q", state.SQLitePath)
	}
}
