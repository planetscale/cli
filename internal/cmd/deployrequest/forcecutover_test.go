package deployrequest

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestDeployRequest_ForceCutoverCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	var number uint64 = 10
	requestedAt := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	svc := &mock.DeployRequestsService{
		ForceCutoverFn: func(ctx context.Context, req *ps.ForceCutoverDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Number, qt.Equals, number)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)

			return &ps.DeployRequest{
				Number: number,
				Deployment: &ps.Deployment{
					State:                   "in_progress_cutover",
					ForceCutoverRequestedAt: &requestedAt,
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

	cmd := ForceCutoverCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ForceCutoverFnInvoked, qt.IsTrue)

	res := &ps.DeployRequest{
		Number: number,
		Deployment: &ps.Deployment{
			State:                   "in_progress_cutover",
			ForceCutoverRequestedAt: &requestedAt,
		},
	}
	c.Assert(buf.String(), qt.JSONEquals, res)
}
