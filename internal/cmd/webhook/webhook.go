package webhook

import (
	"encoding/json"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// WebhookCmd encapsulates the command for managing webhooks.
func WebhookCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "webhook <command>",
		Short:             "Create, list, and manage webhooks",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(DeleteCmd(ch))
	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(TestCmd(ch))
	cmd.AddCommand(UpdateCmd(ch))

	return cmd
}

// Webhook returns a table and json serializable webhook for printing.
type Webhook struct {
	ID        string `header:"id" json:"id"`
	URL       string `header:"url" json:"url"`
	Events    string `header:"events" json:"events"`
	Enabled   bool   `header:"enabled" json:"enabled"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.Webhook
}

func (w *Webhook) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(w.orig, "", "  ")
}

// toWebhook returns a struct that prints out the various fields of a webhook model.
func toWebhook(webhook *ps.Webhook) *Webhook {
	return &Webhook{
		ID:        webhook.ID,
		URL:       webhook.URL,
		Events:    strings.Join(webhook.Events, ", "),
		Enabled:   webhook.Enabled,
		CreatedAt: printer.GetMilliseconds(webhook.CreatedAt),
		UpdatedAt: printer.GetMilliseconds(webhook.UpdatedAt),
		orig:      webhook,
	}
}

func toWebhooks(webhooks []*ps.Webhook) []*Webhook {
	results := make([]*Webhook, 0, len(webhooks))
	for _, webhook := range webhooks {
		results = append(results, toWebhook(webhook))
	}
	return results
}

// WebhookWithSecret includes the webhook secret for display.
type WebhookWithSecret struct {
	ID        string `header:"id" json:"id"`
	URL       string `header:"url" json:"url"`
	Secret    string `header:"secret" json:"secret"`
	Events    string `header:"events" json:"events"`
	Enabled   bool   `header:"enabled" json:"enabled"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.Webhook
}

func (w *WebhookWithSecret) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(w.orig, "", "  ")
}

// toWebhookWithSecret returns a struct that includes the webhook secret.
func toWebhookWithSecret(webhook *ps.Webhook) *WebhookWithSecret {
	return &WebhookWithSecret{
		ID:        webhook.ID,
		URL:       webhook.URL,
		Secret:    webhook.Secret,
		Events:    strings.Join(webhook.Events, ", "),
		Enabled:   webhook.Enabled,
		CreatedAt: printer.GetMilliseconds(webhook.CreatedAt),
		UpdatedAt: printer.GetMilliseconds(webhook.UpdatedAt),
		orig:      webhook,
	}
}
