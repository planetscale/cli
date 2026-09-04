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
		url                      string
		authorizationHeader      string
		clearAuthorizationHeader bool
		events                   []string
		enabled                  bool
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

			if cmd.Flags().Changed("authorization-header") {
				if flags.authorizationHeader == "" {
					return fmt.Errorf("--authorization-header cannot be empty; use --clear-authorization-header to remove the configured header")
				}
				req.AuthorizationHeader = &flags.authorizationHeader
				changed = true
			}

			if flags.clearAuthorizationHeader {
				empty := ""
				req.AuthorizationHeader = &empty
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
				return fmt.Errorf("at least one of --url, --authorization-header, --clear-authorization-header, --events, or --enabled must be provided")
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
	cmd.Flags().StringVar(&flags.authorizationHeader, "authorization-header", "", "The complete Authorization header value, for example Bearer token")
	cmd.Flags().BoolVar(&flags.clearAuthorizationHeader, "clear-authorization-header", false, "Remove the configured Authorization header")
	cmd.Flags().StringSliceVar(&flags.events, "events", nil, "Comma-separated list of events to subscribe to")
	cmd.Flags().BoolVar(&flags.enabled, "enabled", true, "Whether the webhook is enabled")
	cmd.MarkFlagsMutuallyExclusive("authorization-header", "clear-authorization-header")

	return cmd
}
