package readonlyreplica

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// DeleteCmd deletes a dedicated read replica by name.
func DeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database> <branch> <name>",
		Short:   "Delete a dedicated read replica",
		Args:    cmdutil.RequiredArgs("database", "branch", "name"),
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, name := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}
			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "dedicated read replicas"); err != nil {
				return err
			}

			if !force {
				confirmationName := fmt.Sprintf("%s/%s/%s", database, branch, name)
				if err := ch.Printer.ConfirmCommand(confirmationName, "delete dedicated read replica", "deletion of dedicated read replica"); err != nil {
					return err
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting dedicated read replica %s from %s/%s", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			err = client.PostgresReadOnlyReplicas.Delete(ctx, &ps.DeletePostgresReadOnlyReplicaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Replica:      name,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("dedicated read replica %s does not exist on %s/%s (organization: %s)",
						printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Dedicated read replica %s was successfully deleted from %s/%s.\n",
					printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result":   "dedicated read replica deleted",
				"name":     name,
				"database": database,
				"branch":   branch,
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete a dedicated read replica without confirmation")
	return cmd
}
