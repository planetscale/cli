package database

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// IPRestrictionShowCmd shows a single IP restriction entry.
func IPRestrictionShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <entry-id>",
		Short: "Show an IP restriction entry",
		Args:  cmdutil.RequiredArgs("database", "entry-id"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			entryID := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := requirePostgresDatabase(ctx, ch, client, database); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching IP restriction entry %s for %s", printer.BoldBlue(entryID), printer.BoldBlue(database)))
			defer end()

			entry, err := client.PostgresCIDRs.Get(ctx, &ps.GetPostgresCIDRRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				ID:           entryID,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("IP restriction entry %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(entryID), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toIPRestrictionEntry(entry))
		},
	}

	return cmd
}
