package trafficcontrol

import (
	"bytes"
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestBudgetDeleteCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.TrafficBudgetsService{
		DeleteFn: func(ctx context.Context, req *ps.DeleteTrafficBudgetRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.BudgetID, qt.Equals, budgetID)
			return nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{TrafficBudgets: svc}, nil
		},
	}

	cmd := BudgetDeleteCmd(ch)
	cmd.SetArgs([]string{db, branch, budgetID, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)

	res := map[string]string{
		"result":    "budget deleted",
		"budget_id": budgetID,
		"database":  db,
		"branch":    branch,
	}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBudgetDeleteCmd_NotFound(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.TrafficBudgetsService{
		DeleteFn: func(ctx context.Context, req *ps.DeleteTrafficBudgetRequest) error {
			return &ps.Error{Code: ps.ErrNotFound}
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{TrafficBudgets: svc}, nil
		},
	}

	cmd := BudgetDeleteCmd(ch)
	cmd.SetArgs([]string{db, branch, budgetID, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)
}
