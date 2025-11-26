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

func TestWebhook_CreateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	url := "https://example.com/webhook"
	events := []string{"branch.created", "branch.deleted"}
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	webhook := &ps.Webhook{
		ID:        "webhook-123",
		URL:       url,
		Enabled:   true,
		Events:    events,
		CreatedAt: createdAt,
	}

	svc := &mock.WebhooksService{
		CreateFn: func(ctx context.Context, req *ps.CreateWebhookRequest) (*ps.Webhook, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.URL, qt.Equals, url)
			c.Assert(req.Events, qt.DeepEquals, events)
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

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, "--url", url, "--events", "branch.created,branch.deleted"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)

	res := &Webhook{orig: webhook}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestWebhook_CreateCmd_RequiresURL(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)

	org := "planetscale"
	db := "mydb"

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "required flag")
}
