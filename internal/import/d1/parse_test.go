package d1

import "testing"

func TestParseColumnUniqueConstraint(t *testing.T) {
	col := parseColumn("email TEXT NOT NULL UNIQUE")
	if !col.Unique {
		t.Fatal("expected column-level UNIQUE constraint")
	}

	col = parseColumn("unique_token TEXT NOT NULL")
	if col.Unique {
		t.Fatalf("identifier unique_token should not be treated as UNIQUE constraint")
	}

	col = parseColumn("unique_id INTEGER PRIMARY KEY")
	if col.Unique {
		t.Fatalf("identifier unique_id should not be treated as UNIQUE constraint")
	}
}
