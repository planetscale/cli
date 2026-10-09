package deployrequest

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestDeployRequest_RetryCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	var number uint64 = 10

	svc := &mock.DeployRequestsService{
		RetryFailedOperationsFn: func(ctx context.Context, req *ps.RetryFailedOperationsRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Number, qt.Equals, number)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)

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

	cmd := RetryCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10)})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.RetryFailedOperationsFnInvoked, qt.IsTrue)

	res := &ps.DeployRequest{Number: number}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDeployRequest_RetryCmdSurfacesAPIError(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.DeployRequestsService{
		RetryFailedOperationsFn: func(ctx context.Context, req *ps.RetryFailedOperationsRequest) (*ps.DeployRequest, error) {
			return nil, errors.New("A retry was already requested.")
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := RetryCmd(ch)
	cmd.SetArgs([]string{"planetscale", "10"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "A retry was already requested.")
	c.Assert(svc.RetryFailedOperationsFnInvoked, qt.IsTrue)
}
