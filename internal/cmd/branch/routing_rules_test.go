package branch

import (
	"bytes"
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

func TestRoutingRulesGetWarnsWhenSchemaMutationIsInProgress(t *testing.T) {
	c := qt.New(t)

	svc := &mock.DatabaseBranchesService{
		RoutingRulesFn: func(_ context.Context, _ *ps.BranchRoutingRulesRequest) (*ps.RoutingRules, error) {
			return &ps.RoutingRules{
				Raw: `{"rules":[]}`,
				Warnings: []ps.RoutingRulesWarning{{
					Code:    "schema_mutation_in_progress",
					Message: "A vtctld schema mutation is in progress. The routing rules in this response may not describe live routing.",
				}},
			}, nil
		},
	}
	format := printer.Human
	p := printer.NewPrinter(&format)
	var output bytes.Buffer
	p.SetHumanOutput(&output)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DatabaseBranches: svc}, nil
		},
	}

	cmd := GetRoutingRulesCmd(ch)
	var warnings bytes.Buffer
	cmd.SetErr(&warnings)
	cmd.SetArgs([]string{"my-db", "my-branch"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(warnings.String(), qt.Contains, "Warning: A vtctld schema mutation is in progress.")
	c.Assert(warnings.String(), qt.Contains, "may not describe live routing")
	c.Assert(output.String(), qt.Contains, `"rules": []`)
}

func TestRoutingRulesUpdateHelpDescribesBlockedConditions(t *testing.T) {
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
	c.Assert(cmd.Long, qt.Contains, "vtctld schema mutation is in progress")
	c.Assert(cmd.Long, qt.Contains, "schema snapshot is not ready")
}

func TestRoutingRulesUpdateReturnsBlockedError(t *testing.T) {
	c := qt.New(t)

	svc := &mock.DatabaseBranchesService{
		UpdateRoutingRulesFn: func(_ context.Context, _ *ps.UpdateBranchRoutingRulesRequest) (*ps.RoutingRules, error) {
			return nil, &ps.Error{APICode: "schema_snapshot_not_ready"}
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
