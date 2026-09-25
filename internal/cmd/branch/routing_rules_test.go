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

func TestRoutingRulesUpdateHelpDescribesStaleSnapshotBlock(t *testing.T) {
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
}

func TestRoutingRulesUpdateReturnsStaleSnapshotError(t *testing.T) {
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
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	_, err = tmpFile.WriteString(`{"rules":[]}`)
	c.Assert(err, qt.IsNil)
	c.Assert(tmpFile.Close(), qt.IsNil)

	cmd := UpdateRoutingRulesCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch", "--routing-rules", tmpFile.Name()})

	err = cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(svc.UpdateRoutingRulesFnInvoked, qt.IsTrue)
}
