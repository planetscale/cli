package inspect

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
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

func TestInspectCmd_NekiTargetFlags(t *testing.T) {
	c := qt.New(t)

	cmd := InspectCmd(&cmdutil.Helper{Config: &config.Config{Organization: "org"}})
	c.Assert(cmd.PersistentFlags().Lookup("shard"), qt.IsNotNil)
	c.Assert(cmd.PersistentFlags().Lookup("router"), qt.IsNotNil)
	c.Assert(cmd.Long, qt.Contains, "--shard")
	c.Assert(cmd.Long, qt.Contains, "--router")
	c.Assert(cmd.PersistentFlags().Lookup("shard").Usage, qt.Contains, "Neki")
	c.Assert(cmd.PersistentFlags().Lookup("router").Usage, qt.Contains, "Neki")
}
