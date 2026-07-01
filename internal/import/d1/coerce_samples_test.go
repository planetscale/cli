package d1

import "testing"

func TestLooksLikeTimestamp(t *testing.T) {
	cases := map[string]bool{
		"2024-01-15 12:00:00":  true,
		"2024-01-15T12:00:00Z": true,
		"v1-beta":              false,
		"note: pending":        false,
		"draft-2024":           false,
		"CURRENT_TIMESTAMP":    false,
	}
	for val, want := range cases {
		if got := looksLikeTimestamp(val); got != want {
			t.Fatalf("looksLikeTimestamp(%q) = %v, want %v", val, got, want)
		}
	}
}

func TestSampleColumnValuesExternalEntities(t *testing.T) {
	tables, err := ParseDump(testFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	samples, err := SampleColumnValues(testFixture(t), tables)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples["external_entities"]["id"]) == 0 {
		t.Fatalf("expected external_entities.id samples, got %#v", samples["external_entities"])
	}
	ctx, err := BuildTypeCoercionContext(testFixture(t), tables)
	if err != nil {
		t.Fatal(err)
	}
	if !samplesAllowUUID("external_entities", "id", ctx) {
		t.Fatal("expected uuid samples for external_entities.id")
	}
}
