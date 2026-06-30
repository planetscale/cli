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
