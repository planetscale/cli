package token

import (
	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

// TokenCmd encapsulates the command for running snapshots.
func TokenCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service-token <action>",
		Short: "Create, get, and list service tokens",
	}

	cmd.PersistentFlags().Bool("json", false, "Show output as JSON")
	cmd.PersistentFlags().StringVar(&cfg.Organization, "org", cfg.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(cfg))
	cmd.AddCommand(ListCmd(cfg))
	cmd.AddCommand(GetCmd(cfg))
	cmd.AddCommand(AddAccessCmd(cfg))
	cmd.AddCommand(DeleteAccessCmd(cfg))
	cmd.AddCommand(DeleteCmd(cfg))
	return cmd
}
