package snapshot

import (
	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

// SnapshotCmd encapsulates the command for running snapshots.
func SnapshotCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot <action>",
		Short: "Create, get, and list schema snapshots",
	}

	cmd.PersistentFlags().StringVar(&cfg.Organization, "org", cfg.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(cfg))
	cmd.AddCommand(ListCmd(cfg))
	cmd.AddCommand(ShowCmd(cfg))
	return cmd
}
