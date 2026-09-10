package sqlquery

import (
	"context"
	"errors"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestIsReadQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{name: "select", query: "SELECT 1", want: true},
		{name: "leading line comment select", query: "-- load users\nSELECT 1", want: true},
		{name: "leading hash comment select", query: "# load users\nSELECT 1", want: true},
		{name: "leading block comment select", query: "/* load users */ SELECT 1", want: true},
		{name: "with select", query: "  with x as (select 1) select * from x", want: true},
		{name: "with multiple ctes select", query: "WITH a AS (SELECT 1), b AS (SELECT 2) SELECT * FROM b", want: true},
		{name: "with string containing paren", query: "WITH x AS (SELECT ')' AS val) SELECT * FROM x", want: true},
		{name: "insert", query: "INSERT INTO t VALUES (1)", want: false},
		{name: "update", query: "UPDATE t SET x = 1", want: false},
		{name: "with insert", query: "WITH x AS (SELECT 1) INSERT INTO t VALUES (1)", want: false},
		{name: "with update", query: "WITH x AS (SELECT 1) UPDATE t SET x = 1", want: false},
		{name: "with merge", query: "WITH x AS (SELECT 1) MERGE INTO t USING x ON t.id = x.id WHEN MATCHED THEN UPDATE SET x = 1", want: false},
		{name: "with materialized insert", query: "WITH x AS MATERIALIZED (SELECT 1) INSERT INTO t VALUES (1)", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReadQuery(tt.query); got != tt.want {
				t.Fatalf("isReadQuery(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestQueryReturnsRows(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{name: "insert returning", query: "INSERT INTO t VALUES (1) RETURNING id", want: true},
		{name: "update returning", query: "UPDATE t SET x = 1 RETURNING id", want: true},
		{name: "with insert returning", query: "WITH x AS (SELECT 1) INSERT INTO t VALUES (1) RETURNING id", want: true},
		{name: "returning in string", query: "SELECT 'RETURNING id' AS sample", want: false},
		{name: "returning in comment", query: "SELECT 1 -- RETURNING id", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := queryReturnsRows(tt.query); got != tt.want {
				t.Fatalf("queryReturnsRows(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestMySQLDSNDatabase(t *testing.T) {
	if got := mysqlDSNDatabase(Options{}); got != "@primary" {
		t.Fatalf("default = %q, want @primary", got)
	}
	if got := mysqlDSNDatabase(Options{Replica: true}); got != "" {
		t.Fatalf("replica = %q, want empty", got)
	}
	if got := mysqlDSNDatabase(Options{Keyspace: "commerce"}); got != "commerce" {
		t.Fatalf("explicit = %q", got)
	}
}

func TestExecuteValidation(t *testing.T) {
	ch := &cmdutil.Helper{
		Config: &config.Config{Organization: "bb"},
	}

	tests := []struct {
		name    string
		opts    Options
		wantErr string
	}{
		{
			name:    "missing query",
			opts:    Options{Organization: "bb", Database: "db", Branch: "main"},
			wantErr: "query is required",
		},
		{
			name:    "missing org",
			opts:    Options{Query: "SELECT 1", Database: "db", Branch: "main"},
			wantErr: "organization is required (use --org or set org in pscale.yml)",
		},
		{
			name:    "missing database",
			opts:    Options{Organization: "bb", Query: "SELECT 1", Branch: "main"},
			wantErr: "database and branch are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Execute(context.Background(), ch, tt.opts)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestOpenPostgresCleansUpRoleWhenReadinessWaitIsCanceled(t *testing.T) {
	getCalls := 0
	deleteCalls := 0

	roles := &mock.PostgresRolesService{
		CreateFn: func(_ context.Context, req *ps.CreatePostgresRoleRequest) (*ps.PostgresRole, error) {
			if req.Organization != "org" || req.Database != "database" || req.Branch != "main" {
				t.Fatalf("unexpected create request: %+v", req)
			}
			return &ps.PostgresRole{ID: "role-id", Ready: false}, nil
		},
		GetFn: func(_ context.Context, req *ps.GetPostgresRoleRequest) (*ps.PostgresRole, error) {
			getCalls++
			t.Fatalf("unexpected readiness request: %+v", req)
			return nil, nil
		},
		DeleteFn: func(_ context.Context, req *ps.DeletePostgresRoleRequest) error {
			deleteCalls++
			if req.Organization != "org" || req.Database != "database" || req.Branch != "main" || req.RoleId != "role-id" {
				t.Fatalf("unexpected delete request: %+v", req)
			}
			return nil
		},
	}

	format := printer.JSON
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Client: func() (*ps.Client, error) {
			return &ps.Client{PostgresRoles: roles}, nil
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	db, cleanup, err := openPostgres(ctx, ch, Options{
		Organization: "org",
		Database:     "database",
		Branch:       "main",
	}, "postgres", cmdutil.ReaderRole)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
	if db != nil {
		t.Fatalf("db = %v, want nil", db)
	}
	if cleanup != nil {
		t.Fatal("cleanup is not nil")
	}
	if getCalls != 0 {
		t.Fatalf("readiness checks = %d, want 0", getCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("role deletions = %d, want 1", deleteCalls)
	}
}
