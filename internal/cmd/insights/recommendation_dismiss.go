package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// RecommendationDismissCmd dismisses a schema recommendation.
func RecommendationDismissCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		reason string
		force  bool
	}

	cmd := &cobra.Command{
		Use:   "dismiss <database> <number>",
		Short: "Dismiss a schema recommendation",
		Long: `Dismiss a schema recommendation so it no longer appears as open.

<number> is the recommendation sequence number from
'pscale insights recommendations <database>'.`,
		Args: cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			number := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !flags.force {
				if err := ch.Printer.ConfirmCommand(number, "dismiss schema recommendation", "dismissal of schema recommendation"); err != nil {
					return err
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Dismissing schema recommendation %s for %s...",
				printer.BoldBlue(number), printer.BoldBlue(database)))
			defer end()

			recommendation, err := client.SchemaRecommendations.Dismiss(ctx, &ps.DismissSchemaRecommendationRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				ID:           number,
				Reason:       flags.reason,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("schema recommendation %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(number), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(recommendation)
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Schema recommendation %s was dismissed from %s.\n",
					printer.BoldBlue(number), printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result": "schema recommendation dismissed",
				"number": number,
			})
		},
	}

	cmd.Flags().StringVar(&flags.reason, "reason", "", "Optional reason for dismissing the recommendation")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Dismiss without confirmation")

	return cmd
}
