package dedicatedreadreplica

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// ListCmd lists dedicated read replicas for a Postgres branch.
func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List dedicated read replicas for a Postgres branch",
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}
			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "dedicated read replicas"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching dedicated read replicas for %s/%s", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			replicas, err := client.PostgresDedicatedReadReplicas.List(ctx, &ps.ListPostgresDedicatedReadReplicasRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s or branch %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(replicas) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No dedicated read replicas exist for %s/%s.\n", printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(toDedicatedReadReplicas(replicas))
		},
	}

	return cmd
}
