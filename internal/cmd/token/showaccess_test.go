package token

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

func TestServiceToken_ShowAccess(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	token := "123456"

	orig := []*ps.ServiceTokenGrant{
		{
			ID:           "id-1",
			Accesses:     []string{"read_branch"},
			ResourceName: "db1",
		},
		{
			ID:           "id-2",
			Accesses:     []string{"delete_branch"},
			ResourceName: "db2",
		},
	}

	svc := &mock.ServiceTokenService{
		ListGrantsFn: func(ctx context.Context, req *ps.ListServiceTokenGrantsRequest) ([]*ps.ServiceTokenGrant, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.ID, qt.Equals, token)

			return orig, nil
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

	cmd := ShowAccessCmd(ch)
	cmd.SetArgs(args)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListGrantsFnInvoked, qt.IsTrue)

	res := []*ServiceTokenGrant{
		{
			Database: "db1",
			Accesses: []string{"read_branch"},
		},
		{
			Database: "db2",
			Accesses: []string{"delete_branch"},
		},
	}
	c.Assert(buf.String(), qt.JSONEquals, res)
}
