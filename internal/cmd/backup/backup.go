package backup

import (
	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

// BackupCmd handles branch backups.
func BackupCmd(cfg *config.Config) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "backup <command>",
		Short: "Create, read, destroy, and update branch backups",
	}

	cmd.PersistentFlags().StringVar(&cfg.Organization, "org", cfg.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(cfg))
	cmd.AddCommand(ListCmd(cfg))
	cmd.AddCommand(DeleteCmd(cfg))
	cmd.AddCommand(ShowCmd(cfg))

	return cmd
}
