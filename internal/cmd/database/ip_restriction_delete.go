package database

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// IPRestrictionDeleteCmd deletes an IP restriction entry.
func IPRestrictionDeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database> <entry-id>",
		Short:   "Delete an IP restriction entry",
		Long:    "Delete an IP restriction entry from a PostgreSQL database. Deleting an entry removes the restriction from all database branches.",
		Args:    cmdutil.RequiredArgs("database", "entry-id"),
		Aliases: []string{"rm"},
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

			if !force {
				if err := ch.Printer.ConfirmCommand(entryID, "delete IP restriction entry", "deletion of IP restriction entry"); err != nil {
					return err
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting IP restriction entry %s from %s", printer.BoldBlue(entryID), printer.BoldBlue(database)))
			defer end()

			err = client.PostgresCIDRs.Delete(ctx, &ps.DeletePostgresCIDRRequest{
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

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("IP restriction entry %s was successfully deleted from %s.\n",
					printer.BoldBlue(entryID), printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result": "IP restriction entry deleted",
				"id":     entryID,
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete an IP restriction entry without confirmation")
	return cmd
}
