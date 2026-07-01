package d1

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestVerifyRowCounts(t *testing.T) {
	source := map[string]int64{"users": 2, "posts": 2}
	dest := map[string]int64{"users": 2, "posts": 1}

	results, ok := verifyRowCounts([]string{"users", "posts"}, source, dest)
	if ok {
		t.Fatal("expected mismatch")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 table results, got %d", len(results))
	}
	if !results[0].Match || results[1].Match {
		t.Fatalf("unexpected match flags: %+v", results)
	}
}

func TestVerifyRowCountsIncludesImportScopedDestTables(t *testing.T) {
	source := map[string]int64{"users": 1}
	dest := map[string]int64{"users": 1, "legacy_import_table": 5}
	results, ok := verifyRowCounts([]string{"users", "legacy_import_table"}, source, dest)
	if ok {
		t.Fatal("expected mismatch when import-scoped dest table has rows but source does not")
	}
	found := false
	for _, r := range results {
		if r.Table == "legacy_import_table" && !r.Match && r.DestRows == 5 && r.SourceRows == 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected import-scoped dest table mismatch, got %+v", results)
	}
}

func TestColumnReferencesUUIDKey(t *testing.T) {
	tables, err := ParseDump(testFixture(t))
	if err != nil {
		t.Fatalf("ParseDump: %v", err)
	}
	coerceCtx, err := BuildTypeCoercionContext(testFixture(t), tables)
	if err != nil {
		t.Fatalf("BuildTypeCoercionContext: %v", err)
	}

	var entityLinks TableSchema
	for _, table := range tables {
		if table.Name == "entity_links" {
			entityLinks = table
			break
		}
	}
	if entityLinks.Name == "" {
		t.Fatal("missing entity_links table")
	}

	var entityID ColumnSchema
	for _, col := range entityLinks.Columns {
		if col.Name == "entity_id" {
			entityID = col
			break
		}
	}
	if entityID.Name == "" {
		t.Fatal("missing entity_id column")
	}
	if !columnReferencesUUIDKey(entityID, entityLinks, tables, coerceCtx) {
		t.Fatal("expected entity_id to reference UUID primary key")
	}
	if isExplicitUUIDColumn(entityID) {
		t.Fatal("entity_id should not be treated as explicit UUID column")
	}
}

func TestColumnReferencesUUIDKeyCycle(t *testing.T) {
	tables := []TableSchema{
		{
			Name: "nodes_a",
			Columns: []ColumnSchema{{
				Name:       "next_id",
				Type:       "TEXT",
				ForeignKey: `REFERENCES nodes_b(id)`,
			}},
		},
		{
			Name: "nodes_b",
			Columns: []ColumnSchema{{
				Name:       "next_id",
				Type:       "TEXT",
				ForeignKey: `REFERENCES nodes_a(id)`,
			}},
		},
	}
	if columnReferencesUUIDKey(tables[0].Columns[0], tables[0], tables, nil) {
		t.Fatal("expected cyclic FK chain to resolve as non-UUID without stack overflow")
	}
}

func TestLooksLikeRailsSchemaMigrations(t *testing.T) {
	rails := TableSchema{
		Name: "schema_migrations",
		Columns: []ColumnSchema{{
			Name: "version",
			Type: "VARCHAR(255)",
		}},
	}
	if !looksLikeRailsSchemaMigrations(rails) {
		t.Fatal("expected rails-like schema_migrations")
	}

	appTable := TableSchema{
		Name: "schema_migrations",
		Columns: []ColumnSchema{
			{Name: "id", Type: "INTEGER", PrimaryKey: true},
			{Name: "name", Type: "TEXT"},
		},
	}
	if looksLikeRailsSchemaMigrations(appTable) {
		t.Fatal("expected app schema_migrations to differ from rails layout")
	}
}

