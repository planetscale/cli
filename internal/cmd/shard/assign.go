package shard

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func AssignCmd(ch *cmdutil.Helper) *cobra.Command {
	var configurationProfile string

	cmd := &cobra.Command{
		Use:   "assign <database> <branch> <shard-id>...",
		Short: "Assign shards to a Neki configuration profile",
		Args:  cmdutil.RequiredArgs("database", "branch", "shard-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]
			shardIDs := args[2:]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Assigning %d shard(s) to configuration profile %s", len(shardIDs), printer.BoldBlue(configurationProfile)))
			defer end()

			assignments, err := client.NekiShards.Assign(cmd.Context(), &ps.AssignNekiShardsRequest{
				Organization:         ch.Config.Organization,
				Database:             database,
				Branch:               branch,
				ConfigurationProfile: configurationProfile,
				ShardIDs:             shardIDs,
			})
			if err != nil {
				return handleError(err, database, branch, "", configurationProfile)
			}
			end()

			return ch.Printer.PrintResource(toShardAssignments(assignments))
		},
	}

	cmd.Flags().StringVar(&configurationProfile, "config-profile", "", "Configuration profile to assign the shards to")
	cmd.MarkFlagRequired("config-profile") // nolint:errcheck

	return cmd
}
