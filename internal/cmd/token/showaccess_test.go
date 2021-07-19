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
	db := "planetscale"
	token := "123456"
	accesses := []string{"read_branch", "delete_branch"}

	orig := []*ps.ServiceTokenAccess{
		{
			ID:       "id-1",
			Access:   "read_branch",
			Resource: ps.Database{Name: db},
		},
		{
			ID:       "id-2",
			Access:   "delete_branch",
			Resource: ps.Database{Name: db},
		},
	}

	svc := &mock.ServiceTokenService{
		GetAccessFn: func(ctx context.Context, req *ps.GetServiceTokenAccessRequest) ([]*ps.ServiceTokenAccess, error) {
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
	c.Assert(svc.GetAccessFnInvoked, qt.IsTrue)

	res := []*ServiceTokenAccess{
		{
			Database: db,
			Accesses: accesses,
		},
	}
	c.Assert(buf.String(), qt.JSONEquals, res)
}
