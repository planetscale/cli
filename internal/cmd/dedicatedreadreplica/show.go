package dedicatedreadreplica

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// ShowCmd shows a dedicated read replica by name.
func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <branch> <name>",
		Short: "Show a dedicated read replica",
		Args:  cmdutil.RequiredArgs("database", "branch", "name"),
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching dedicated read replica %s for %s/%s", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			replica, err := client.PostgresDedicatedReadReplicas.Get(ctx, &ps.GetPostgresDedicatedReadReplicaRequest{
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

			return ch.Printer.PrintResource(toDedicatedReadReplica(replica))
		},
	}

	return cmd
}
