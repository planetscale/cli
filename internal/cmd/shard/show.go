package shard

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show <database> <branch> <shard-id>",
		Short:   "Show a shard for a Neki database branch",
		Args:    cmdutil.ExactArgs("database", "branch", "shard-id"),
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, shardID := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching shard %s", printer.BoldBlue(shardID)))
			defer end()

			shard, err := client.NekiShards.Get(cmd.Context(), &ps.GetNekiShardRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Shard:        shardID,
			})
			if err != nil {
				return handleError(err, database, branch, shardID, "")
			}
			end()

			return ch.Printer.PrintResource(toShard(shard))
		},
	}

	return cmd
}
