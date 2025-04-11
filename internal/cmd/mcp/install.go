package mcp

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// InstallCmd returns a new cobra.Command for the mcp install command.
func InstallCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the MCP server",
		Long:  `Install the PlanetScale model context protocol (MCP) server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("hello mcp install")
			return nil
		},
	}

	return cmd
}