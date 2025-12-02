package webhook

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <webhook-id>",
		Short: "Show a webhook for a database",
		Args:  cmdutil.RequiredArgs("database", "webhook-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			webhookID := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching webhook %s for %s", printer.BoldBlue(webhookID), printer.BoldBlue(database)))
			defer end()

			webhook, err := client.Webhooks.Get(ctx, &planetscale.GetWebhookRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				ID:           webhookID,
			})
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

			return ch.Printer.PrintResource(toWebhookWithSecret(webhook))
		},
	}

	return cmd
}
