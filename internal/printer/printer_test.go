package printer

import (
	"bytes"
	"strings"
	"testing"
)

type nilCellRow struct {
	Name      string `header:"name" json:"name"`
	MaxCount  *int   `header:"max_count,n/a" json:"max_count"`
	Enabled   bool   `header:"enabled" json:"enabled"`
	Target    *int   `header:"target,n/a" json:"target"`
	CreatedAt int64  `header:"created_at" json:"created_at"`
}

// A nil pointer field used to render no cell at all, so every value after it
// lined up under the wrong header.
func TestPrintResourceKeepsColumnsAlignedWithNilFields(t *testing.T) {
	format := Human
	var out bytes.Buffer
	p := NewPrinter(&format)
	p.SetResourceOutput(&out)

	if err := p.PrintResource(&nilCellRow{Name: "main", Enabled: true, CreatedAt: 42}); err != nil {
		t.Fatalf("print resource: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected a header and a row, got %q", out.String())
	}

	// The separator line has one run of dashes per rendered column.
	columns := len(strings.Fields(lines[1]))
	values := strings.Fields(lines[len(lines)-1])
	if len(values) != columns {
		t.Fatalf("expected %d values for %d columns, got %v", columns, columns, values)
	}

	want := []string{"main", "n/a", "Yes", "n/a", "42"}
	for i := range want {
		if values[i] != want[i] {
			t.Errorf("column %d: got %q, want %q", i, values[i], want[i])
		}
	}
}

func TestPrintJSONDoesNotEscapeHTMLPlaceholders(t *testing.T) {
	format := JSON
	var out bytes.Buffer
	p := NewPrinter(&format)
	p.SetResourceOutput(&out)

	if err := p.PrintJSON(map[string]string{"next_step": "pscale branch list <database> --org <org> --format json"}); err != nil {
		t.Fatalf("print json: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "<database>") || !strings.Contains(got, "<org>") {
		t.Fatalf("expected unescaped placeholders, got %s", got)
	}
	if strings.Contains(got, `\u003c`) || strings.Contains(got, `\u003e`) {
		t.Fatalf("expected no escaped angle brackets, got %s", got)
	}
}
