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

func TestDeployRequest_UnblockCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	var number uint64 = 10

	svc := &mock.DeployRequestsService{
		GetFn: func(ctx context.Context, req *ps.GetDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Number, qt.Equals, number)
			return &ps.DeployRequest{
				Number:     number,
				Deployment: &ps.Deployment{State: "complete_error"},
			}, nil
		},
		UnblockFn: func(ctx context.Context, req *ps.UnblockDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Number, qt.Equals, number)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			return &ps.DeployRequest{Number: number}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := UnblockCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10)})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.UnblockFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &ps.DeployRequest{Number: number})
}

func TestDeployRequest_UnblockCmd_RevertError(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.DeployRequestsService{
		GetFn: func(ctx context.Context, req *ps.GetDeployRequestRequest) (*ps.DeployRequest, error) {
			return &ps.DeployRequest{
				Number:     10,
				Deployment: &ps.Deployment{State: "complete_revert_error"},
			}, nil
		},
		UnblockFn: func(ctx context.Context, req *ps.UnblockDeployRequestRequest) (*ps.DeployRequest, error) {
			return &ps.DeployRequest{Number: 10}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := UnblockCmd(ch)
	cmd.SetArgs([]string{"planetscale", "10"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UnblockFnInvoked, qt.IsTrue)
}

func TestDeployRequest_UnblockCmd_PendingCutover(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.DeployRequestsService{
		GetFn: func(ctx context.Context, req *ps.GetDeployRequestRequest) (*ps.DeployRequest, error) {
			return &ps.DeployRequest{
				Number:     10,
				Deployment: &ps.Deployment{State: "pending_cutover"},
			}, nil
		},
		UnblockFn: func(ctx context.Context, req *ps.UnblockDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Fatal("UnblockDeploy should not be called")
			return nil, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := UnblockCmd(ch)
	cmd.SetArgs([]string{"planetscale", "10"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, ".*waiting to apply changes.*deploy-request apply.*")
	c.Assert(svc.UnblockFnInvoked, qt.IsFalse)
}

func TestDeployRequest_UnblockCmd_DeployCheckError(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.DeployRequestsService{
		GetFn: func(ctx context.Context, req *ps.GetDeployRequestRequest) (*ps.DeployRequest, error) {
			return &ps.DeployRequest{
				Number: 10,
				Deployment: &ps.Deployment{
					State:             "error",
					DeployCheckErrors: "incompatible unique index",
				},
			}, nil
		},
		UnblockFn: func(ctx context.Context, req *ps.UnblockDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Fatal("UnblockDeploy should not be called")
			return nil, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := UnblockCmd(ch)
	cmd.SetArgs([]string{"planetscale", "10"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, ".*failed deploy checks.*incompatible unique index")
	c.Assert(svc.UnblockFnInvoked, qt.IsFalse)
}
