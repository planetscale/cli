package mcp

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// McpCmd returns a new cobra.Command for the mcp command.
func McpCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp <command>",
		Short: "Manage and use the MCP server",
		Long:  `Manage and use the PlanetScale model context protocol (MCP) server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("hello mcp")
			return nil
		},
	}

	return cmd
}
