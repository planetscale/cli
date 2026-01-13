package token

import (
	"bytes"
	"context"
	"testing"
	"time"

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
	name1 := "token-one"
	createdAt1 := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	lastUsedAt1 := time.Date(2025, 1, 20, 14, 45, 0, 0, time.UTC)
	createdAt2 := time.Date(2025, 1, 16, 11, 0, 0, 0, time.UTC)

	orig := []*ps.ServiceToken{
		{ID: "1", Name: &name1, CreatedAt: createdAt1, LastUsedAt: &lastUsedAt1},
		{ID: "2", CreatedAt: createdAt2},
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

func TestServiceToken_ListCmdWithExpiresAt(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	name1 := "token-one"
	createdAt1 := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	lastUsedAt1 := time.Date(2025, 1, 20, 14, 45, 0, 0, time.UTC)
	expiresAt1 := time.Date(2025, 2, 15, 10, 30, 0, 0, time.UTC)
	createdAt2 := time.Date(2025, 1, 16, 11, 0, 0, 0, time.UTC)
	expiresAt2 := time.Date(2025, 2, 16, 11, 0, 0, 0, time.UTC)

	orig := []*ps.ServiceToken{
		{ID: "1", Name: &name1, CreatedAt: createdAt1, LastUsedAt: &lastUsedAt1, ExpiresAt: &expiresAt1},
		{ID: "2", CreatedAt: createdAt2, ExpiresAt: &expiresAt2},
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
