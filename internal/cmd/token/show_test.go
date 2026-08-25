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
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestServiceToken_ShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	id := "token-id"
	name := "deploy-token"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	lastUsedAt := time.Date(2025, 1, 20, 14, 45, 0, 0, time.UTC)
	token := &ps.ServiceToken{
		ID:         id,
		Name:       &name,
		Token:      "must-not-be-printed",
		CreatedAt:  createdAt,
		LastUsedAt: &lastUsedAt,
	}

	svc := &mock.ServiceTokenService{
		GetFn: func(ctx context.Context, req *ps.GetServiceTokenRequest) (*ps.ServiceToken, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.ID, qt.Equals, id)
			return token, nil
		},
	}

	ch := serviceTokenShowHelper(p, org, svc)
	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{id})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]any{
		"id":           id,
		"name":         name,
		"created_at":   "2025-01-15T10:30:00Z",
		"last_used_at": "2025-01-20T14:45:00Z",
		"expires_at":   nil,
	})
	c.Assert(buf.String(), qt.Not(qt.Contains), token.Token)
}

func TestServiceToken_ShowCmdHuman(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)
	p.SetResourceOutput(&buf)

	name := "deploy-token"
	svc := &mock.ServiceTokenService{
		GetFn: func(ctx context.Context, req *ps.GetServiceTokenRequest) (*ps.ServiceToken, error) {
			return &ps.ServiceToken{
				ID:        "token-id",
				Name:      &name,
				Token:     "must-not-be-printed",
				CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			}, nil
		},
	}

	cmd := ShowCmd(serviceTokenShowHelper(p, "planetscale", svc))
	cmd.SetArgs([]string{"token-id"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "token-id")
	c.Assert(buf.String(), qt.Contains, "deploy-token")
	c.Assert(buf.String(), qt.Not(qt.Contains), "must-not-be-printed")
}

func TestServiceToken_ShowCmdNotFound(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.ServiceTokenService{
		GetFn: func(ctx context.Context, req *ps.GetServiceTokenRequest) (*ps.ServiceToken, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	cmd := ShowCmd(serviceTokenShowHelper(p, "planetscale", svc))
	cmd.SetArgs([]string{"missing-token"})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "service token missing-token does not exist in organization planetscale")
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
}

func serviceTokenShowHelper(p *printer.Printer, org string, svc *mock.ServiceTokenService) *cmdutil.Helper {
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{ServiceTokens: svc}, nil
		},
	}
}
