package d1

import (
	"strings"
	"testing"
)

func TestMapSQLiteCheckFunction(t *testing.T) {
	cases := []struct {
		name, args, want string
		ok               bool
	}{
		{"json_valid", "row_json", "(row_json) IS JSON", true},
		{"JSON_VALID", `"row_json"`, `("row_json") IS JSON`, true},
		{"json_valid", `"row_json", 1`, `("row_json") IS JSON`, true},
		{"json_valid", "", "", false},
		{"ifnull", "a, 0", "coalesce(a, 0)", true},
		{"iif", "a > 0, a, 0", "CASE WHEN (a > 0) THEN (a) ELSE (0) END", true},
		{"iif", "a > 0, a", "CASE WHEN (a > 0) THEN (a) ELSE NULL END", true},
		{"instr", "name, 'x'", "strpos(name, 'x')", true},
		{"instr", "name, 'x', 2", "", false},
		{"likely", "a > 0", "(a > 0)", true},
		{"unlikely", "a > 0", "(a > 0)", true},
		{"likelihood", "a > 0, 0.9", "(a > 0)", true},
		{"hex", "id", "upper(encode(convert_to((id)::text, 'UTF8'), 'hex'))", true},
		{"unhex", "'00ff'", "decode(('00ff'), 'hex')", true},
		{"quote", "name", "quote_literal(name)", true},
		{"unicode", "name", "ascii(name)", true},
		{"char", "65, 66", "chr(65) || chr(66)", true},
		{"json", "payload", "((payload)::json)", true},
		{"jsonb", "payload", "((payload)::jsonb)", true},
		{"json_extract", "data, '$.title'", "((data)::jsonb #>> '{title}')", true},
		{"json_extract", `"data", '$.title'`, `(("data")::jsonb #>> '{title}')`, true},
		{"json_extract", "data, path", "", false},
		{"json_pretty", "payload", "jsonb_pretty((payload)::jsonb)", true},
		{"json_quote", "name", "to_jsonb(name)", true},
		{"json_array", "1, 2", "json_build_array(1, 2)", true},
		{"json_object", "'k', 1", "json_build_object('k', 1)", true},
		{"json_array_length", "payload", "json_array_length((payload)::json)", true},
		{"datetime", "'now'", "now()", true},
		{"unixepoch", "'now'", "now()", true},
		{"printf", "'%s', name", "", false},
		{"json_set", "data, '$.x', 1", "", false},
		{"length", "name", "", false},
	}
	for _, tc := range cases {
		got, ok := mapSQLiteCheckFunction(tc.name, tc.args)
		if ok != tc.ok || got != tc.want {
			t.Errorf("mapSQLiteCheckFunction(%q, %q) = %q, %v; want %q, %v",
				tc.name, tc.args, got, ok, tc.want, tc.ok)
		}
	}
}

func TestParseSQLiteJSONPath(t *testing.T) {
	cases := map[string][]string{
		"$.foo":        {"foo"},
		"$.foo.bar":    {"foo", "bar"},
		"$[0]":         {"0"},
		"$.foo[0].bar": {"foo", "0", "bar"},
		`$."a.b"`:      {"a.b"},
		"$":            nil,
	}
	for path, want := range cases {
		got, ok := parseSQLiteJSONPath(path)
		if path == "$" {
			if ok {
				t.Fatalf("parseSQLiteJSONPath($) should fail (handled as whole document)")
			}
			continue
		}
		if !ok {
			t.Fatalf("parseSQLiteJSONPath(%q) failed", path)
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("parseSQLiteJSONPath(%q) = %#v, want %#v", path, got, want)
		}
	}
	if _, ok := parseSQLiteJSONPath("$[#]"); ok {
		t.Fatal("last-element path should be rejected")
	}
}

