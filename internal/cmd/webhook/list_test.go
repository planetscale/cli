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

func TestWebhook_ListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	webhooks := []*ps.Webhook{
		{
			ID:        "webhook-123",
			URL:       "https://example.com/webhook",
			Enabled:   true,
			Events:    []string{"branch.created", "branch.deleted"},
			CreatedAt: createdAt,
		},
	}

	svc := &mock.WebhooksService{
		ListFn: func(ctx context.Context, req *ps.ListWebhooksRequest, opts ...ps.ListOption) ([]*ps.Webhook, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return webhooks, nil
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

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{db})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)

	res := []*Webhook{
		{orig: webhooks[0]},
	}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestWebhook_ListCmd_Empty(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)

	org := "planetscale"
	db := "mydb"

	svc := &mock.WebhooksService{
		ListFn: func(ctx context.Context, req *ps.ListWebhooksRequest, opts ...ps.ListOption) ([]*ps.Webhook, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return []*ps.Webhook{}, nil
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

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{db})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "No webhooks exist")
}
