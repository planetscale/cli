package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ListTabletsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-tablets <database> <branch>",
		Short: "List tablets for a branch, grouped by keyspace and shard",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Listing tablets on %s\u2026",
					progressTarget(ch.Config.Organization, database, branch)))
			defer end()

			groups, err := client.Vtctld.ListTablets(ctx, &ps.ListBranchTabletsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrintJSON(groups)
		},
	}

	return cmd
}
