package d1

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// writeDump writes sql to a temp file and returns its path, for tests that need to run the
// full ParseDump -> BuildTypeCoercionContext -> convert pipeline.
func writeDump(t *testing.T, sql string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "dump.sql")
	if err := os.WriteFile(path, []byte(sql), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func convertTablesDDL(t *testing.T, sql string) string {
	t.Helper()
	parts, _, err := ConvertSchemaParts(writeDump(t, sql))
	if err != nil {
		t.Fatalf("ConvertSchemaParts: %v", err)
	}
	return parts.Tables
}

// pgTestDBName is the scratch database used to verify generated DDL against a real
// Postgres instance when one is reachable (see assertValidPostgresDDL).
func pgTestDBName() string {
	if db := os.Getenv("D1_TEST_PG_DB"); db != "" {
		return db
	}
	return "d1_fix_test"
}

// postgresTestConn probes for a locally reachable Postgres, trying a direct connection
// first and falling back to `sudo -u postgres psql` (a common local dev setup), and returns
// the command + prefix args to reach it, or ok=false if neither works.
func postgresTestConn() (cmdName string, prefixArgs []string, ok bool) {
	psqlPath, err := exec.LookPath("psql")
	if err != nil {
		return "", nil, false
	}
	db := pgTestDBName()
	if runPsqlCheck(psqlPath, nil, db) {
		return psqlPath, nil, true
	}
	if _, err := exec.LookPath("sudo"); err == nil {
		prefix := []string{"-n", "-u", "postgres", psqlPath}
		if runPsqlCheck("sudo", prefix, db) {
			return "sudo", prefix, true
		}
	}
	return "", nil, false
}

func runPsqlCheck(cmdName string, prefixArgs []string, db string) bool {
	args := append(append([]string{}, prefixArgs...), "-d", db, "-c", "SELECT 1")
	return exec.Command(cmdName, args...).Run() == nil
}

// assertValidPostgresDDL executes ddl against a real local Postgres instance inside a
// transaction that is always rolled back, to catch the psql-level syntax/semantic failures
// unit tests based on string matching alone could miss. Skips gracefully when no local
// Postgres is reachable (e.g. in CI), so it never fails a build for environment reasons.
func assertValidPostgresDDL(t *testing.T, ddl string) {
	t.Helper()
	cmdName, prefixArgs, ok := postgresTestConn()
	if !ok {
		t.Skip("no local Postgres reachable for DDL verification")
	}
	args := append(append([]string{}, prefixArgs...), "-v", "ON_ERROR_STOP=1", "-d", pgTestDBName(), "-c", "BEGIN; "+ddl+" ROLLBACK;")
	out, err := exec.Command(cmdName, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("generated DDL failed against Postgres:\n%s\n--- psql output ---\n%s", ddl, out)
	}
}

func TestLooksLikeTimestampColumnName(t *testing.T) {
	cases := map[string]bool{
		"created_at":    true,
		"updated_at":    true,
		"event_date":    true,
		"date_of_birth": true,
		"date":          true,
		"timestamp_raw": true,
		"candidate":     false,
		"mandate":       false,
		"metadata":      false,
	}
	for name, want := range cases {
		if got := looksLikeTimestampColumnName(name); got != want {
			t.Fatalf("looksLikeTimestampColumnName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestIsTimestampTextIgnoresFalsePositiveNames(t *testing.T) {
	for _, name := range []string{"candidate", "mandate"} {
		col := ColumnSchema{Name: name, Type: "TEXT"}
		if isTimestampText(col) {
			t.Fatalf("isTimestampText(%q) = true, want false", name)
		}
	}
}

func TestMapSQLiteDefaultFunctionUnixEpoch(t *testing.T) {
	cases := map[string]struct {
		def    string
		pgType string
		want   string
	}{
		"unixepoch('now') timestamptz":    {"unixepoch('now')", "TIMESTAMPTZ", "now()"},
		"UNIXEPOCH('now') timestamptz":    {"UNIXEPOCH('now')", "TIMESTAMPTZ", "now()"},
		"UnixEpoch('now') timestamptz":    {"UnixEpoch('now')", "TIMESTAMPTZ", "now()"},
		"(UNIXEPOCH('now')) timestamptz":  {"(UNIXEPOCH('now'))", "TIMESTAMPTZ", "now()"},
		"UNIXEPOCH('subsec') timestamptz": {"UNIXEPOCH('subsec')", "TIMESTAMPTZ", "clock_timestamp()"},
		"unixepoch() timestamptz":         {"unixepoch()", "TIMESTAMPTZ", "now()"},
		"unixepoch('now') bigint":         {"unixepoch('now')", "BIGINT", "extract(epoch from now())::bigint"},
		"unixepoch numeric timestamptz":   {"UNIXEPOCH(1700000000)", "TIMESTAMPTZ", "to_timestamp(1700000000)"},
		"CURRENT_TIMESTAMP":               {"CURRENT_TIMESTAMP", "TIMESTAMPTZ", "CURRENT_TIMESTAMP"},
		"datetime('now')":                 {"datetime('now')", "TIMESTAMPTZ", "now()"},
		"date('now') timestamptz":         {"date('now')", "TIMESTAMPTZ", "CURRENT_DATE"},
		"(date('now')) timestamptz":       {"(date('now'))", "TIMESTAMPTZ", "CURRENT_DATE"},
		"date('now') text":                {"date('now')", "TEXT", "CURRENT_DATE"},
	}
	for name, tc := range cases {
		got := mapSQLiteDefaultFunction(tc.def, tc.pgType)
		if got != tc.want {
			t.Fatalf("%s: mapSQLiteDefaultFunction(%q, %q) = %q, want %q", name, tc.def, tc.pgType, got, tc.want)
		}
	}
}

func TestSplitTopLevelArgsHandlesDoubledSingleQuoteEscapes(t *testing.T) {
	cases := map[string]struct {
		in   string
		want []string
	}{
		"comma inside a doubled-quote-escaped literal": {
			`'%Y-%m-%d', 'it''s, now'`,
			[]string{`'%Y-%m-%d'`, ` 'it''s, now'`},
		},
		"comma immediately after a doubled-quote escape": {
			`'%Y-%m-%d', 'a'',b'`,
			[]string{`'%Y-%m-%d'`, ` 'a'',b'`},
		},
		"multiple escaped quotes around commas in one literal": {
			`'%Y-%m-%d', 'foo'',''bar'`,
			[]string{`'%Y-%m-%d'`, ` 'foo'',''bar'`},
		},
		"plain top-level split unaffected": {
			`'now', '+1 day'`,
			[]string{`'now'`, ` '+1 day'`},
		},
	}
	for name, tc := range cases {
		got := splitTopLevelArgs(tc.in)
		if len(got) != len(tc.want) {
			t.Fatalf("%s: splitTopLevelArgs(%q) = %#v, want %#v", name, tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%s: splitTopLevelArgs(%q) = %#v, want %#v", name, tc.in, got, tc.want)
			}
		}
	}
}

func TestMapSQLiteDefaultFunctionStrftimeNow(t *testing.T) {
	utcDay := "date_trunc('day', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'"
	cases := map[string]struct {
		def    string
		pgType string
		want   string
	}{
		"strftime iso8601 'now' on TIMESTAMPTZ":       {"strftime('%Y-%m-%dT%H:%M:%SZ', 'now')", "TIMESTAMPTZ", "date_trunc('second', now())"},
		"parenthesized strftime 'now' on TIMESTAMPTZ": {"(strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))", "TIMESTAMPTZ", "date_trunc('second', now())"},
		"date-only strftime becomes UTC midnight":     {"STRFTIME('%Y-%m-%d', 'NOW')", "TIMESTAMPTZ", utcDay},
		"unknown format is not guessed":               {"strftime('%Q', 'now')", "TIMESTAMPTZ", ""},
		"strftime 'now' left unmapped on TEXT":        {"strftime('%Y-%m-%dT%H:%M:%SZ', 'now')", "TEXT", ""},
		"strftime non-'now' value left unmapped":      {"strftime('%Y-%m-%d', modified_at)", "TIMESTAMPTZ", ""},
	}
	for name, tc := range cases {
		got := mapSQLiteDefaultFunction(tc.def, tc.pgType)
		if got != tc.want {
			t.Fatalf("%s: mapSQLiteDefaultFunction(%q, %q) = %q, want %q", name, tc.def, tc.pgType, got, tc.want)
		}
	}
}

func TestMapSQLiteDefaultFunctionStrftimeModifiers(t *testing.T) {
	currentTime := "(TIMESTAMPTZ '2000-01-01 00:00:00+00' + ((now() AT TIME ZONE 'UTC') - date_trunc('day', now() AT TIME ZONE 'UTC')))"
	utcDay := "date_trunc('day', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'"
	cases := map[string]struct {
		def    string
		pgType string
		want   string
	}{
		"'+1 day' modifier": {
			"strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+1 day')", "TIMESTAMPTZ",
			"date_trunc('second', (now() + make_interval(secs => (1)::double precision * 86400)))",
		},
		"'-30 minutes' modifier": {
			"strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-30 minutes')", "TIMESTAMPTZ",
			"date_trunc('second', (now() - make_interval(secs => (30)::double precision * 60)))",
		},
		"'localtime' modifier is a no-op": {
			"strftime('%Y-%m-%dT%H:%M:%SZ', 'now', 'localtime')", "TIMESTAMPTZ",
			"date_trunc('second', now())",
		},
		"'utc' modifier is a no-op": {
			"strftime('%Y-%m-%dT%H:%M:%SZ', 'now', 'utc')", "TIMESTAMPTZ",
			"date_trunc('second', now())",
		},
		"'UTC' modifier is case-insensitive": {
			"strftime('%Y-%m-%dT%H:%M:%SZ', 'now', 'UTC')", "TIMESTAMPTZ",
			"date_trunc('second', now())",
		},
		"'utc' combined with a real interval modifier still applies the interval": {
			"strftime('%Y-%m-%dT%H:%M:%SZ', 'now', 'utc', '+1 day')", "TIMESTAMPTZ",
			"date_trunc('second', (now() + make_interval(secs => (1)::double precision * 86400)))",
		},
		"'start of day' modifier": {
			"strftime('%Y-%m-%dT%H:%M:%SZ', 'now', 'start of day')", "TIMESTAMPTZ",
			"date_trunc('second', " + utcDay + ")",
		},
		"calendar month arithmetic is not guessed": {
			"strftime('%Y-%m-%d', 'now', 'start of month', '+1 month')", "TIMESTAMPTZ",
			"",
		},
		"unrecognized 'weekday N' modifier is not guessed": {
			"strftime('%Y-%m-%d', 'now', 'weekday 1')", "TIMESTAMPTZ",
			"",
		},
		"bare CURRENT_TIMESTAMP time-value": {
			"strftime('%Y-%m-%dT%H:%M:%SZ', CURRENT_TIMESTAMP)", "TIMESTAMPTZ",
			"date_trunc('second', now())",
		},
		"bare CURRENT_TIME uses SQLite's 2000-01-01 date": {
			"strftime('%Y-%m-%dT%H:%M:%SZ', CURRENT_TIME)", "TIMESTAMPTZ",
			"date_trunc('second', " + currentTime + ")",
		},
		"bare CURRENT_DATE time-value": {
			"strftime('%Y-%m-%dT%H:%M:%SZ', CURRENT_DATE)", "TIMESTAMPTZ",
			"date_trunc('second', " + utcDay + ")",
		},
		"CURRENT_DATE with '+1 day' modifier": {
			"strftime('%Y-%m-%dT%H:%M:%SZ', CURRENT_DATE, '+1 day')", "TIMESTAMPTZ",
			"date_trunc('second', (" + utcDay + " + make_interval(secs => (1)::double precision * 86400)))",
		},
		"quoted literal date (not 'now') is not a current-time synonym": {
			"strftime('%Y-%m-%d', '2024-01-01')", "TIMESTAMPTZ",
			"",
		},
		"%s format on BIGINT maps to epoch seconds": {
			"strftime('%s', 'now')", "BIGINT",
			"floor(extract(epoch from now()))::bigint",
		},
		"%s format on INTEGER maps to epoch seconds": {
			"strftime('%s', 'now')", "INTEGER",
			"floor(extract(epoch from now()))::integer",
		},
		"%s format on DOUBLE PRECISION maps to epoch seconds": {
			"strftime('%s', 'now')", "DOUBLE PRECISION",
			"floor(extract(epoch from now()))::double precision",
		},
		"%s format applies safe modifiers": {
			"strftime('%s', 'now', '+1 day')", "BIGINT",
			"floor(extract(epoch from (now() + make_interval(secs => (1)::double precision * 86400))))::bigint",
		},
		"non-%s format on BIGINT left unmapped": {
			"strftime('%Y-%m-%d', 'now')", "BIGINT",
			"",
		},
		"%s format on TEXT left unmapped": {
			"strftime('%s', 'now')", "TEXT",
			"",
		},
	}
	for name, tc := range cases {
		got := mapSQLiteDefaultFunction(tc.def, tc.pgType)
		if got != tc.want {
			t.Fatalf("%s: mapSQLiteDefaultFunction(%q, %q) = %q, want %q", name, tc.def, tc.pgType, got, tc.want)
		}
	}
}

func TestConvertDefaultStrftimeNowOnInferredTimestamp(t *testing.T) {
	sql := `CREATE TABLE events (
  id INTEGER PRIMARY KEY,
  created_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
INSERT INTO events (id, created_at) VALUES (1, '2024-01-01T00:00:00Z');
`
	ddl := convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `"created_at" TIMESTAMPTZ`) {
		t.Fatalf("expected created_at to be inferred as TIMESTAMPTZ:\n%s", ddl)
	}
	if strings.Contains(ddl, `'(strftime(`) {
		t.Fatalf("strftime('now') default was not mapped, still contains broken literal:\n%s", ddl)
	}
	if !strings.Contains(ddl, `DEFAULT date_trunc('second', now())`) {
		t.Fatalf("expected strftime('now') default to preserve second precision:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertDefaultStrftimeModifierOnInferredTimestamp(t *testing.T) {
	sql := `CREATE TABLE events (
  id INTEGER PRIMARY KEY,
  expires_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+1 day')),
  starts_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', CURRENT_TIMESTAMP))
);
INSERT INTO events (id, expires_at, starts_at) VALUES (1, '2024-01-02T00:00:00Z', '2024-01-01T00:00:00Z');
`
	ddl := convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `"expires_at" TIMESTAMPTZ`) || !strings.Contains(ddl, `"starts_at" TIMESTAMPTZ`) {
		t.Fatalf("expected both columns to be inferred as TIMESTAMPTZ:\n%s", ddl)
	}
	if !strings.Contains(ddl, "DEFAULT date_trunc('second', (now() + make_interval(secs => (1)::double precision * 86400)))") {
		t.Fatalf("expected +1 day modifier mapped to an interval addition:\n%s", ddl)
	}
	if !strings.Contains(ddl, "DEFAULT date_trunc('second', now())") {
		t.Fatalf("expected bare CURRENT_TIMESTAMP time-value mapped to now():\n%s", ddl)
	}
	if strings.Contains(ddl, "strftime(") {
		t.Fatalf("strftime() must not leak into Postgres DDL:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertDefaultStrftimeUtcLocaltimeModifierIsNoOp(t *testing.T) {
	sql := `CREATE TABLE events (
  id INTEGER PRIMARY KEY,
  starts_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now', 'utc')),
  ends_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now', 'localtime'))
);
INSERT INTO events (id, starts_at, ends_at) VALUES (1, '2024-01-01T00:00:00Z', '2024-01-02T00:00:00Z');
`
	ddl := convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `"starts_at" TIMESTAMPTZ`) || !strings.Contains(ddl, `"ends_at" TIMESTAMPTZ`) {
		t.Fatalf("expected both columns to be inferred as TIMESTAMPTZ:\n%s", ddl)
	}
	if strings.Contains(ddl, "strftime(") {
		t.Fatalf("strftime() must not leak into Postgres DDL:\n%s", ddl)
	}
	for _, col := range []string{"starts_at", "ends_at"} {
		column := strings.Split(strings.Split(ddl, `"`+col+`" TIMESTAMPTZ`)[1], "\n")[0]
		if !strings.Contains(column, "DEFAULT date_trunc('second', now())") {
			t.Fatalf("expected 'utc'/'localtime' modifier to be a no-op, not drop the DEFAULT clause on %q:\n%s", col, ddl)
		}
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertDefaultStrftimeUnmappableModifierFallsBack(t *testing.T) {
	sql := `CREATE TABLE events (
  id INTEGER PRIMARY KEY,
  next_monday TEXT DEFAULT (strftime('%Y-%m-%d', 'now', 'weekday 1'))
);
INSERT INTO events (id, next_monday) VALUES (1, '2024-01-01'), (2, '2024-01-08');
`
	ddl := convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `'(strftime(''%Y-%m-%d'', ''now'', ''weekday 1''))'`) {
		t.Fatalf("expected unmappable modifier to fall back to the escaped-literal default:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertDefaultStrftimeUnmappableModifierOnTimestampOmitsDefault(t *testing.T) {
	sql := `CREATE TABLE events (
  id INTEGER PRIMARY KEY,
  next_monday_at TEXT DEFAULT (strftime('%Y-%m-%d', 'now', 'weekday 1'))
);
INSERT INTO events (id, next_monday_at) VALUES (1, '2024-01-01');
`
	ddl := convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `"next_monday_at" TIMESTAMPTZ`) {
		t.Fatalf("expected next_monday_at to be inferred as TIMESTAMPTZ:\n%s", ddl)
	}
	if strings.Contains(ddl, "strftime(") {
		t.Fatalf("strftime() must not leak into Postgres DDL:\n%s", ddl)
	}
	column := strings.Split(strings.Split(ddl, `"next_monday_at" TIMESTAMPTZ`)[1], "\n")[0]
	if strings.Contains(column, "DEFAULT") {
		t.Fatalf("unsupported strftime modifier must omit the default rather than change its value:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertDefaultStrftimeUnsupportedFormatOmitsDefault(t *testing.T) {
	sql := `CREATE TABLE events (
  id INTEGER PRIMARY KEY,
  created_at TEXT DEFAULT (strftime('%Q', 'now'))
);
INSERT INTO events (id, created_at) VALUES (1, '2024-01-01T00:00:00Z');
`
	ddl := convertTablesDDL(t, sql)
	column := strings.Split(strings.Split(ddl, `"created_at" TIMESTAMPTZ`)[1], "\n")[0]
	if strings.Contains(column, "DEFAULT") {
		t.Fatalf("unsupported strftime format must omit the default:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

// TestConvertDefaultDateExpressionNotQuotedAsString guards against regressing
// to treating a SQLite expression default such as (date('now')) as a plain
// string literal. Quoting the whole expression produces invalid Postgres DDL
// (a syntax/type error at import time) instead of a valid expression default.
func TestConvertDefaultDateExpressionNotQuotedAsString(t *testing.T) {
	cases := map[string]struct {
		def    string
		pgType string
		want   string
	}{
		"parenthesized date('now') on TIMESTAMPTZ": {"(date('now'))", "TIMESTAMPTZ", "CURRENT_DATE"},
		"bare date('now') on TIMESTAMPTZ":          {"date('now')", "TIMESTAMPTZ", "CURRENT_DATE"},
		"date('now') on TEXT":                      {"date('now')", "TEXT", "CURRENT_DATE"},
	}
	for name, tc := range cases {
		got, ok := convertDefault(tc.def, tc.pgType)
		if !ok || got != tc.want {
			t.Fatalf("%s: convertDefault(%q, %q) = %q, want %q", name, tc.def, tc.pgType, got, tc.want)
		}
		if len(got) > 0 && got[0] == '\'' {
			t.Fatalf("%s: convertDefault(%q, %q) = %q, must not be a quoted string literal", name, tc.def, tc.pgType, got)
		}
	}
}

// TestConvertDefaultStringLiteralsStillQuoted ensures genuine SQLite string
// defaults are still emitted as Postgres string literals; the date()
// expression fix must not change this behavior.
func TestConvertDefaultStringLiteralsStillQuoted(t *testing.T) {
	cases := map[string]struct {
		def    string
		pgType string
		want   string
	}{
		"already-quoted string default":             {"'foo'", "TEXT", "'foo'"},
		"bare text default gets quoted":             {"foo", "TEXT", "'foo'"},
		"quoted strftime text is not a call":        {"'strftime(not a call)'", "TIMESTAMPTZ", "'strftime(not a call)'"},
		"double-quoted strftime text is not a call": {`"strftime(not a call)"`, "TIMESTAMPTZ", "'strftime(not a call)'"},
	}
	for name, tc := range cases {
		got, ok := convertDefault(tc.def, tc.pgType)
		if !ok || got != tc.want {
			t.Fatalf("%s: convertDefault(%q, %q) = %q, want %q", name, tc.def, tc.pgType, got, tc.want)
		}
	}
}

// Bug 1: DEFAULT (time('now')) was not recognized and corrupted into invalid SQL.
func TestConvertDefaultTimeNow(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE events (id INTEGER PRIMARY KEY, event_time TEXT DEFAULT (time('now')));`)
	if strings.Contains(ddl, `'(time(`) {
		t.Fatalf("time('now') default was not mapped, still contains broken literal:\n%s", ddl)
	}
	if !strings.Contains(ddl, "to_char(now(), 'HH24:MI:SS')") {
		t.Fatalf("expected mapped time('now') default:\n%s", ddl)
	}
}

func TestConvertDefaultEscapesEmbeddedQuotes(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE logs (id INTEGER PRIMARY KEY, ts TEXT DEFAULT (strftime('%Y-%m-%d','now')));`)
	if !strings.Contains(ddl, `'(strftime(''%Y-%m-%d'',''now''))'`) {
		t.Fatalf("expected properly escaped literal default:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertDefaultRandomInteger(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE tokens (id INTEGER PRIMARY KEY, nonce INTEGER DEFAULT (random()));`)
	if !strings.Contains(ddl, "floor(random() * 9223372036854775807)") {
		t.Fatalf("expected genuine random bigint default:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertDefaultJuliandayNow(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE t (id INTEGER PRIMARY KEY, jd REAL DEFAULT (julianday('now')));`)
	if strings.Contains(strings.ToUpper(ddl), "JULIANDAY(") {
		t.Fatalf("julianday() must not appear in Postgres DDL:\n%s", ddl)
	}
	if !strings.Contains(ddl, "2440587.5") {
		t.Fatalf("expected julian day epoch conversion:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertDefaultRandomblob(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE t (id INTEGER PRIMARY KEY, blob_val BLOB DEFAULT (randomblob(16)));`)
	if strings.Contains(strings.ToUpper(ddl), "RANDOMBLOB(") {
		t.Fatalf("randomblob() must not appear in Postgres DDL:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

// TestConvertDefaultRandomblobOnNonByteaColumn guards against emitting a bytea-producing
// DEFAULT expression for a bare `randomblob(N)` default on a column whose Postgres type
// isn't BYTEA (e.g. TEXT). Postgres has no implicit cast from bytea to text, so the raw
// decode(...) expression from randomBytesExpr fails CREATE TABLE outright; non-BYTEA
// columns must fall back to a hex-encoded text representation instead.
func TestConvertDefaultRandomblobOnNonByteaColumn(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE t (id INTEGER PRIMARY KEY, token TEXT DEFAULT (randomblob(8)));`)
	if strings.Contains(strings.ToUpper(ddl), "RANDOMBLOB(") {
		t.Fatalf("randomblob() must not appear in Postgres DDL:\n%s", ddl)
	}
	if strings.Contains(ddl, "DEFAULT decode(") {
		t.Fatalf("TEXT column must not receive a bytea-typed DEFAULT expression:\n%s", ddl)
	}
	if !strings.Contains(ddl, "encode(") {
		t.Fatalf("expected a hex-encoded text fallback for randomblob() default on a non-BYTEA column:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertDefaultUUIDGeneratorExpr(t *testing.T) {
	sql := `CREATE TABLE items (
  id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(6))))
);
INSERT INTO items (id) VALUES ('11111111-1111-4111-8111-111111111111');
`
	ddl := convertTablesDDL(t, sql)
	if !strings.Contains(ddl, "UUID") {
		t.Fatalf("expected id to be inferred as UUID:\n%s", ddl)
	}
	if !strings.Contains(ddl, "gen_random_uuid()") {
		t.Fatalf("expected gen_random_uuid() default:\n%s", ddl)
	}
	if strings.Contains(strings.ToUpper(ddl), "RANDOMBLOB") {
		t.Fatalf("randomblob() must not leak into Postgres DDL:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

// TestConvertDefaultUUIDGeneratorExprWithoutSamples guards against the fake-UUID
// randomblob()/hex() idiom being mapped to a short, wrong-length hex string when the
// column's Postgres type wasn't inferred as UUID (e.g. a schema-only dump with no sampled
// rows to match against the UUID shape). The DEFAULT expression's shape alone must be
// enough to recognize the idiom and map it to gen_random_uuid(), independent of pgType.
func TestConvertDefaultUUIDGeneratorExprWithoutSamples(t *testing.T) {
	sql := `CREATE TABLE items (
  id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(6))))
);
`
	ddl := convertTablesDDL(t, sql)
	if !strings.Contains(ddl, "gen_random_uuid()") {
		t.Fatalf("expected gen_random_uuid() default even without sampled rows to infer UUID type:\n%s", ddl)
	}
	if strings.Contains(strings.ToUpper(ddl), "RANDOMBLOB") {
		t.Fatalf("randomblob() must not leak into Postgres DDL:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

// An FK column must keep the type of the primary key it references, even when its sampled
// values look boolean-like (only 0/1), or the two columns end up with mismatched types.
func TestBooleanHeuristicIgnoresForeignKeys(t *testing.T) {
	sql := `CREATE TABLE roles (id INTEGER PRIMARY KEY);
CREATE TABLE users (id INTEGER PRIMARY KEY, role_id INTEGER REFERENCES roles(id));
INSERT INTO roles (id) VALUES (0), (1), (2);
INSERT INTO users (id, role_id) VALUES (1, 0), (2, 1);
`
	ddl := convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `"role_id" BIGINT`) {
		t.Fatalf("expected FK column to stay BIGINT despite 0/1-only sampled values:\n%s", ddl)
	}
	if !strings.Contains(ddl, `FOREIGN KEY ("role_id") REFERENCES "roles" ("id")`) {
		t.Fatalf("expected deferred FK for role_id:\n%s", ddl)
	}
	if strings.Contains(ddl, `"role_id" BOOLEAN`) {
		t.Fatalf("FK column must not be coerced to BOOLEAN:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

// Autoincrement identity columns stay INTEGER even when sampled values are only 0/1.
// Verify must not run boolean coercion checks against them or Postgres rejects
// "integer = boolean" comparisons.
func TestBooleanHeuristicIgnoresAutoIncrement(t *testing.T) {
	sql := `CREATE TABLE counters (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  value INTEGER NOT NULL
);
INSERT INTO counters (id, value) VALUES (0, 10), (1, 20);
`
	path := writeDump(t, sql)
	tables, err := ParseDump(path)
	if err != nil {
		t.Fatalf("ParseDump: %v", err)
	}
	coerceCtx, err := BuildTypeCoercionContext(path, tables)
	if err != nil {
		t.Fatalf("BuildTypeCoercionContext: %v", err)
	}

	var idCol ColumnSchema
	for _, col := range tables[0].Columns {
		if col.Name == "id" {
			idCol = col
			break
		}
	}
	if idCol.Name == "" {
		t.Fatal("missing id column")
	}
	if isBooleanLikeColumn(idCol, tables[0], coerceCtx) {
		t.Fatal("autoincrement id must not be treated as boolean-like")
	}
	if got := sqliteTypeToPostgres(idCol, tables[0], tables, coerceCtx); got != "INTEGER" {
		t.Fatalf("expected autoincrement id to map to INTEGER, got %s", got)
	}

	ddl := convertTablesDDL(t, sql)
	if strings.Contains(ddl, `"id" BOOLEAN`) {
		t.Fatalf("autoincrement id must not be coerced to BOOLEAN:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertDefaultBooleanNonZeroOneLiteral(t *testing.T) {
	sql := `CREATE TABLE flags (
  id INTEGER PRIMARY KEY,
  rollout_state INTEGER DEFAULT 2
);
INSERT INTO flags (id, rollout_state) VALUES (1, 0);
INSERT INTO flags (id, rollout_state) VALUES (2, 1);
`
	ddl := convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `"rollout_state" BOOLEAN DEFAULT TRUE`) {
		t.Fatalf("expected boolean column with TRUE default for non-zero literal:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertDefaultDoubleQuotedLiteral(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE t (id INTEGER PRIMARY KEY, status TEXT DEFAULT "active");`)
	if !strings.Contains(ddl, `DEFAULT 'active'`) {
		t.Fatalf("expected double-quoted default converted to a string literal:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertDefaultCastUnixepoch(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE t (id INTEGER PRIMARY KEY, cst TEXT DEFAULT (CAST(unixepoch() AS TEXT)));`)
	if strings.Contains(ddl, "'CAST(") || strings.Contains(ddl, "CAST(unixepoch") {
		t.Fatalf("CAST(unixepoch() AS TEXT) default must not become a literal string:\n%s", ddl)
	}
	if !strings.Contains(ddl, "extract(epoch from now())::bigint)::text") {
		t.Fatalf("expected epoch-as-text default:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertNumericDecimalTypes(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE t (id INTEGER PRIMARY KEY, amount DECIMAL(10,2) NOT NULL, tax NUMERIC NOT NULL);`)
	if !strings.Contains(ddl, `"amount" NUMERIC(10,2) NOT NULL`) {
		t.Fatalf("expected DECIMAL(10,2) to map to NUMERIC(10,2):\n%s", ddl)
	}
	if !strings.Contains(ddl, `"tax" NUMERIC NOT NULL`) {
		t.Fatalf("expected bare NUMERIC to map to NUMERIC:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertGeneratedStoredColumn(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE line_items (id INTEGER PRIMARY KEY, price REAL NOT NULL, qty REAL NOT NULL, total REAL GENERATED ALWAYS AS (price * qty) STORED);`)
	if !strings.Contains(ddl, `GENERATED ALWAYS AS ("price" * "qty") STORED`) {
		t.Fatalf("expected generated column expression preserved:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertGeneratedVirtualColumnFallsBackToStored(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE line_items (id INTEGER PRIMARY KEY, price REAL NOT NULL, qty REAL NOT NULL, total REAL GENERATED ALWAYS AS (price * qty) VIRTUAL);`)
	if !strings.Contains(ddl, `GENERATED ALWAYS AS ("price" * "qty") STORED`) {
		t.Fatalf("expected VIRTUAL generated column to fall back to STORED (only mode Postgres 16 supports):\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertGeneratedColumnHasNoDefault(t *testing.T) {
	col := ColumnSchema{Name: "total", Type: "REAL", GeneratedExpr: "price * qty", GeneratedMode: "STORED", DefaultValue: "0"}
	table := TableSchema{Name: "line_items", Columns: []ColumnSchema{
		{Name: "price", Type: "REAL"},
		{Name: "qty", Type: "REAL"},
		col,
	}}
	got := convertColumn(col, table, []TableSchema{table}, nil)
	if strings.Contains(got, "DEFAULT") {
		t.Fatalf("generated column must not also emit DEFAULT: %q", got)
	}
}

// Unrecognized DEFAULT expressions on typed columns must never be emitted verbatim:
// an unbalanced ")" plus ";" escapes CREATE TABLE and injects arbitrary SQL into psql.
func TestConvertDefaultRejectsSQLInjectionOnIntegerColumn(t *testing.T) {
	cases := []string{
		`(0)); DROP TABLE users; --`,
		`0); DROP TABLE users; --`,
		`1; DROP TABLE users; --`,
		`(1) + (SELECT 1)`,
	}
	for _, def := range cases {
		got, ok := convertDefault(def, "BIGINT")
		if ok || got != "" {
			t.Fatalf("convertDefault(%q, BIGINT) = %q, ok=%v; must omit unrecognized/hostile defaults", def, got, ok)
		}
	}

	ddl := convertTablesDDL(t, `CREATE TABLE evil (n INTEGER DEFAULT (0));`)
	if strings.Contains(strings.ToUpper(ddl), "DROP TABLE") {
		t.Fatalf("injected DROP must not appear in converted DDL:\n%s", ddl)
	}

	// Direct regression for the reported payload shape on a BIGINT column.
	col := ColumnSchema{Name: "n", Type: "INTEGER", DefaultValue: `(0)); DROP TABLE users; --`}
	table := TableSchema{Name: "t", Columns: []ColumnSchema{col}}
	got := convertColumn(col, table, []TableSchema{table}, nil)
	if strings.Contains(strings.ToUpper(got), "DROP TABLE") || strings.Contains(got, ");") {
		t.Fatalf("hostile default must not appear in column DDL: %q", got)
	}
	if strings.Contains(got, "DEFAULT") {
		t.Fatalf("hostile default must be omitted entirely: %q", got)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertDefaultKeepsSafeNumericLiterals(t *testing.T) {
	cases := map[string]struct {
		def    string
		pgType string
		want   string
	}{
		"bare zero":           {"0", "BIGINT", "0"},
		"parenthesized zero":  {"(0)", "BIGINT", "0"},
		"nested parens":       {"((-1))", "BIGINT", "-1"},
		"float":               {"1.5", "DOUBLE PRECISION", "1.5"},
		"parenthesized float": {"(3.14)", "NUMERIC", "3.14"},
	}
	for name, tc := range cases {
		got, ok := convertDefault(tc.def, tc.pgType)
		if !ok || got != tc.want {
			t.Fatalf("%s: convertDefault(%q, %q) = %q, ok=%v; want %q", name, tc.def, tc.pgType, got, ok, tc.want)
		}
	}
}

func TestConvertDefaultRejectsBrokenQuotedLiteral(t *testing.T) {
	cases := []string{
		`'foo'); DROP TABLE users; --`,
		`'unclosed`,
		`"active"); DROP TABLE users; --`,
	}
	for _, def := range cases {
		got, ok := convertDefault(def, "TEXT")
		if ok || got != "" {
			t.Fatalf("convertDefault(%q, TEXT) = %q, ok=%v; must reject broken quotes", def, got, ok)
		}
		got, ok = convertDefault(def, "BIGINT")
		if ok || got != "" {
			t.Fatalf("convertDefault(%q, BIGINT) = %q, ok=%v; must reject broken quotes", def, got, ok)
		}
	}
}

func TestConvertDefaultUUIDRejectsInjection(t *testing.T) {
	got, ok := convertDefault(`x'); DROP TABLE users; --`, "UUID")
	if ok || got != "" {
		t.Fatalf("UUID default must not quote hostile bare input: got %q ok=%v", got, ok)
	}
	got, ok = convertDefault(`550e8400-e29b-41d4-a716-446655440000`, "UUID")
	if !ok || got != `'550e8400-e29b-41d4-a716-446655440000'` {
		t.Fatalf("safe bare UUID = %q ok=%v", got, ok)
	}
}
