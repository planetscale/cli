package webhook

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		url     string
		events  []string
		enabled bool
	}

	cmd := &cobra.Command{
		Use:   "create <database>",
		Short: "Create a webhook for a database",
		Args:  cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			req := &planetscale.CreateWebhookRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				URL:          flags.url,
				Events:       flags.events,
			}

			if cmd.Flags().Changed("enabled") {
				req.Enabled = &flags.enabled
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating webhook for %s", printer.BoldBlue(database)))
			defer end()

			webhook, err := client.Webhooks.Create(ctx, req)
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

			return ch.Printer.PrintResource(toWebhookWithSecret(webhook))
		},
	}

	cmd.Flags().StringVar(&flags.url, "url", "", "The URL to send webhook events to (required)")
	cmd.Flags().StringSliceVar(&flags.events, "events", nil, "Comma-separated list of events to subscribe to")
	cmd.Flags().BoolVar(&flags.enabled, "enabled", true, "Whether the webhook is enabled")

	cmd.MarkFlagRequired("url") // nolint:errcheck

	return cmd
}
