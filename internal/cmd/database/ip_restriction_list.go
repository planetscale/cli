package database

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// IPRestrictionListCmd lists IP restriction entries for a Postgres database.
func IPRestrictionListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <database>",
		Short: "List IP restriction entries for a Postgres database",
		Args:  cmdutil.RequiredArgs("database"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := requirePostgresDatabase(ctx, ch, client, database); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching IP restriction entries for %s", printer.BoldBlue(database)))
			defer end()

			entries, err := client.PostgresCIDRs.List(ctx, &ps.ListPostgresCIDRsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(entries) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No IP restriction entries exist for database %s.\n", printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(toIPRestrictionEntries(entries))
		},
	}

	return cmd
}
