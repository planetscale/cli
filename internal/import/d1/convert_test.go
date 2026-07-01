package d1

import "testing"

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
	}
	for name, tc := range cases {
		got := mapSQLiteDefaultFunction(tc.def, tc.pgType)
		if got != tc.want {
			t.Fatalf("%s: mapSQLiteDefaultFunction(%q, %q) = %q, want %q", name, tc.def, tc.pgType, got, tc.want)
		}
	}
}
