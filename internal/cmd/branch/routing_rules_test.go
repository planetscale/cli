package branch

import (
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
)

func TestRoutingRulesGetHelpDescribesSnapshot(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
	}

	cmd := GetRoutingRulesCmd(ch)
	c.Assert(cmd.Short, qt.Contains, "schema snapshot")
	c.Assert(cmd.Long, qt.Contains, "vtctld get-routing-rules")
	c.Assert(cmd.Long, qt.Contains, "replaces the entire cluster routing map")
}

func TestRoutingRulesUpdateHelpDescribesReplace(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
	}

	cmd := UpdateRoutingRulesCmd(ch)
	c.Assert(cmd.Short, qt.Contains, "Replace")
	c.Assert(cmd.Long, qt.Contains, "full replacement")
	c.Assert(cmd.Long, qt.Contains, "vtctld get-routing-rules")
}
