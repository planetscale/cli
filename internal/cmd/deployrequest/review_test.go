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
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestDeployRequest_ReviewCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	var number uint64 = 10
	action := ps.ReviewComment
	comment := "this is a comment"

	res := &ps.DeployRequestReview{
		Body: "foo",
	}

	svc := &mock.DeployRequestsService{
		CreateReviewFn: func(ctx context.Context, req *ps.ReviewDeployRequestRequest) (*ps.DeployRequestReview, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Number, qt.Equals, number)
			c.Assert(req.ReviewAction, qt.Equals, action)
			c.Assert(req.CommentText, qt.Equals, comment)

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
				DeployRequests: svc,
			}, nil

		},
	}

	cmd := ReviewCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--comment", comment})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateReviewFnInvoked, qt.IsTrue)

	c.Assert(buf.String(), qt.JSONEquals, res)
}
