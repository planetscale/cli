package sqlquery

import (
	"errors"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
)

func TestIsDestructiveQuery(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{query: "SELECT 1", want: false},
		{query: "INSERT INTO t VALUES (1)", want: false},
		{query: "UPDATE t SET x = 1", want: false},
		{query: "DELETE FROM users WHERE id = 1", want: true},
		{query: "drop table users", want: true},
		{query: "TRUNCATE TABLE users", want: true},
		{query: "ALTER TABLE users DROP COLUMN email", want: true},
		{query: "SELECT 1; DELETE FROM users", want: true},
		{query: "  delete from users", want: true},
		{query: "SELECT 1\nDELETE FROM users", want: true},
		{query: "WITH c AS (SELECT 1) DELETE FROM users", want: true},
		{query: "SELECT '; DELETE FROM users' AS sample", want: false},
		{query: "SELECT 'DROP TABLE users' AS sample", want: false},
		{query: "SELECT \"TRUNCATE\" FROM users", want: false},
		{query: "INSERT INTO logs VALUES ('created; still same statement')", want: false},
		{query: "SELECT 1 -- DELETE FROM users", want: false},
		{query: "SELECT 1 # DELETE FROM users", want: false},
		{query: "SELECT 1 /* DROP TABLE users */", want: false},
		{query: "SELECT 1 -- ; DELETE FROM users\nSELECT 2", want: false},
		{query: "SELECT 1 # ; DELETE FROM users\nSELECT 2", want: false},
		{query: "SELECT 1 /* ; TRUNCATE users */; SELECT 2", want: false},
		{query: "SELECT 1; /* comment */ DELETE FROM users", want: true},
		// MySQL executable comments are live SQL, not inert comments.
		{query: "/*! DROP TABLE users */", want: true},
		{query: "/*!50000 DROP TABLE users */", want: true},
		{query: "/*!80000 DELETE FROM users WHERE id = 1 */", want: true},
		{query: "/*!99999 TRUNCATE TABLE users */", want: true},
		{query: "/*! SELECT 1 */", want: false},
		{query: "SELECT 1 /* ! DROP TABLE users */", want: false},
		{query: "SELECT deleted_at FROM users", want: false},
		{query: "SELECT is_deleted FROM users", want: false},
		{query: "SELECT * FROM delete_queue", want: false},
		{query: "SELECT drop FROM t", want: false},
		{query: "SELECT 1 AS drop", want: false},
		{query: "SELECT truncate FROM t", want: false},
		{query: "MERGE INTO target USING source ON target.id = source.id WHEN MATCHED THEN DELETE", want: true},
		{query: "MERGE INTO target USING source ON target.id = source.id WHEN NOT MATCHED THEN INSERT VALUES (1)", want: false},
		{query: "WITH c AS MATERIALIZED (SELECT 1) DELETE FROM users", want: true},
		{query: "WITH c AS NOT MATERIALIZED (SELECT 1) DELETE FROM users", want: true},
		{query: "SELECT '\\'; DROP TABLE users; --'", want: true},
		{query: "WITH d AS (DELETE FROM users RETURNING *) SELECT count(*) FROM d", want: true},
		{query: "EXPLAIN ANALYZE DELETE FROM users", want: true},
		{query: "EXPLAIN (ANALYZE) DELETE FROM users", want: true},
		{query: "EXPLAIN DELETE FROM users", want: false},
		{query: "EXPLAIN ANALYZE SELECT 1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			if got := IsDestructiveQuery(tt.query); got != tt.want {
				t.Fatalf("IsDestructiveQuery(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestExecuteBlocksDestructiveWithoutForce(t *testing.T) {
	ch := &cmdutil.Helper{
		Config: &config.Config{Organization: "bb"},
	}

	for _, query := range []string{
		"DELETE FROM users",
		"/*! DROP TABLE users */",
		"/*!50000 DROP TABLE users */",
	} {
		t.Run(query, func(t *testing.T) {
			_, err := Execute(t.Context(), ch, Options{
				Organization: "bb",
				Database:     "db",
				Branch:       "main",
				Query:        query,
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if _, ok := errors.AsType[*DestructiveQueryError](err); !ok {
				t.Fatalf("error = %T (%v), want *DestructiveQueryError", err, err)
			}
		})
	}
}
