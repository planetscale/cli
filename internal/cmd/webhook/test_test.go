package webhook

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

func TestWebhook_TestCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	webhookID := "webhook-123"

	svc := &mock.WebhooksService{
		TestFn: func(ctx context.Context, req *ps.TestWebhookRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.ID, qt.Equals, webhookID)
			return nil
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

	cmd := TestCmd(ch)
	cmd.SetArgs([]string{db, webhookID})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.TestFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]string{"result": "test event sent"})
}

func TestWebhook_TestCmd_Human(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)

	org := "planetscale"
	db := "mydb"
	webhookID := "webhook-123"

	svc := &mock.WebhooksService{
		TestFn: func(ctx context.Context, req *ps.TestWebhookRequest) error {
			return nil
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

	cmd := TestCmd(ch)
	cmd.SetArgs([]string{db, webhookID})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.TestFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "successfully sent")
}

func TestWebhook_TestCmd_NotFound(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)

	org := "planetscale"
	db := "mydb"
	webhookID := "webhook-123"

	svc := &mock.WebhooksService{
		TestFn: func(ctx context.Context, req *ps.TestWebhookRequest) error {
			return &ps.Error{Code: ps.ErrNotFound}
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

	cmd := TestCmd(ch)
	cmd.SetArgs([]string{db, webhookID})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
}
