package webhook

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <database>",
		Short: "List webhooks for a database",
		Args:  cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching webhooks for %s", printer.BoldBlue(database)))
			defer end()

			webhooks, err := client.Webhooks.List(ctx, &planetscale.ListWebhooksRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if len(webhooks) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No webhooks exist in database %s.\n", printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(toWebhooks(webhooks))
		},
	}

	return cmd
}
