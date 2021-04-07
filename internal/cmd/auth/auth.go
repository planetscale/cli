package auth

import (
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// AuthCmd returns the base command for authentication.
func AuthCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth <command>",
		Short: "Login, logout, and refresh your authentication",
		Long:  "Manage authentication",
	}

	cmd.AddCommand(LoginCmd(ch))
	cmd.AddCommand(LogoutCmd(ch))
	return cmd
}
