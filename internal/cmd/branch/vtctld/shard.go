package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// GetShardCmd reads a shard record from the cluster via vtctld.
func GetShardCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
		shard    string
	}

	cmd := &cobra.Command{
		Use:   "get-shard <database> <branch>",
		Short: "Get a shard record for a branch",
		Long:  "Get a live shard record from the cluster via vtctld, including tablet controls and denied tables.",
		Args:  cmdutil.RequiredArgs("database", "branch"),
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
				fmt.Sprintf("Fetching shard %s/%s for %s\u2026",
					flags.keyspace, flags.shard,
					progressTarget(ch.Config.Organization, database, branch)))
			defer end()

			data, err := client.Vtctld.GetShard(ctx, &ps.VtctldGetShardRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     flags.keyspace,
				Shard:        flags.shard,
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
	cmd.MarkFlagRequired("keyspace")
	cmd.MarkFlagRequired("shard")

	return cmd
}
