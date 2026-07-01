package d1

import "testing"

func TestParseTableLevelForeignKey(t *testing.T) {
	cols, refs := parseTableLevelForeignKey(`FOREIGN KEY (entity_id) REFERENCES external_entities(id)`)
	if len(cols) != 1 || cols[0] != "entity_id" {
		t.Fatalf("unexpected columns: %#v", cols)
	}
	refTable, refCol := parseReferencesTarget(refs)
	if refTable != "external_entities" || refCol != "id" {
		t.Fatalf("unexpected ref target: %s.%s", refTable, refCol)
	}
}

func TestColumnFKTargetUsesTableConstraint(t *testing.T) {
	table := TableSchema{
		Name: "entity_links",
		Columns: []ColumnSchema{
			{Name: "entity_id", Type: "TEXT", NotNull: true},
			{Name: "post_id", Type: "INTEGER", NotNull: true},
		},
		Constraints: []string{
			`PRIMARY KEY (entity_id, post_id)`,
			`FOREIGN KEY (entity_id) REFERENCES external_entities(id)`,
			`FOREIGN KEY (post_id) REFERENCES posts(id)`,
		},
	}
	col := table.Columns[0]

	refTable, refCol := columnFKTarget(col, table)
	if refTable != "external_entities" || refCol != "id" {
		t.Fatalf("got %s.%s", refTable, refCol)
	}
}
