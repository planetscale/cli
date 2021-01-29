package snapshot

import (
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

// GetCmd makes a command for fetching a single snapshot by its ID.
func GetCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <snapshot_id>",
		Short: "Get a specific schema snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return cmd
}
