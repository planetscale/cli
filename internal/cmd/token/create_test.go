package token

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestServiceToken_CreateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	id := "123456"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	orig := &ps.ServiceToken{ID: id, CreatedAt: createdAt}

	svc := &mock.ServiceTokenService{
		CreateFn: func(ctx context.Context, req *ps.CreateServiceTokenRequest) (*ps.ServiceToken, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Name, qt.IsNil)
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

	cmd := CreateCmd(ch)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)

	res := &ServiceTokenWithSecret{orig: orig}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestServiceToken_CreateCmdWithName(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	id := "123456"
	name := "my-token"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	orig := &ps.ServiceToken{ID: id, Name: &name, CreatedAt: createdAt}

	svc := &mock.ServiceTokenService{
		CreateFn: func(ctx context.Context, req *ps.CreateServiceTokenRequest) (*ps.ServiceToken, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Name, qt.IsNotNil)
			c.Assert(*req.Name, qt.Equals, name)
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

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{"--name", name})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)

	res := &ServiceTokenWithSecret{orig: orig}
	c.Assert(buf.String(), qt.JSONEquals, res)
}
