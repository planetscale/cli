package webhook

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		url                            string
		webhookAuthorizationToken      string
		clearWebhookAuthorizationToken bool
		events                         []string
		enabled                        bool
	}

	cmd := &cobra.Command{
		Use:   "update <database> <webhook-id>",
		Short: "Update a webhook for a database",
		Args:  cmdutil.RequiredArgs("database", "webhook-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			webhookID := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			req := &planetscale.UpdateWebhookRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				ID:           webhookID,
			}

			changed := false

			if cmd.Flags().Changed("url") {
				req.URL = &flags.url
				changed = true
			}

			if cmd.Flags().Changed("webhook-authorization-token") {
				if flags.webhookAuthorizationToken == "" {
					return fmt.Errorf("--webhook-authorization-token cannot be empty; use --clear-webhook-authorization-token to remove the configured token")
				}
				req.WebhookAuthorizationToken = &flags.webhookAuthorizationToken
				changed = true
			}

			if flags.clearWebhookAuthorizationToken {
				req.ClearWebhookAuthorizationToken = true
				changed = true
			}

			if cmd.Flags().Changed("events") {
				req.Events = flags.events
				changed = true
			}

			if cmd.Flags().Changed("enabled") {
				req.Enabled = &flags.enabled
				changed = true
			}

			if !changed {
				return fmt.Errorf("at least one of --url, --webhook-authorization-token, --clear-webhook-authorization-token, --events, or --enabled must be provided")
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating webhook %s for %s", printer.BoldBlue(webhookID), printer.BoldBlue(database)))
			defer end()

			webhook, err := client.Webhooks.Update(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("webhook %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(webhookID), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			return ch.Printer.PrintResource(toWebhook(webhook))
		},
	}

	cmd.Flags().StringVar(&flags.url, "url", "", "The URL to send webhook events to")
	cmd.Flags().StringVar(&flags.webhookAuthorizationToken, "webhook-authorization-token", "", "Bearer token to include in the Authorization header")
	cmd.Flags().BoolVar(&flags.clearWebhookAuthorizationToken, "clear-webhook-authorization-token", false, "Remove the configured webhook authorization token")
	cmd.Flags().StringSliceVar(&flags.events, "events", nil, "Comma-separated list of events to subscribe to")
	cmd.Flags().BoolVar(&flags.enabled, "enabled", true, "Whether the webhook is enabled")
	cmd.MarkFlagsMutuallyExclusive("webhook-authorization-token", "clear-webhook-authorization-token")

	return cmd
}
