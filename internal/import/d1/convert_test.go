package d1

import "testing"

func TestMapSQLiteDefaultFunctionUnixEpoch(t *testing.T) {
	cases := map[string]string{
		"unixepoch('now')":    "to_timestamp('now')",
		"UNIXEPOCH('now')":    "to_timestamp('now')",
		"UnixEpoch('now')":    "to_timestamp('now')",
		"(UNIXEPOCH('now'))":  "to_timestamp('now')",
		"UNIXEPOCH('subsec')": "to_timestamp('subsec')",
		"CURRENT_TIMESTAMP":   "CURRENT_TIMESTAMP",
		"datetime('now')":     "now()",
	}
	for def, want := range cases {
		got := mapSQLiteDefaultFunction(def, "TIMESTAMPTZ")
		if got != want {
			t.Fatalf("mapSQLiteDefaultFunction(%q) = %q, want %q", def, got, want)
		}
	}
}
