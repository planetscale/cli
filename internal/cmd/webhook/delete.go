package webhook

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func DeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <database> <webhook-id>",
		Short: "Delete a webhook for a database",
		Args:  cmdutil.RequiredArgs("database", "webhook-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			webhookID := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting webhook %s for %s", printer.BoldBlue(webhookID), printer.BoldBlue(database)))
			defer end()

			err = client.Webhooks.Delete(ctx, &planetscale.DeleteWebhookRequest{
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

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Webhook %s was successfully deleted from %s.\n", printer.BoldBlue(webhookID), printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result": "webhook deleted",
			})
		},
	}

	return cmd
}
