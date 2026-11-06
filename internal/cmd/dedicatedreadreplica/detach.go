package dedicatedreadreplica

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmd/branch"
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// DetachCmd detaches a dedicated read replica from its branch and promotes it
// to a standalone branch of the database.
func DetachCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "detach <database> <branch> <name>",
		Short: "Detach a dedicated read replica into a standalone branch",
		Long: `Detach a dedicated read replica from its branch and promote it to a
standalone branch of the database.

The new branch keeps the replica's name, region, cluster size, replica count,
storage configuration, and parameters, and receives a copy of the source
branch's roles. The dedicated read replica no longer exists once it is
detached. This cannot be undone.

The replica must be ready with no pending changes, and any dedicated
PgBouncers attached to it must be deleted first.

This command is only available for PostgreSQL databases.`,
		Args: cmdutil.RequiredArgs("database", "branch", "name"),
		// Detachment is behind a feature flag on the API. Keep the command out
		// of help output until it is generally available.
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, sourceBranch, name := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}
			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "dedicated read replicas"); err != nil {
				return err
			}

			if !force {
				confirmationName := fmt.Sprintf("%s/%s/%s", database, sourceBranch, name)
				if err := ch.Printer.ConfirmCommand(confirmationName, "detach dedicated read replica", "detachment of dedicated read replica"); err != nil {
					return err
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Detaching dedicated read replica %s from %s/%s", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(sourceBranch)))
			defer end()

			detached, err := client.PostgresDedicatedReadReplicas.Detach(ctx, &ps.DetachPostgresDedicatedReadReplicaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       sourceBranch,
				Replica:      name,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("dedicated read replica %s does not exist on %s/%s, or detaching is not available for this branch (organization: %s)",
						printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(sourceBranch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Dedicated read replica %s was successfully detached from %s/%s and is now branch %s.\n",
					printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(sourceBranch), printer.BoldBlue(detached.Name))
				if !detached.Ready {
					ch.Printer.Printf("The new branch is still being promoted. Check it with %s.\n",
						printer.BoldBlue(fmt.Sprintf("pscale branch show %s %s", database, detached.Name)))
				}
				return nil
			}

			return ch.Printer.PrintResource(branch.ToPostgresBranch(detached))
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Detach a dedicated read replica without confirmation")
	return cmd
}
