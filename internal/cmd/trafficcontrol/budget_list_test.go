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
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestBudgetListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	budgets := []*ps.TrafficBudget{
		{
			ID:        budgetID,
			Name:      "CPU Limiter",
			Mode:      "enforce",
			CreatedAt: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	svc := &mock.TrafficBudgetsService{
		ListFn: func(ctx context.Context, req *ps.ListTrafficBudgetsRequest) ([]*ps.TrafficBudget, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Fingerprint, qt.Equals, "")
			return budgets, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{TrafficBudgets: svc}, nil
		},
	}

	cmd := BudgetListCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, []*TrafficBudget{{orig: budgets[0]}})
}

func TestBudgetListCmd_FiltersByFingerprint(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.TrafficBudgetsService{
		ListFn: func(ctx context.Context, req *ps.ListTrafficBudgetsRequest) ([]*ps.TrafficBudget, error) {
			c.Assert(req.Fingerprint, qt.Equals, "abc123")
			return nil, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{TrafficBudgets: svc}, nil
		},
	}

	cmd := BudgetListCmd(ch)
	cmd.SetArgs([]string{db, branch, "--fingerprint", "abc123"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
}

func TestBudgetListCmd_NotFound(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.TrafficBudgetsService{
		ListFn: func(ctx context.Context, req *ps.ListTrafficBudgetsRequest) ([]*ps.TrafficBudget, error) {
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

	cmd := BudgetListCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
}
