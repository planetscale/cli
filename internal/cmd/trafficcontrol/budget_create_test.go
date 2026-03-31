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

func TestBudgetCreateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	cap, rate, burst, conc, warnTh := 80, 50, 60, 40, 30

	created := &ps.TrafficBudget{
		ID:               budgetID,
		Name:             "CPU Limiter",
		Mode:             "enforce",
		Capacity:         &cap,
		Rate:             &rate,
		Burst:            &burst,
		Concurrency:      &conc,
		WarningThreshold: &warnTh,
		CreatedAt:        time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
	}

	svc := &mock.TrafficBudgetsService{
		CreateFn: func(ctx context.Context, req *ps.CreateTrafficBudgetRequest) (*ps.TrafficBudget, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, "CPU Limiter")
			c.Assert(req.Mode, qt.Equals, "enforce")
			c.Assert(*req.Capacity, qt.Equals, 80)
			c.Assert(*req.Rate, qt.Equals, 50)
			c.Assert(*req.Burst, qt.Equals, 60)
			c.Assert(*req.Concurrency, qt.Equals, 40)
			c.Assert(*req.WarningThreshold, qt.Equals, 30)
			return created, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{TrafficBudgets: svc}, nil
		},
	}

	cmd := BudgetCreateCmd(ch)
	cmd.SetArgs([]string{db, branch,
		"--name", "CPU Limiter",
		"--mode", "enforce",
		"--capacity", "80",
		"--rate", "50",
		"--burst", "60",
		"--concurrency", "40",
		"--warning-threshold", "30",
	})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)

	res := &TrafficBudget{orig: created}
	c.Assert(buf.String(), qt.JSONEquals, res)
}
