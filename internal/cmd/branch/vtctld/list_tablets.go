package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/spf13/cobra"
)

func ListTabletsCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace      string
		shard         string
		tabletType    string
		tabletAliases []string
	}

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
				Organization:  ch.Config.Organization,
				Database:      database,
				Branch:        branch,
				Keyspace:      flags.keyspace,
				Shard:         flags.shard,
				TabletType:    flags.tabletType,
				TabletAliases: flags.tabletAliases,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrintJSON(groups)
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Only list tablets in this keyspace")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Only list tablets in this shard (requires --keyspace)")
	cmd.Flags().StringVar(&flags.tabletType, "tablet-type", "", "Only list tablets of this type (e.g. primary, replica, rdonly)")
	cmd.Flags().StringSliceVar(&flags.tabletAliases, "tablet-alias", nil,
		"Only list the tablet(s) with these aliases, e.g. zone1-0000000100 (comma-separated; overrides the other filters)")

	return cmd
}
