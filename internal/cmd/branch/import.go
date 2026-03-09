package branch

import (
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
)

// ImportCmd is the top-level command for PostgreSQL database imports.
func ImportCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <command>",
		Short: "Import PostgreSQL databases into PlanetScale",
		Long: `Import PostgreSQL databases from external sources (Supabase, Neon, Digital Ocean, etc.)
into PlanetScale using logical replication.

This command group provides tools to:
- Start a new import using logical replication
- Monitor the status of an active import
- Complete an import by finalizing sequence values
- Cancel and clean up an import
- List active imports

The import process uses PostgreSQL logical replication to copy data from a source
database to a PlanetScale PostgreSQL branch with minimal downtime.`,
	}

	cmd.AddCommand(ImportStartCmd(ch))
	cmd.AddCommand(ImportStatusCmd(ch))
	cmd.AddCommand(ImportCompleteCmd(ch))
	cmd.AddCommand(ImportCancelCmd(ch))
	cmd.AddCommand(ImportListCmd(ch))

	return cmd
}
