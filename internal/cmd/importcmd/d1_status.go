package importcmd

import (
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/import/d1"
)

func d1StatusCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		migrationID string
	}

	cmd := &cobra.Command{
		Use:     "status <database> [branch]",
		Short:   "Show local migration state",
		Args:    d1DatabaseBranchArgs,
		Example: `  pscale import d1 status mydb --migration-id abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := parseDatabaseBranch(args)
			state, err := d1.Status(d1Org(ch), database, branch, flags.migrationID)
			if err != nil {
				return writeD1(ch, d1.ErrorResponse("status", err))
			}
			return writeD1(ch, d1.StatusResponse(state))
		},
	}

	cmd.Flags().StringVar(&flags.migrationID, "migration-id", "", "Migration ID")
	cmd.MarkFlagRequired("migration-id")
	return cmd
}
