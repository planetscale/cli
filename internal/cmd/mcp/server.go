package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/planetscale-go/planetscale"
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

			// Create a list_databases tool with an optional org parameter
			listDatabasesTool := mcp.NewTool("list_databases",
				mcp.WithDescription("List all databases in an organization"),
				mcp.WithString("org", 
					mcp.Description("The organization name (uses default organization if not specified)"),
				),
			)

			// Add the list_databases tool handler
			s.AddTool(listDatabasesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				// Get the PlanetScale client
				client, err := ch.Client()
				if err != nil {
					return nil, fmt.Errorf("failed to initialize PlanetScale client: %w", err)
				}

				// Get the organization from the parameters or use the default
				var orgName string
				if org, ok := request.Params.Arguments["org"].(string); ok && org != "" {
					orgName = org
				} else {
					// Try to load from default config file
					fileConfig, err := ch.ConfigFS.DefaultConfig()
					if err == nil && fileConfig.Organization != "" {
						orgName = fileConfig.Organization
					} else {
						// Fall back to the config passed to the helper
						orgName = ch.Config.Organization
					}
				}

				if orgName == "" {
					return nil, fmt.Errorf("no organization specified and no default organization set")
				}

				// Get the list of databases
				databases, err := client.Databases.List(ctx, &planetscale.ListDatabasesRequest{
					Organization: orgName,
				})
				if err != nil {
					switch cmdutil.ErrCode(err) {
					case planetscale.ErrNotFound:
						return nil, fmt.Errorf("organization %s does not exist or your account is not authorized to access it", orgName)
					default:
						return nil, fmt.Errorf("failed to list databases: %w", err)
					}
				}

				// Extract only the database names
				dbNames := make([]string, 0, len(databases))
				for _, db := range databases {
					dbNames = append(dbNames, db.Name)
				}

				// Convert to JSON
				dbNamesJSON, err := json.Marshal(dbNames)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal database names: %w", err)
				}

				// Return the JSON array as text
				return mcp.NewToolResultText(string(dbNamesJSON)), nil
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