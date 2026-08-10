package pgbouncer

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// DeleteCmd deletes a dedicated PgBouncer by name.
func DeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database> <branch> <name>",
		Short:   "Delete a dedicated PgBouncer",
		Args:    cmdutil.RequiredArgs("database", "branch", "name"),
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			name := args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "PgBouncers"); err != nil {
				return err
			}

			if !force {
				if err := ch.Printer.ConfirmCommand(name, "delete PgBouncer", "deletion of PgBouncer"); err != nil {
					return err
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting PgBouncer %s from %s/%s", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			err = client.PostgresBouncers.Delete(ctx, &ps.DeletePostgresBouncerRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Bouncer:      name,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("PgBouncer %s does not exist on %s/%s (organization: %s)",
						printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("PgBouncer %s was successfully deleted from %s/%s.\n",
					printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result":   "PgBouncer deleted",
				"name":     name,
				"database": database,
				"branch":   branch,
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete a PgBouncer without confirmation")
	return cmd
}
