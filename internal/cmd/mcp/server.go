package mcp

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// ServerCmd returns a new cobra.Command for the mcp server command.
func ServerCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the MCP server",
		Long:  `Start the PlanetScale model context protocol (MCP) server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("hello mcp server")
			return nil
		},
	}

	return cmd
}