func TestGlobToPOSIXRegex(t *testing.T) {
	cases := map[string]string{
		"*.md":    `^.*\.md$`,
		"file?":   `^file.$`,
		"[abc]":   `^[abc]$`,
		"[!0-9]*": `^[^0-9].*$`,
		"a+b":     `^a\+b$`,
		"plain":   `^plain$`,
	}
	for in, want := range cases {
		if got := globToPOSIXRegex(in); got != want {
			t.Errorf("globToPOSIXRegex(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestConvertCheckExprSQLiteFunctions(t *testing.T) {
	table := TableSchema{Name: "t", Columns: []ColumnSchema{
		{Name: "row_json", Type: "TEXT"},
		{Name: "name", Type: "TEXT"},
		{Name: "a", Type: "INTEGER"},
		{Name: "data", Type: "TEXT"},
	}}
	cases := map[string]string{
		"json_valid(row_json)":              `("row_json") IS JSON`,
		`json_valid("row_json")`:            `("row_json") IS JSON`,
		"JSON_VALID(row_json)":              `("row_json") IS JSON`,
		"ifnull(a, 0) > 0":                  `coalesce("a", 0) > 0`,
		"iif(a > 0, a, 0)":                  `CASE WHEN ("a" > 0) THEN ("a") ELSE (0) END`,
		"instr(name, 'x') > 0":              `strpos("name", 'x') > 0`,
		"likely(a > 0)":                     `("a" > 0)`,
		`json_extract(data, '$.title')`:     `(("data")::jsonb #>> '{title}')`,
		`json_extract(data, '$.meta.tags')`: `(("data")::jsonb #>> '{meta,tags}')`,
		"a == 1":                            `"a" = 1`,
		`name GLOB '*.md'`:                  `"name" ~ '^.*\.md$'`,
		`name NOT GLOB '*.md'`:              `"name" !~ '^.*\.md$'`,
		`name REGEXP '^[a-z]+$'`:            `"name" ~ '^[a-z]+$'`,
		"length(name) > 0":                  `length("name") > 0`,
		"typeof(a) = 'integer'":             sqliteTypeofExpr(`"a"`) + ` = 'integer'`,
	}
	for expr, want := range cases {
		if got := convertCheckExpr(expr, table, nil); got != want {
			t.Errorf("convertCheckExpr(%q) = %q, want %q", expr, got, want)
		}
	}
}

func TestConvertCheckConstraintJSONValid(t *testing.T) {
	sql := `CREATE TABLE deferred_source_files (
  id INTEGER PRIMARY KEY,
  row_json TEXT,
  CONSTRAINT deferred_source_files_row_json_check CHECK (json_valid("row_json"))
);`
	ddl := convertTablesDDL(t, sql)
	if strings.Contains(ddl, "json_valid") {
		t.Fatalf("json_valid must be rewritten, got:\n%s", ddl)
	}
	if !strings.Contains(ddl, `("row_json") IS JSON`) {
		t.Fatalf("expected IS JSON rewrite:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertCheckConstraintDrizzleJSONValidColumnLevel(t *testing.T) {
	sql := `CREATE TABLE docs (
  id INTEGER PRIMARY KEY,
  payload TEXT NOT NULL CHECK (json_valid(payload))
);`
	ddl := convertTablesDDL(t, sql)
	if strings.Contains(ddl, "json_valid") {
		t.Fatalf("json_valid must be rewritten, got:\n%s", ddl)
	}
	if !strings.Contains(ddl, `("payload") IS JSON`) {
		t.Fatalf("expected IS JSON rewrite:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertGeneratedJSONExtract(t *testing.T) {
	sql := `CREATE TABLE items (
  id INTEGER PRIMARY KEY,
  data TEXT,
  title TEXT GENERATED ALWAYS AS (json_extract(data, '$.title')) VIRTUAL
);`
	ddl := convertTablesDDL(t, sql)
	if strings.Contains(ddl, "json_extract") {
		t.Fatalf("json_extract must be rewritten, got:\n%s", ddl)
	}
	if !strings.Contains(ddl, `GENERATED ALWAYS AS ((("data")::jsonb #>> '{title}')) STORED`) {
		t.Fatalf("expected json_extract rewrite:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertCheckConstraintIfnullInstrIif(t *testing.T) {
	sql := `CREATE TABLE t (
  id INTEGER PRIMARY KEY,
  name TEXT,
  qty INTEGER,
  CHECK (ifnull(qty, 0) >= 0),
  CHECK (instr(name, '@') = 0),
  CHECK (iif(qty > 0, 1, 0) = 1)
);`
	ddl := convertTablesDDL(t, sql)
	for _, leftover := range []string{"ifnull(", "instr(", "iif("} {
		if strings.Contains(strings.ToLower(ddl), leftover) {
			t.Fatalf("%s must be rewritten, got:\n%s", leftover, ddl)
		}
	}
	if !strings.Contains(ddl, `coalesce("qty", 0)`) {
		t.Fatalf("expected ifnull → coalesce:\n%s", ddl)
	}
	if !strings.Contains(ddl, `strpos("name", '@')`) {
		t.Fatalf("expected instr → strpos:\n%s", ddl)
	}
	if !strings.Contains(ddl, `CASE WHEN ("qty" > 0)`) {
		t.Fatalf("expected iif → CASE:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestLintSQLiteFunctionUnconverted(t *testing.T) {
	sql := `CREATE TABLE t (
  id INTEGER PRIMARY KEY,
  name TEXT,
  data TEXT,
  CHECK (printf('%s', name) = name),
  CHECK (json_valid(data))
);`
	result, err := Lint(writeDump(t, sql))
	if err != nil {
		t.Fatal(err)
	}
	var foundPrintf, foundJSONValid bool
	for _, issue := range result.Issues {
		if issue.Code != "SQLITE_FUNCTION" {
			continue
		}
		if strings.Contains(strings.ToLower(issue.Message), "printf") {
			foundPrintf = true
		}
		if strings.Contains(strings.ToLower(issue.Message), "json_valid") {
			foundJSONValid = true
		}
	}
	if !foundPrintf {
		t.Fatalf("expected SQLITE_FUNCTION error for printf, issues=%#v", result.Issues)
	}
	if foundJSONValid {
		t.Fatal("json_valid is converted and must not be a lint error")
	}
}

func TestUnconvertedSQLiteFunctions(t *testing.T) {
	if got := unconvertedSQLiteFunctions("json_valid(data)"); len(got) != 0 {
		t.Fatalf("json_valid is convertible, got %v", got)
	}
	if got := unconvertedSQLiteFunctions("printf('%s', name)"); len(got) != 1 || !strings.EqualFold(got[0], "printf") {
		t.Fatalf("printf: got %v", got)
	}
	if got := unconvertedSQLiteFunctions("name GLOB pattern"); len(got) != 1 || got[0] != "GLOB" {
		t.Fatalf("non-literal GLOB: got %v", got)
	}
	if got := unconvertedSQLiteFunctions("name GLOB '*.md'"); len(got) != 0 {
		t.Fatalf("literal GLOB is convertible, got %v", got)
	}
}
