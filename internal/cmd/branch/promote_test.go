package branch

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

func TestBranch_PromoteCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"

	res := &ps.DatabaseBranch{
		Name: branch,
	}

	svc := &mock.DatabaseBranchesService{
		PromoteFn: func(ctx context.Context, req *ps.PromoteRequest) (*ps.BranchPromotionRequest, error) {
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)

			return &ps.BranchPromotionRequest{
				Branch: branch,
				State:  "pending",
			}, nil
		},
		GetPromotionRequestFn: func(ctx context.Context, req *ps.GetPromotionRequestRequest) (*ps.BranchPromotionRequest, error) {
			return &ps.BranchPromotionRequest{
				Branch: branch,
				State:  "promoted",
			}, nil
		},
		GetFn: func(ctx context.Context, req *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)

			return res, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DatabaseBranches: svc,
			}, nil

		},
	}

	cmd := PromoteCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.PromoteFnInvoked, qt.IsTrue)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.GetPromotionRequestFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}
