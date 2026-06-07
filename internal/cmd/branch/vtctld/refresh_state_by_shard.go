package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// RefreshStateByShardCmd reloads tablet records for all tablets in a shard via vtctld.
func RefreshStateByShardCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
		shard    string
		cells    []string
	}

	cmd := &cobra.Command{
		Use:   "refresh-state-by-shard <database> <branch>",
		Short: "Reload tablet records for all tablets in a shard",
		Long: "Reload the tablet record for all tablets in a shard via vtctld, " +
			"optionally limited to the specified cells.",
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			if flags.keyspace == "" {
				return fmt.Errorf("keyspace is required")
			}
			if flags.shard == "" {
				return fmt.Errorf("shard is required")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Refreshing tablet state for %s/%s on %s…",
					flags.keyspace, flags.shard,
					progressTarget(ch.Config.Organization, database, branch)))
			defer end()

			data, err := client.Vtctld.RefreshStateByShard(ctx, &ps.VtctldRefreshStateByShardRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     flags.keyspace,
				Shard:        flags.shard,
				Cells:        flags.cells,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace name")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Shard name (e.g. \"-\" for unsharded)")
	cmd.Flags().StringSliceVar(&flags.cells, "cells", nil, "Cells to refresh (comma-separated)")
	cmd.MarkFlagRequired("keyspace")
	cmd.MarkFlagRequired("shard")

	return cmd
}
