package inspect

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestPrintCSVExplainsSkippedCheck(t *testing.T) {
	c := qt.New(t)

	var out bytes.Buffer
	err := printCSV(&out, &CheckResult{
		Check:    "redundant-indexes",
		Database: "mydb",
		Branch:   "main",
		Skipped:  "This check is not available for PostgreSQL.",
		NextSteps: []string{
			"pscale insights recommendations mydb --org myorg --format json",
		},
	})
	c.Assert(err, qt.IsNil)

	records, err := csv.NewReader(strings.NewReader(out.String())).ReadAll()
	c.Assert(err, qt.IsNil)
	c.Assert(records, qt.DeepEquals, [][]string{
		{"check", "database", "branch", "skipped", "next_steps"},
		{
			"redundant-indexes",
			"mydb",
			"main",
			"This check is not available for PostgreSQL.",
			"pscale insights recommendations mydb --org myorg --format json",
		},
	})
}
