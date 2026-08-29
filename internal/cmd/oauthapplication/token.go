package oauthapplication

import (
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func TokenCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token <command>",
		Short: "List and manage OAuth application tokens",
	}
	cmd.AddCommand(TokenListCmd(ch))
	cmd.AddCommand(TokenShowCmd(ch))
	cmd.AddCommand(TokenDeleteCmd(ch))
	cmd.AddCommand(TokenCreateCmd(ch))
	return cmd
}
