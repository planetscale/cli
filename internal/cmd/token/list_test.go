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

func TestServiceToken_ListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"

	orig := []*ps.ServiceToken{
		{ID: "1"},
		{ID: "2"},
	}

	svc := &mock.ServiceTokenService{
		ListFn: func(ctx context.Context, req *ps.ListServiceTokensRequest) ([]*ps.ServiceToken, error) {
			c.Assert(req.Organization, qt.Equals, org)
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

	cmd := ListCmd(ch)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)

	res := []*ServiceToken{
		{orig: orig[0]},
		{orig: orig[1]},
	}
	c.Assert(buf.String(), qt.JSONEquals, res)
}
