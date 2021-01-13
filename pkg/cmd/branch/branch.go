package branch

import (
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

func BranchCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "branch <command|source-database> <branch-name> [options]",
		Short:   "Branch a production database",
		Aliases: []string{"b"},
	}

	cmd.Flags().StringVar(nil, "notes", "", "notes for the database branch")

	return cmd
}
