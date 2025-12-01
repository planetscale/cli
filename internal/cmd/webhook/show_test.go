package webhook

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

func TestWebhook_ShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	webhookID := "webhook-123"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	webhook := &ps.Webhook{
		ID:        webhookID,
		URL:       "https://example.com/webhook",
		Secret:    "abcdefgh",
		Enabled:   true,
		Events:    []string{"branch.created", "branch.deleted"},
		CreatedAt: createdAt,
	}

	svc := &mock.WebhooksService{
		GetFn: func(ctx context.Context, req *ps.GetWebhookRequest) (*ps.Webhook, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.ID, qt.Equals, webhookID)
			return webhook, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Webhooks: svc,
			}, nil
		},
	}

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, webhookID})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)

	res := &WebhookWithSecret{orig: webhook}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestWebhook_ShowCmd_NotFound(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)

	org := "planetscale"
	db := "mydb"
	webhookID := "webhook-123"

	svc := &mock.WebhooksService{
		GetFn: func(ctx context.Context, req *ps.GetWebhookRequest) (*ps.Webhook, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Webhooks: svc,
			}, nil
		},
	}

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, webhookID})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
}
