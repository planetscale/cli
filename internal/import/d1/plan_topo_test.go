package d1

import (
	"strings"
	"testing"
)

func TestTopologicalLoadOrderCaseInsensitiveFK(t *testing.T) {
	tables := []TableSchema{
		{Name: "creator_turns"},
		{
			Name: "creator_turn_citations",
			Columns: []ColumnSchema{{
				Name:       "turn_id",
				Type:       "INTEGER",
				ForeignKey: `REFERENCES Creator_Turns(id)`,
			}},
			RawDDL: `CREATE TABLE creator_turn_citations (id INTEGER PRIMARY KEY, turn_id INTEGER, FOREIGN KEY (turn_id) REFERENCES Creator_Turns(id))`,
		},
	}
	order := topologicalLoadOrder(tables)
	if len(order) != 2 {
		t.Fatalf("order=%v", order)
	}
	if order[0] != "creator_turns" || order[1] != "creator_turn_citations" {
		t.Fatalf("expected creator_turns before creator_turn_citations, got %v", order)
	}
}

func TestBuildImportTablesSQLCaseInsensitiveFKOrder(t *testing.T) {
	tables := []TableSchema{
		{
			Name: "creator_turns",
			Columns: []ColumnSchema{{
				Name:        "id",
				Type:        "INTEGER",
				PrimaryKey:  true,
				AutoIncrement: true,
			}},
			RawDDL: `CREATE TABLE creator_turns (id INTEGER PRIMARY KEY AUTOINCREMENT)`,
		},
		{
			Name: "creator_turn_citations",
			Columns: []ColumnSchema{
				{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
				{Name: "turn_id", Type: "INTEGER", ForeignKey: `REFERENCES Creator_Turns(id)`},
			},
			RawDDL: `CREATE TABLE creator_turn_citations (id INTEGER PRIMARY KEY AUTOINCREMENT, turn_id INTEGER, FOREIGN KEY (turn_id) REFERENCES Creator_Turns(id))`,
		},
	}

	sql, err := buildImportTablesSQL("", tables)
	if err != nil {
		t.Fatalf("buildImportTablesSQL: %v", err)
	}
	turnsIdx := strings.Index(sql, `CREATE TABLE IF NOT EXISTS "creator_turns"`)
	citationsIdx := strings.Index(sql, `CREATE TABLE IF NOT EXISTS "creator_turn_citations"`)
	if turnsIdx < 0 || citationsIdx < 0 {
		t.Fatalf("expected both tables in import SQL:\n%s", sql)
	}
	if turnsIdx > citationsIdx {
		t.Fatalf("expected creator_turns before creator_turn_citations in import SQL:\n%s", sql)
	}
}
