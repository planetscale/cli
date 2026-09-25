package vtctld

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestGetKeyspaceRoutingRules(t *testing.T) {
	c := qt.New(t)

	svc := &mock.VtctldService{
		GetKeyspaceRoutingRulesFn: func(_ context.Context, req *ps.VtctldGetKeyspaceRoutingRulesRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, "my-org")
			c.Assert(req.Database, qt.Equals, "my-db")
			c.Assert(req.Branch, qt.Equals, "my-branch")
			return json.RawMessage(`{"rules":[]}`), nil
		},
	}
	ch := keyspaceRoutingRulesHelper(svc)

	cmd := GetKeyspaceRoutingRulesCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch"})

	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetKeyspaceRoutingRulesFnInvoked, qt.IsTrue)
}

func TestApplyKeyspaceRoutingRules(t *testing.T) {
	c := qt.New(t)

	svc := &mock.VtctldService{
		ApplyKeyspaceRoutingRulesFn: func(_ context.Context, req *ps.VtctldApplyKeyspaceRoutingRulesRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, "my-org")
			c.Assert(req.Database, qt.Equals, "my-db")
			c.Assert(req.Branch, qt.Equals, "my-branch")
			c.Assert(req.Rules, qt.DeepEquals, []ps.VtctldKeyspaceRoutingRule{{
				FromKeyspace: "source",
				ToKeyspace:   "target",
			}})
			c.Assert(req.SkipRebuild, qt.IsTrue)
			c.Assert(req.RebuildCells, qt.DeepEquals, []string{"zone1"})
			return json.RawMessage(`{"rules":[]}`), nil
		},
	}
	ch := keyspaceRoutingRulesHelper(svc)

	cmd := ApplyKeyspaceRoutingRulesCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch"})
	c.Assert(cmd.Flags().Set("rules", `{"rules":[{"from_keyspace":"source","to_keyspace":"target"}]}`), qt.IsNil)
	c.Assert(cmd.Flags().Set("skip-rebuild", "true"), qt.IsNil)
	c.Assert(cmd.Flags().Set("cells", "zone1"), qt.IsNil)

	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ApplyKeyspaceRoutingRulesFnInvoked, qt.IsTrue)
}

func TestApplyKeyspaceRoutingRulesFileClearsRules(t *testing.T) {
	c := qt.New(t)

	svc := &mock.VtctldService{
		ApplyKeyspaceRoutingRulesFn: func(_ context.Context, req *ps.VtctldApplyKeyspaceRoutingRulesRequest) (json.RawMessage, error) {
			c.Assert(req.Rules, qt.DeepEquals, []ps.VtctldKeyspaceRoutingRule{})
			return json.RawMessage(`{"rules":[]}`), nil
		},
	}
	ch := keyspaceRoutingRulesHelper(svc)

	rulesFile := filepath.Join(t.TempDir(), "rules.json")
	c.Assert(os.WriteFile(rulesFile, []byte(`{}`), 0o600), qt.IsNil)

	cmd := ApplyKeyspaceRoutingRulesCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch"})
	c.Assert(cmd.Flags().Set("rules-file", rulesFile), qt.IsNil)

	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ApplyKeyspaceRoutingRulesFnInvoked, qt.IsTrue)
}

func TestApplyKeyspaceRoutingRulesRequiresOneInput(t *testing.T) {
	c := qt.New(t)

	cmd := ApplyKeyspaceRoutingRulesCmd(keyspaceRoutingRulesHelper(&mock.VtctldService{}))
	cmd.SetArgs([]string{"my-db", "my-branch"})

	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, "must specify exactly one of --rules or --rules-file")
}

func keyspaceRoutingRulesHelper(svc *mock.VtctldService) *cmdutil.Helper {
	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Vtctld: svc}, nil
		},
	}
}
