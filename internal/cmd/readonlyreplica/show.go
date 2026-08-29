package readonlyreplica

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// ShowCmd shows a read-only replica by name.
func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <branch> <name>",
		Short: "Show a read-only replica",
		Args:  cmdutil.RequiredArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, name := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}
			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "read-only replicas"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching read-only replica %s for %s/%s", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			replica, err := client.PostgresReadOnlyReplicas.Get(ctx, &ps.GetPostgresReadOnlyReplicaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Replica:      name,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("read-only replica %s does not exist on %s/%s (organization: %s)",
						printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toReadOnlyReplica(replica))
		},
	}

	return cmd
}
