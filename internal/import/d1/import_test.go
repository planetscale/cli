package d1

import (
	"context"
	"errors"
	"testing"

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

func TestIsEphemeralImportRole(t *testing.T) {
	if !isEphemeralImportRole("pscale_api_abc123") {
		t.Fatal("expected pscale_api role to match")
	}
	if isEphemeralImportRole("postgres") {
		t.Fatal("postgres should not match")
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
