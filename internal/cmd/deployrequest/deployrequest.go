package deployrequest

import (
	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

// DeployRequestCmd encapsulates the commands for creatind and managing Deploy
// Requests.
func DeployRequestCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deploy-request <command>",
		Short:   "Create, approve, diff, and manage deploy requests",
		Aliases: []string{"dr"},
	}

	cmd.PersistentFlags().StringVar(&cfg.Organization, "org", cfg.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CloseCmd(cfg))
	cmd.AddCommand(CreateCmd(cfg))
	cmd.AddCommand(DeployCmd(cfg))
	cmd.AddCommand(DiffCmd(cfg))
	cmd.AddCommand(ListCmd(cfg))
	cmd.AddCommand(ReviewCmd(cfg))
	cmd.AddCommand(ShowCmd(cfg))

	return cmd
}
