package sqlquery

import (
	"context"
	"errors"
	"testing"

	qt "github.com/frankban/quicktest"

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
	}, "postgres", cmdutil.ReaderRole, ps.DatabaseEnginePostgres)

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

func TestValidateEngineOptions(t *testing.T) {
	c := qt.New(t)

	c.Assert(validateEngineOptions(ps.DatabaseEngineNeki, Options{Shard: "shard-1"}), qt.IsNil)
	c.Assert(validateEngineOptions(ps.DatabaseEngineNeki, Options{Router: "default"}), qt.IsNil)
	c.Assert(validateEngineOptions(ps.DatabaseEngineNeki, Options{Shard: "shard-1", Router: "default"}), qt.IsNil)

	c.Assert(validateEngineOptions(ps.DatabaseEnginePostgres, Options{Shard: "shard-1"}), qt.ErrorMatches, `--shard/--router are only supported for Neki databases`)
	c.Assert(validateEngineOptions(ps.DatabaseEngineMySQL, Options{Router: "default"}), qt.ErrorMatches, `--shard/--router are only supported for Neki databases`)
	c.Assert(validateEngineOptions(ps.DatabaseEngineNeki, Options{Shard: "commerce/-80"}), qt.ErrorMatches, `--shard must be a Neki shard ID \(from pscale branch shard list\), not a Vitess keyspace/shard`)
	c.Assert(validateEngineOptions(ps.DatabaseEngineNeki, Options{Shard: "bad shard"}), qt.ErrorMatches, `invalid --shard "bad shard"`)
}

func TestPostgresUsername(t *testing.T) {
	c := qt.New(t)

	username, err := postgresUsername("role-abc", Options{}, ps.DatabaseEngineNeki)
	c.Assert(err, qt.IsNil)
	c.Assert(username, qt.Equals, "role-abc")

	username, err = postgresUsername("role-abc", Options{Replica: true}, ps.DatabaseEnginePostgres)
	c.Assert(err, qt.IsNil)
	c.Assert(username, qt.Equals, "role-abc|replica")

	username, err = postgresUsername("role-abc", Options{Replica: true}, ps.DatabaseEngineNeki)
	c.Assert(err, qt.IsNil)
	c.Assert(username, qt.Equals, "role-abc")

	username, err = postgresUsername("role-abc", Options{Router: "analytics"}, ps.DatabaseEngineNeki)
	c.Assert(err, qt.IsNil)
	c.Assert(username, qt.Equals, "role-abc|analytics")

	username, err = postgresUsername("role-abc", Options{Replica: true, Router: "analytics"}, ps.DatabaseEngineNeki)
	c.Assert(err, qt.IsNil)
	c.Assert(username, qt.Equals, "role-abc|analytics")

	_, err = postgresUsername("role-abc", Options{Router: "analytics"}, ps.DatabaseEnginePostgres)
	c.Assert(err, qt.ErrorMatches, "--router is only supported for Neki databases")
}

func TestPostgresOptions(t *testing.T) {
	c := qt.New(t)

	options, err := postgresOptions(Options{Replica: true}, ps.DatabaseEngineNeki)
	c.Assert(err, qt.IsNil)
	c.Assert(options, qt.Equals, "-c __neki.target=REPLICA")

	options, err = postgresOptions(Options{Shard: "shard-1"}, ps.DatabaseEngineNeki)
	c.Assert(err, qt.IsNil)
	c.Assert(options, qt.Equals, "-c __neki.shard=shard-1")

	options, err = postgresOptions(Options{Replica: true, Shard: "shard-1"}, ps.DatabaseEngineNeki)
	c.Assert(err, qt.IsNil)
	c.Assert(options, qt.Equals, "-c __neki.target=REPLICA -c __neki.shard=shard-1")

	options, err = postgresOptions(Options{Replica: true, Shard: "shard-1"}, ps.DatabaseEnginePostgres)
	c.Assert(err, qt.ErrorMatches, "--shard is only supported for Neki databases")
	c.Assert(options, qt.Equals, "")

	options, err = postgresOptions(Options{Replica: true}, ps.DatabaseEnginePostgres)
	c.Assert(err, qt.IsNil)
	c.Assert(options, qt.Equals, "")
}

func TestPostgresConnStrIncludesOptions(t *testing.T) {
	c := qt.New(t)
	c.Assert(postgresConnStr("db.example", "5432", "role-abc|default", "secret", "postgres", "-c __neki.shard=shard-1"),
		qt.Equals, "host=db.example port=5432 user=role-abc|default password=secret dbname=postgres sslmode=verify-full options='-c __neki.shard=shard-1'")
}

func TestNewSessionRejectsNekiFlagsBeforeMintingRole(t *testing.T) {
	createCalled := false
	ch := &cmdutil.Helper{
		Config: &config.Config{Organization: "org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases: &mock.DatabaseService{
					GetFn: func(context.Context, *ps.GetDatabaseRequest) (*ps.Database, error) {
						return &ps.Database{Name: "app", Kind: ps.DatabaseEnginePostgres}, nil
					},
				},
				DatabaseBranches: &mock.DatabaseBranchesService{
					GetFn: func(context.Context, *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
						return &ps.DatabaseBranch{Name: "main", Ready: true}, nil
					},
				},
				PostgresRoles: &mock.PostgresRolesService{
					CreateFn: func(context.Context, *ps.CreatePostgresRoleRequest) (*ps.PostgresRole, error) {
						createCalled = true
						return nil, errors.New("should not create a role")
					},
				},
			}, nil
		},
	}

	_, err := NewSession(context.Background(), ch, Options{
		Organization: "org",
		Database:     "app",
		Branch:       "main",
		Shard:        "shard-1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "--shard/--router are only supported for Neki databases" {
		t.Fatalf("error = %q", err.Error())
	}
	if createCalled {
		t.Fatal("created a role before rejecting --shard")
	}
}
