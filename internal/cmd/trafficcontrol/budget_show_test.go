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

func TestBudgetShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "database"
	branch := "main"
	budgetID := "qok87ki4xlau"
	cap := 80

	budget := &ps.TrafficBudget{
		ID:        budgetID,
		Name:      "CPU Limiter",
		Mode:      "enforce",
		Capacity:  &cap,
		Rules:     []ps.TrafficRule{{ID: "rule-1", Kind: "fingerprint"}},
		CreatedAt: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
	}

	svc := &mock.TrafficBudgetsService{
		GetFn: func(ctx context.Context, req *ps.GetTrafficBudgetRequest) (*ps.TrafficBudget, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.BudgetID, qt.Equals, budgetID)
			return budget, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{TrafficBudgets: svc}, nil
		},
	}

	cmd := BudgetShowCmd(ch)
	cmd.SetArgs([]string{db, branch, budgetID})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)

	res := &TrafficBudget{orig: budget}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBudgetShowCmd_NotFound(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "database"
	branch := "main"
	budgetID := "qok87ki4xlau"

	svc := &mock.TrafficBudgetsService{
		GetFn: func(ctx context.Context, req *ps.GetTrafficBudgetRequest) (*ps.TrafficBudget, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{TrafficBudgets: svc}, nil
		},
	}

	cmd := BudgetShowCmd(ch)
	cmd.SetArgs([]string{db, branch, budgetID})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
}