func TestParseSQLiteCLIFields(t *testing.T) {
	got := parseSQLiteCLIFields([]byte("120|0|0\n"))
	if len(got) != 3 || got[0] != "120" || got[1] != "0" || got[2] != "0" {
		t.Fatalf("parseSQLiteCLIFields() = %v", got)
	}
	got = parseSQLiteCLIFields([]byte("94400 123456\n"))
	if len(got) != 2 || got[0] != "94400" {
		t.Fatalf("parseSQLiteCLIFields() = %v", got)
	}
}

func TestJSONValuesEqual(t *testing.T) {
	a := `{"priority": 0, "labels": ["seed"]}`
	b := `{"labels": ["seed"], "priority": 0}`
	if !jsonValuesEqual(a, b) {
		t.Fatal("expected equivalent JSON objects to match")
	}
	if jsonValuesEqual(a, `{"priority": 1}`) {
		t.Fatal("expected different JSON objects to mismatch")
	}
}

func TestByteaValuesEqual(t *testing.T) {
	text := "attachment-1-payload"
	hex := `\x` + hex.EncodeToString([]byte(text))
	if !byteaValuesEqual(text, hex) {
		t.Fatalf("expected bytea hex %q to match text %q", hex, text)
	}
}

func TestByteaSignatureExprsUseHex(t *testing.T) {
	col := ColumnSchema{Name: "payload", Type: "BLOB"}
	table := TableSchema{Name: "attachments", Columns: []ColumnSchema{col}}

	sqliteExpr := sqliteSignatureColumnExpr(col, table, nil)
	if !strings.Contains(sqliteExpr, "hex(") {
		t.Fatalf("sqlite blob signature should use hex(), got %q", sqliteExpr)
	}

	pgExpr := postgresSignatureColumnExpr(col, table, nil, nil)
	if !strings.Contains(pgExpr, "encode(") || !strings.Contains(pgExpr, "'hex'") {
		t.Fatalf("postgres bytea signature should use encode(..., 'hex'), got %q", pgExpr)
	}
}

func TestTimestampValuesEqual(t *testing.T) {
	if !timestampValuesEqual("2024-01-15 12:00:00", "2024-01-15T12:00:00Z") {
		t.Fatal("expected space and ISO timestamp forms to match")
	}
	if timestampValuesEqual("2024-01-15 12:00:00", "2024-01-16 12:00:00") {
		t.Fatal("expected different timestamps to mismatch")
	}
}

func TestByteaValuesEqualBinaryHex(t *testing.T) {
	raw := string([]byte{0x00, 0xff, 0xfe, 0x01})
	hexSig := hex.EncodeToString([]byte(raw))
	if !byteaValuesEqual(hexSig, hexSig) {
		t.Fatalf("expected matching hex signatures for binary blob")
	}
	if !byteaValuesEqual(strings.ToUpper(hexSig), strings.ToLower(hexSig)) {
		t.Fatalf("expected hex signatures to match regardless of case")
	}
}

func TestSummarizeRowSignatureForOutputOmitsBlobPayload(t *testing.T) {
	table := TableSchema{
		Name: "attachments",
		Columns: []ColumnSchema{
			{Name: "id", Type: "INTEGER"},
			{Name: "task_id", Type: "INTEGER"},
			{Name: "filename", Type: "TEXT"},
			{Name: "mime_type", Type: "TEXT"},
			{Name: "size_bytes", Type: "INTEGER"},
			{Name: "checksum", Type: "TEXT"},
			{Name: "payload", Type: "BLOB"},
		},
	}
	longHex := strings.Repeat("ab", 200)
	sig := strings.Join([]string{"3", "3", "file-3.bin", "application/octet-stream", "47952", "sha256:00000003", longHex}, "|")

	got := summarizeRowSignatureForOutput(sig, table)
	if strings.Contains(got, longHex) {
		t.Fatalf("expected blob hex to be truncated, got %q", got)
	}
	if !strings.Contains(got, "file-3.bin") {
		t.Fatalf("expected non-blob fields preserved, got %q", got)
	}
	if !strings.Contains(got, "(200 bytes)") {
		t.Fatalf("expected blob byte count in summary, got %q", got)
	}
}
