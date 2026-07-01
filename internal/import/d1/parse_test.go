package d1

import "testing"

func TestParseColumnDefaultBeforeNotNull(t *testing.T) {
	col := parseColumn("active INTEGER DEFAULT 1 NOT NULL")
	if col.DefaultValue != "1" {
		t.Fatalf("default = %q, want 1", col.DefaultValue)
	}
	if !col.NotNull {
		t.Fatal("expected NOT NULL")
	}
}

func TestParseColumnDefaultStringNotNullNotConstraint(t *testing.T) {
	col := parseColumn("status TEXT DEFAULT 'value NOT NULL'")
	if col.DefaultValue != "'value NOT NULL'" {
		t.Fatalf("default = %q, want quoted literal", col.DefaultValue)
	}
	if col.NotNull {
		t.Fatal("NOT NULL inside default string must not set column constraint")
	}
}

func TestParseColumnCheckNotNullNotConstraint(t *testing.T) {
	col := parseColumn("status TEXT CHECK (status IS NOT NULL)")
	if col.NotNull {
		t.Fatal("NOT NULL inside CHECK must not set column constraint")
	}
}

func TestTrimDefaultClause(t *testing.T) {
	cases := map[string]string{
		"1 NOT NULL":              "1",
		"'draft' NOT NULL UNIQUE": "'draft'",
		"CURRENT_TIMESTAMP":       "CURRENT_TIMESTAMP",
	}
	for in, want := range cases {
		if got := trimDefaultClause(in); got != want {
			t.Fatalf("trimDefaultClause(%q) = %q, want %q", in, got, want)
		}
	}
}

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
