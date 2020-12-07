package auth

import (
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

// AuthCmd returns the base command for authentication.
func AuthCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth <command>",
		Short: "Login, logout, and refresh your authentication",
		Long:  "Manage psctl's authentication state.",
	}

	cmd.AddCommand(LoginCmd(cfg))
	cmd.AddCommand(LogoutCmd(cfg))
	return cmd
}
