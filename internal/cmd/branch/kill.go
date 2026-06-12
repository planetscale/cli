package branch

import (
	"strconv"

	"github.com/planetscale/cli/internal/cmd/branch/connections"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// ProcesslistKillCmd kills a running MySQL process on a Vitess branch.
func ProcesslistKillCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
		shard    string
		query    bool
	}

	cmd := &cobra.Command{
		Use:   "kill <database> <branch> <id>",
		Short: "Kill a running MySQL process on a branch (Vitess only)",
		Long: `Compatibility command for killing Vitess processlist IDs.

This command is only supported for Vitess (MySQL) databases.

Use "pscale branch connections kill" with connection_id and query_id values for
new workflows.

By default the entire connection is killed (KILL <id>). Pass --query to kill
only the currently running statement (KILL QUERY <id>). The process must live on
the same primary tablet that the process list was read from, so the same
--keyspace/--shard targeting rules apply.`,
		Args: cmdutil.RequiredArgs("database", "branch", "id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]

			if err := requireProcesslistDatabase(cmd.Context(), ch, database); err != nil {
				return err
			}

			id, err := vitessConnectionID(args[2])
			if err != nil {
				return err
			}

			target := connections.ConnectionTarget{Keyspace: flags.keyspace, Shard: flags.shard}
			idText := strconv.FormatInt(id, 10)
			if flags.query {
				return connections.RunCancelQuery(cmd.Context(), ch, database, branch, idText, target)
			}
			return connections.RunKillConnection(cmd.Context(), ch, database, branch, idText, target)
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace to target (required when the database has multiple keyspaces)")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Shard to target (required when the targeted keyspace is sharded)")
	cmd.Flags().BoolVar(&flags.query, "query", false, "Kill only the running query (KILL QUERY) instead of the whole connection")
	cmd.SetFlagErrorFunc(vitessConnectionIDFlagError)

	return cmd
}
