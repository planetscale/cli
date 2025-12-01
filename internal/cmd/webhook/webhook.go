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
		Short:             "List webhooks",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ListCmd(ch))

	return cmd
}

// Webhook returns a table and json serializable webhook for printing.
type Webhook struct {
	ID        string `header:"id" json:"id"`
	URL       string `header:"url" json:"url"`
	Events    string `header:"events" json:"events"`
	Enabled   bool   `header:"enabled" json:"enabled"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`

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
