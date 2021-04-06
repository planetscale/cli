package branch

import (
	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

// BranchCmd handles the branching of a database.
func BranchCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch <command>",
		Short: "Create, delete, and manage branches",
	}

	cmd.PersistentFlags().StringVar(&cfg.Organization, "org", cfg.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(cfg))
	cmd.AddCommand(ListCmd(cfg))
	cmd.AddCommand(StatusCmd(cfg))
	cmd.AddCommand(DeleteCmd(cfg))
	cmd.AddCommand(ShowCmd(cfg))
	cmd.AddCommand(SwitchCmd(cfg))

	return cmd
}
