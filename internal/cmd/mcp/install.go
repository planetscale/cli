package mcp

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// InstallCmd returns a new cobra.Command for the mcp install command.
func InstallCmd(ch *cmdutil.Helper) *cobra.Command {
	var target string
	
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the MCP server",
		Long:  `Install the PlanetScale model context protocol (MCP) server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if target != "claude" {
				return fmt.Errorf("invalid target vendor: %s (only 'claude' is supported)", target)
			}
			fmt.Printf("hello mcp install for target: %s\n", target)
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Target vendor for MCP installation (required). Possible values: [claude]")
	cmd.MarkFlagRequired("target")
	cmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"claude"}, cobra.ShellCompDirectiveDefault
	})

	return cmd
}
