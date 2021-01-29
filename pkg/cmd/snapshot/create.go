package snapshot

import (
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

func CreateCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create a new schema snapshot for a database branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return cmd
}
