package branch

import (
	"context"
	"errors"

	"github.com/planetscale/cli/internal/cmd/branch/connections"
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// ProcesslistCmd manages MySQL process lists for a Vitess branch.
func ProcesslistCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "processlist <command>",
		Short: "Show and kill running MySQL processes for a branch",
		Long: `Show and kill running MySQL processes for a branch.

This command is only supported for Vitess databases.`,
		Hidden: true,
	}

	cmd.AddCommand(ProcesslistShowCmd(ch))
	cmd.AddCommand(ProcesslistKillCmd(ch))

	return cmd
}

// ProcesslistShowCmd shows the MySQL process list for a Vitess branch.
func ProcesslistShowCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
		shard    string
	}

	cmd := &cobra.Command{
		Use:   "show <database> <branch>",
		Short: "Show the running MySQL processes for a branch",
		Long: `Compatibility alias for "pscale branch connections show".

This command is only supported for Vitess databases.

Use "pscale branch connections show" for new workflows. Use connection_id and
query_id values with "pscale branch connections kill".

The process list is read from a single primary tablet. If the database has a
single unsharded keyspace, that primary is targeted automatically. If it has
multiple keyspaces, pass --keyspace; if the targeted keyspace is sharded, also
pass --shard.`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireProcesslistDatabase(cmd.Context(), ch, args[0]); err != nil {
				return err
			}

			return connections.RunList(cmd.Context(), ch, args[0], args[1], connections.ConnectionFilter{}, connections.ConnectionTarget{
				Keyspace: flags.keyspace,
				Shard:    flags.shard,
			})
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace to target (required when the database has multiple keyspaces)")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Shard to target (required when the targeted keyspace is sharded)")

	return cmd
}

func requireProcesslistDatabase(ctx context.Context, ch *cmdutil.Helper, database string) error {
	engine, err := databaseEngine(ctx, ch, database)
	if err != nil {
		return err
	}
	if engine != ps.DatabaseEngineMySQL {
		return errors.New("processlist is only supported for Vitess databases")
	}
	return nil
}
