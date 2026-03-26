package trafficcontrol

import (
	"bytes"
	"context"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

var (
	ruleID = "9rvegkj5qmri"
)

func TestRuleCreateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	created := &ps.TrafficRule{
		ID:   ruleID,
		Type: "TrafficRule",
		Kind: "match",
		Tags: []ps.TrafficRuleTag{
			{Type: "TrafficRuleTag", KeyID: "Squery", Key: "query", Value: "SELECT 1", Source: "sql"},
		},
		Actor:     ps.Actor{ID: "user-1", Type: "User", Name: "Alice"},
		CreatedAt: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
	}

	svc := &mock.TrafficRulesService{
		CreateFn: func(ctx context.Context, req *ps.CreateTrafficRuleRequest) (*ps.TrafficRule, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.BudgetID, qt.Equals, budgetID)
			c.Assert(req.Kind, qt.Equals, "match")
			c.Assert(req.Tags, qt.IsNotNil)
			c.Assert(*req.Tags, qt.HasLen, 1)
			c.Assert((*req.Tags)[0].Key, qt.Equals, "query")
			c.Assert((*req.Tags)[0].Value, qt.Equals, "SELECT 1")
			c.Assert((*req.Tags)[0].Source, qt.Equals, "sql")
			return created, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{TrafficRules: svc}, nil
		},
	}

	cmd := RuleCreateCmd(ch)
	cmd.SetArgs([]string{db, branch, budgetID,
		"--kind", "match",
		"--tag", "key=query,value=SELECT 1",
	})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)

	res := &TrafficRuleDisplay{orig: created}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestRuleCreateCmd_Fingerprint(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	fp := "abc123"

	created := &ps.TrafficRule{
		ID:          ruleID,
		Type:        "TrafficRule",
		Kind:        "match",
		Fingerprint: &fp,
		CreatedAt:   time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
	}

	svc := &mock.TrafficRulesService{
		CreateFn: func(ctx context.Context, req *ps.CreateTrafficRuleRequest) (*ps.TrafficRule, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.BudgetID, qt.Equals, budgetID)
			c.Assert(req.Kind, qt.Equals, "match")
			c.Assert(*req.Fingerprint, qt.Equals, "abc123")
			c.Assert(req.Tags, qt.IsNil)
			return created, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{TrafficRules: svc}, nil
		},
	}

	cmd := RuleCreateCmd(ch)
	cmd.SetArgs([]string{db, branch, budgetID,
		"--fingerprint", "abc123",
	})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)

	res := &TrafficRuleDisplay{orig: created}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestRuleCreateCmd_MultipleTags(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	created := &ps.TrafficRule{
		ID:   ruleID,
		Type: "TrafficRule",
		Kind: "match",
		Tags: []ps.TrafficRuleTag{
			{Type: "TrafficRuleTag", Key: "query", Value: "SELECT 1", Source: "sql"},
			{Type: "TrafficRuleTag", Key: "user", Value: "admin", Source: "context"},
		},
		CreatedAt: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
	}

	svc := &mock.TrafficRulesService{
		CreateFn: func(ctx context.Context, req *ps.CreateTrafficRuleRequest) (*ps.TrafficRule, error) {
			c.Assert(req.Kind, qt.Equals, "match")
			c.Assert(req.Tags, qt.IsNotNil)
			c.Assert(*req.Tags, qt.HasLen, 2)
			c.Assert((*req.Tags)[0].Key, qt.Equals, "query")
			c.Assert((*req.Tags)[0].Value, qt.Equals, "SELECT 1")
			c.Assert((*req.Tags)[0].Source, qt.Equals, "sql")
			c.Assert((*req.Tags)[1].Key, qt.Equals, "user")
			c.Assert((*req.Tags)[1].Value, qt.Equals, "admin")
			c.Assert((*req.Tags)[1].Source, qt.Equals, "context")
			return created, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{TrafficRules: svc}, nil
		},
	}

	cmd := RuleCreateCmd(ch)
	cmd.SetArgs([]string{db, branch, budgetID,
		"--kind", "match",
		"--tag", "key=query,value=SELECT 1,source=sql",
		"--tag", "key=user,value=admin,source=context",
	})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)

	res := &TrafficRuleDisplay{orig: created}
	c.Assert(buf.String(), qt.JSONEquals, res)
}
