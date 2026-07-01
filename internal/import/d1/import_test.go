package d1

import (
	"context"
	"errors"
	"testing"
	"time"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type failingImportClient struct{}

func (failingImportClient) GetDatabase(context.Context, string, string) (*ps.Database, error) {
	return nil, errors.New("get database: boom")
}

func TestImportFailureReturnsPreparedResult(t *testing.T) {
	if _, err := FindPgloader(); err != nil {
		t.Skip("pgloader not installed")
	}
	t.Setenv("PSCALE_TEST_MODE", "1")

	prepared, err := PrepareImport(ImportOptions{
		InputPath: testFixture(t),
		Org:       "acme",
		Database:  "mydb",
		Branch:    "main",
	})
	if err != nil {
		t.Fatalf("PrepareImport: %v", err)
	}

	result, err := Import(context.Background(), nil, failingImportClient{}, ImportOptions{
		Org:      "acme",
		Database: "mydb",
		Branch:   "main",
	}, prepared)
	if err == nil {
		t.Fatal("expected import failure")
	}
	if result == nil {
		t.Fatal("expected populated result on failure")
	}
	if result.MigrationID != prepared.MigrationID {
		t.Fatalf("migration_id = %q, want %q", result.MigrationID, prepared.MigrationID)
	}
	if result.Method != prepared.Method {
		t.Fatalf("method = %q, want %q", result.Method, prepared.Method)
	}
	if result.Lint == nil || result.Plan == nil {
		t.Fatal("expected lint and plan on failure result")
	}
}

func TestImportResumeEnabled(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	migrationID := "resume001"
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
	}

	if enabled, err := importSchemaResumeEnabled(context.Background(), opts, ""); err != nil || enabled {
		t.Fatalf("expected no schema resume before schema is applied (enabled=%v err=%v)", enabled, err)
	}
	if enabled, err := importDataResumeEnabled(context.Background(), opts, ""); err != nil || enabled {
		t.Fatalf("expected no data resume before any tables loaded (enabled=%v err=%v)", enabled, err)
	}

	if err := updateMigrationState(org, database, branch, migrationID, func(state *MigrationState) {
		state.Phase = PhaseFailed
		state.SchemaApplied = true
	}); err != nil {
		t.Fatalf("update state schema applied: %v", err)
	}
	if enabled, err := importSchemaResumeEnabled(context.Background(), opts, ""); err != nil || enabled {
		t.Fatalf("expected schema resume deferred until destination is known (enabled=%v err=%v)", enabled, err)
	}
	if !shouldPreserveImportProgress(context.Background(), opts, "") {
		t.Fatal("expected import progress preserved when failed after schema applied")
	}

	if err := updateMigrationState(org, database, branch, migrationID, func(state *MigrationState) {
		state.SchemaApplied = false
		state.LoadedTables = []string{"users"}
	}); err != nil {
		t.Fatalf("update state: %v", err)
	}
	if enabled, err := importDataResumeEnabled(context.Background(), opts, ""); err != nil || enabled {
		t.Fatalf("expected data resume deferred until destination is known (enabled=%v err=%v)", enabled, err)
	}
	if !shouldPreserveImportProgress(context.Background(), opts, "") {
		t.Fatal("expected import progress preserved when failed with loaded tables")
	}

	if err := SetMigrationPhase(org, database, branch, migrationID, PhasePlanned); err != nil {
		t.Fatalf("SetMigrationPhase: %v", err)
	}
	if enabled, err := importDataResumeEnabled(context.Background(), opts, ""); err != nil || enabled {
		t.Fatalf("expected no data resume from planned phase even with loaded_tables (enabled=%v err=%v)", enabled, err)
	}
}

func TestSkipLoadedTablesForResume(t *testing.T) {
	loaded := []string{"users", "posts", "comments"}
	withRows := map[string]struct{}{
		"users": {},
		"posts": {},
	}
	got := skipLoadedTablesForResume(loaded, withRows)
	want := []string{"users", "posts"}
	if len(got) != len(want) {
		t.Fatalf("skip tables = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("skip tables = %v, want %v", got, want)
		}
	}
}

func TestAppendLoadedTableAndReset(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	migrationID := "resume002"
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
		DBName:      "myapp",
	}
	if err := appendLoadedTable(opts, "users"); err != nil {
		t.Fatalf("appendLoadedTable: %v", err)
	}
	if err := appendLoadedTable(opts, "users"); err != nil {
		t.Fatalf("appendLoadedTable duplicate: %v", err)
	}
	if err := appendLoadedTable(opts, "posts"); err != nil {
		t.Fatalf("appendLoadedTable posts: %v", err)
	}

	state, err := LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(state.LoadedTables) != 2 {
		t.Fatalf("loaded_tables = %v, want [users posts]", state.LoadedTables)
	}

	if err := resetImportProgress(opts, PhaseImporting, "/tmp/stage.sqlite"); err != nil {
		t.Fatalf("resetImportProgress: %v", err)
	}
	state, err = LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState after reset: %v", err)
	}
	if len(state.LoadedTables) != 0 {
		t.Fatalf("expected loaded_tables cleared, got %v", state.LoadedTables)
	}
	if state.SchemaApplied {
		t.Fatal("expected schema_applied cleared on reset")
	}
	if state.DBName != "myapp" {
		t.Fatalf("db_name = %q, want myapp", state.DBName)
	}
	if state.SQLitePath != "/tmp/stage.sqlite" {
		t.Fatalf("sqlite_path = %q", state.SQLitePath)
	}
}

func TestIsStaleImportRole(t *testing.T) {
	current := "pscale_api_current.m82qolzonhmz"
	tests := []struct {
		name string
		role *ps.PostgresRole
		want bool
	}{
		{
			name: "current import role",
			role: &ps.PostgresRole{Name: "d1-import-1710000000", Username: current},
			want: false,
		},
		{
			name: "stale prior import role",
			role: &ps.PostgresRole{Name: "d1-import-1709999999", Username: "pscale_api_old.m82qolzonhmz"},
			want: true,
		},
		{
			name: "shell role",
			role: &ps.PostgresRole{Name: "pscale-cli-shell-abc", Username: "pscale_api_shell.m82qolzonhmz"},
			want: false,
		},
		{
			name: "manual admin role",
			role: &ps.PostgresRole{Name: "schema-reset", Username: "pscale_api_admin.m82qolzonhmz"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStaleImportRole(tt.role, current); got != tt.want {
				t.Fatalf("isStaleImportRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDefaultPostgresRole(t *testing.T) {
	if !isDefaultPostgresRole("postgres") {
		t.Fatal("expected postgres to match")
	}
	if !isDefaultPostgresRole("postgres.qi39g1yfjxyj") {
		t.Fatal("expected routed postgres role to match")
	}
	if isDefaultPostgresRole("pscale_api_abc123") {
		t.Fatal("pscale_api should not match")
	}
}

func TestWithConnectionRetryRecovers(t *testing.T) {
	attempts := 0
	err := withConnectionRetry(context.Background(), func() error {
		attempts++
		if attempts < 2 {
			return errors.New("bad connection")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withConnectionRetry: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestWithConnectionRetryNonRetryable(t *testing.T) {
	attempts := 0
	want := errors.New("syntax error at line 1")
	err := withConnectionRetry(context.Background(), func() error {
		attempts++
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestWithConnectionRetryRespectsContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := withConnectionRetry(ctx, func() error {
		return errors.New("bad connection")
	})
	if err == nil {
		t.Fatal("expected context or retry error")
	}
}
