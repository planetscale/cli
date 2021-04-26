package token

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestServiceToken_DeleteAccessCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	token := "123456"
	db := "planetscale"
	accesses := []string{"read_branch", "delete_branch"}

	svc := &mock.ServiceTokenService{
		DeleteAccessFn: func(ctx context.Context, req *ps.DeleteServiceTokenAccessRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.ID, qt.Equals, token)
			c.Assert(req.ID, qt.Equals, token)
			c.Assert(req.Accesses, qt.DeepEquals, accesses)
			return nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				ServiceTokens: svc,
			}, nil

		},
	}

	args := []string{token}
	args = append(args, accesses...)
	args = append(args, "--database", db)

	cmd := DeleteAccessCmd(ch)
	cmd.SetArgs(args)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DeleteAccessFnInvoked, qt.IsTrue)

	res := map[string]string{
		"result": "accesses deleted",
		"perms":  "read_branch,delete_branch",
	}
	c.Assert(buf.String(), qt.JSONEquals, res)
}
