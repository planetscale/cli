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

func TestBudgetUpdateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	branch := "main"
	budgetID := "budget-123"
	cap := 90

	updated := &ps.TrafficBudget{
		ID:        budgetID,
		Name:      "Renamed Budget",
		Mode:      "warn",
		Capacity:  &cap,
		CreatedAt: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 6, 16, 12, 0, 0, 0, time.UTC),
	}

	svc := &mock.TrafficBudgetsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateTrafficBudgetRequest) (*ps.TrafficBudget, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.BudgetID, qt.Equals, budgetID)
			c.Assert(*req.Name, qt.Equals, "Renamed Budget")
			c.Assert(*req.Mode, qt.Equals, "warn")
			c.Assert(*req.Capacity, qt.Equals, 90)
			c.Assert(req.Rate, qt.IsNil)
			c.Assert(req.Burst, qt.IsNil)
			c.Assert(req.Concurrency, qt.IsNil)
			c.Assert(req.Rules, qt.IsNil)
			return updated, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{TrafficBudgets: svc}, nil
		},
	}

	cmd := BudgetUpdateCmd(ch)
	cmd.SetArgs([]string{db, branch, budgetID,
		"--name", "Renamed Budget",
		"--mode", "warn",
		"--capacity", "90",
	})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)

	res := &TrafficBudget{orig: updated}
	c.Assert(buf.String(), qt.JSONEquals, res)
}
