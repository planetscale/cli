package snapshot

import (
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

// ListCmd makes a command for listing all snapshots for a database and branch.
func ListCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return cmd
}
