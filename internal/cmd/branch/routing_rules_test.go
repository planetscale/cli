package branch

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
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
	c.Assert(cmd.Long, qt.Contains, "--reject-stale")
	c.Assert(cmd.Long, qt.Contains, "vtctld get-routing-rules")
	c.Assert(cmd.Long, qt.Contains, "replaces the entire cluster routing map")
}

func TestRoutingRulesGetRejectsStaleSnapshot(t *testing.T) {
	c := qt.New(t)

	svc := &mock.DatabaseBranchesService{
		RoutingRulesFn: func(_ context.Context, req *ps.BranchRoutingRulesRequest) (*ps.RoutingRules, error) {
			c.Assert(req.RejectStale, qt.IsTrue)
			return &ps.RoutingRules{Raw: `{"rules":[]}`}, nil
		},
	}
	format := printer.JSON
	p := printer.NewPrinter(&format)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DatabaseBranches: svc}, nil
		},
	}

	cmd := GetRoutingRulesCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch", "--reject-stale"})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.RoutingRulesFnInvoked, qt.IsTrue)
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
