package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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
			// Create a new MCP server
			s := server.NewMCPServer(
				"PlanetScale MCP Server",
				"0.1.0",
			)

			// Create a list_orgs tool without parameters
			listOrgsTool := mcp.NewTool("list_orgs",
				mcp.WithDescription("List all available organizations"),
			)

			// Add the list_orgs tool handler
			s.AddTool(listOrgsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				// Get the PlanetScale client
				client, err := ch.Client()
				if err != nil {
					return nil, fmt.Errorf("failed to initialize PlanetScale client: %w", err)
				}

				// Get the list of organizations
				orgs, err := client.Organizations.List(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to list organizations: %w", err)
				}

				// Extract only the organization names
				orgNames := make([]string, 0, len(orgs))
				for _, org := range orgs {
					orgNames = append(orgNames, org.Name)
				}

				// Convert to JSON
				orgNamesJSON, err := json.Marshal(orgNames)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal organization names: %w", err)
				}

				// Return the JSON array as text
				return mcp.NewToolResultText(string(orgNamesJSON)), nil
			})

			// Start the server
			if err := server.ServeStdio(s); err != nil {
				return fmt.Errorf("MCP server error: %v", err)
			}

			return nil
		},
	}

	return cmd
}