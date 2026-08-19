package deployrequest

import (
	"bytes"
	"context"
	"strconv"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestDeployRequest_DeployCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	var number uint64 = 10

	svc := &mock.DeployRequestsService{
		DeployFn: func(ctx context.Context, req *ps.PerformDeployRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Number, qt.Equals, number)

			return &ps.DeployRequest{Number: number}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DeployRequests: svc,
			}, nil
		},
	}

	cmd := DeployCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10)})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DeployFnInvoked, qt.IsTrue)

	res := &ps.DeployRequest{Number: number}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDeployRequest_DeployCmdWithWaitPrintsFinalDeployRequest(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	number := uint64(10)
	pending := &ps.DeployRequest{
		Number:     number,
		Deployment: &ps.Deployment{State: "pending"},
	}
	complete := &ps.DeployRequest{
		Number:     number,
		Deployment: &ps.Deployment{State: "complete"},
	}

	svc := &mock.DeployRequestsService{
		DeployFn: func(_ context.Context, _ *ps.PerformDeployRequest) (*ps.DeployRequest, error) {
			return pending, nil
		},
		GetFn: func(_ context.Context, req *ps.GetDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Assert(req, qt.DeepEquals, &ps.GetDeployRequestRequest{
				Organization: org,
				Database:     db,
				Number:       number,
			})
			return complete, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}
	debug := false
	ch.SetDebug(&debug)

	cmd := DeployCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--wait"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, complete)
}

func TestDeployRequest_DeployBranchName(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	number := uint64(10)
	branchName := "dev"

	svc := &mock.DeployRequestsService{
		DeployFn: func(ctx context.Context, req *ps.PerformDeployRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Number, qt.Equals, number)

			return &ps.DeployRequest{Number: number}, nil
		},
		ListFn: func(ctx context.Context, req *ps.ListDeployRequestsRequest) ([]*ps.DeployRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branchName)

			return []*ps.DeployRequest{
				{
					Number: number,
				},
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DeployRequests: svc,
			}, nil
		},
	}

	cmd := DeployCmd(ch)
	cmd.SetArgs([]string{db, branchName})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DeployFnInvoked, qt.IsTrue)

	res := &ps.DeployRequest{Number: number}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDeployRequest_DeployStrategyFlag(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	var number uint64 = 10

	svc := &mock.DeployRequestsService{
		DeployFn: func(ctx context.Context, req *ps.PerformDeployRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Strategy, qt.Equals, "parallel")
			return &ps.DeployRequest{Number: number}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DeployRequests: svc,
			}, nil
		},
	}

	cmd := DeployCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--strategy", "parallel"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DeployFnInvoked, qt.IsTrue)
}

func TestDeployRequest_DeployStrategyFlagInvalid(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	var number uint64 = 10

	svc := &mock.DeployRequestsService{
		DeployFn: func(ctx context.Context, req *ps.PerformDeployRequest) (*ps.DeployRequest, error) {
			t.Fatal("Deploy should not be called when --strategy is invalid")
			return nil, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DeployRequests: svc,
			}, nil
		},
	}

	cmd := DeployCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--strategy", "nope"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `invalid --strategy "nope".*`)
	c.Assert(svc.DeployFnInvoked, qt.IsFalse)
}
