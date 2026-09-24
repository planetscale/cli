package branch

import (
	"context"
	"os"
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
	c.Assert(cmd.Long, qt.Contains, "fails when a routing change has not produced a new schema snapshot")
	c.Assert(cmd.Long, qt.Contains, "vtctld get-routing-rules")
}

func TestRoutingRulesUpdateRejectsStaleSnapshot(t *testing.T) {
	c := qt.New(t)

	svc := &mock.DatabaseBranchesService{
		UpdateRoutingRulesFn: func(_ context.Context, _ *ps.UpdateBranchRoutingRulesRequest) (*ps.RoutingRules, error) {
			return nil, &ps.Error{APICode: "routing_rules_snapshot_stale"}
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

	tmpFile, err := os.CreateTemp("", "routing-rules.json")
	c.Assert(err, qt.IsNil)
	_, err = tmpFile.Write([]byte(`{"rules":[]}`))
	c.Assert(err, qt.IsNil)
	c.Assert(tmpFile.Close(), qt.IsNil)

	cmd := UpdateRoutingRulesCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch", "--routing-rules", tmpFile.Name()})

	c.Assert(cmd.Execute(), qt.IsNotNil)
	c.Assert(svc.UpdateRoutingRulesFnInvoked, qt.IsTrue)
}
