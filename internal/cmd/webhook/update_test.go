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

func TestWebhook_UpdateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	webhookID := "webhook-123"
	newURL := "https://example.com/new-webhook"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	webhook := &ps.Webhook{
		ID:        webhookID,
		URL:       newURL,
		Enabled:   true,
		Events:    []string{"branch.created"},
		CreatedAt: createdAt,
	}

	svc := &mock.WebhooksService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateWebhookRequest) (*ps.Webhook, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.ID, qt.Equals, webhookID)
			c.Assert(*req.URL, qt.Equals, newURL)
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

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{db, webhookID, "--url", newURL})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)

	res := &Webhook{orig: webhook}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestWebhook_UpdateCmd_EnabledFlag(t *testing.T) {
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
		Enabled:   false,
		Events:    []string{"branch.created"},
		CreatedAt: createdAt,
	}

	svc := &mock.WebhooksService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateWebhookRequest) (*ps.Webhook, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.ID, qt.Equals, webhookID)
			c.Assert(*req.Enabled, qt.IsFalse)
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

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{db, webhookID, "--enabled=false"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestWebhook_UpdateCmd_RequiresAtLeastOneFlag(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)

	org := "planetscale"
	db := "mydb"
	webhookID := "webhook-123"

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{db, webhookID})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "at least one of")
}
