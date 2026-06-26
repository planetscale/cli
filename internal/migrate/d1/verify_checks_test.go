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

func TestColumnReferencesUUIDKey(t *testing.T) {
	tables, err := ParseDump(testFixture(t))
	if err != nil {
		t.Fatalf("ParseDump: %v", err)
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
	if !columnReferencesUUIDKey(entityID, entityLinks, tables) {
		t.Fatal("expected entity_id to reference UUID primary key")
	}
	if isExplicitUUIDColumn(entityID) {
		t.Fatal("entity_id should not be treated as explicit UUID column")
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

	sqliteExpr := sqliteSignatureColumnExpr(col)
	if !strings.Contains(sqliteExpr, "hex(") {
		t.Fatalf("sqlite blob signature should use hex(), got %q", sqliteExpr)
	}

	pgExpr := postgresSignatureColumnExpr(col, table, nil)
	if !strings.Contains(pgExpr, "encode(") || !strings.Contains(pgExpr, "'hex'") {
		t.Fatalf("postgres bytea signature should use encode(..., 'hex'), got %q", pgExpr)
	}
}

func TestByteaValuesEqualBinaryHex(t *testing.T) {
	raw := string([]byte{0x00, 0xff, 0xfe, 0x01})
	hexSig := hex.EncodeToString([]byte(raw))
	if !byteaValuesEqual(hexSig, hexSig) {
		t.Fatalf("expected matching hex signatures for binary blob")
	}
}
