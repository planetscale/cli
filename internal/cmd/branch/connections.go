package branch

import (
	"github.com/planetscale/cli/internal/cmd/branch/connections"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// ConnectionsCmd manages branch connections across supported database engines.
func ConnectionsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connections <command>",
		Short: "Show and kill branch connections",
		Long: `Show and kill branch connections.

Agent workflow:
  1. Run: pscale branch connections show <database> <branch> --format json
  2. Inspect query_id, transaction_id, and connection_id from the selected row.
  3. Explain the proposed action and wait for user approval before running it.
  4. Run exactly one action command with the matching ID.
  5. Run show again to verify the result.

Action semantics:
  kill <database> <branch> <query-id> --query        Cancels the listed query_id.
  kill-transaction <database> <branch> <transaction-id>
                                                     Postgres only. destructive. Terminates the listed transaction_id if it still matches server state.
  kill <database> <branch> <connection-id>           destructive. Terminates the listed connection_id.

Use --format json when an agent or script needs to inspect query_id,
transaction_id, and connection_id fields. Human output uses vertical records so
query text and action IDs are not truncated.`,
	}

	cmd.AddCommand(ConnectionsShowCmd(ch))
	cmd.AddCommand(ConnectionsKillCmd(ch))
	cmd.AddCommand(ConnectionsKillTransactionCmd(ch))
	cmd.AddCommand(connections.TopCmd(ch))

	return cmd
}
