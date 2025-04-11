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

// Tool handler function type
type ToolHandler func(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error)

// Tool definition
type ToolDef struct {
	tool    mcp.Tool
	handler ToolHandler
}

// HandleListOrgs implements the list_orgs tool
func HandleListOrgs(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
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
}

// HandleListDatabases implements the list_databases tool
func HandleListDatabases(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
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
}

// HandleListBranches implements the list_branches tool
func HandleListBranches(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
	// Get the PlanetScale client
	client, err := ch.Client()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PlanetScale client: %w", err)
	}

	// Get the required database parameter
	dbArg, ok := request.Params.Arguments["database"]
	if !ok || dbArg == "" {
		return nil, fmt.Errorf("database parameter is required")
	}
	database := dbArg.(string)

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

	// Get the list of branches
	branches, err := client.DatabaseBranches.List(ctx, &planetscale.ListDatabaseBranchesRequest{
		Organization: orgName,
		Database:     database,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case planetscale.ErrNotFound:
			return nil, fmt.Errorf("database %s does not exist in organization %s", database, orgName)
		default:
			return nil, fmt.Errorf("failed to list branches: %w", err)
		}
	}

	// Extract the branch names
	branchNames := make([]string, 0, len(branches))
	for _, branch := range branches {
		branchNames = append(branchNames, branch.Name)
	}

	// Convert to JSON
	branchNamesJSON, err := json.Marshal(branchNames)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal branch names: %w", err)
	}

	// Return the JSON array as text
	return mcp.NewToolResultText(string(branchNamesJSON)), nil
}

// getToolDefinitions returns the list of all available MCP tools
func getToolDefinitions() []ToolDef {
	return []ToolDef{
		{
			tool: mcp.NewTool("list_orgs",
				mcp.WithDescription("List all available organizations"),
			),
			handler: HandleListOrgs,
		},
		{
			tool: mcp.NewTool("list_databases",
				mcp.WithDescription("List all databases in an organization"),
				mcp.WithString("org",
					mcp.Description("The organization name (uses default organization if not specified)"),
				),
			),
			handler: HandleListDatabases,
		},
		{
			tool: mcp.NewTool("list_branches",
				mcp.WithDescription("List all branches for a database"),
				mcp.WithString("database",
					mcp.Description("The database name"),
					mcp.Required(),
				),
				mcp.WithString("org",
					mcp.Description("The organization name (uses default organization if not specified)"),
				),
			),
			handler: HandleListBranches,
		},
	}
}

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

			// Register all tools
			for _, toolDef := range getToolDefinitions() {
				// Create a tool-specific handler that will forward to our function
				// We need to create a local copy of the tool definition to avoid closure issues
				def := toolDef
				// AddTool expects the tool value directly
				s.AddTool(def.tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return def.handler(ctx, request, ch)
				})
			}

			// Start the server
			if err := server.ServeStdio(s); err != nil {
				return fmt.Errorf("MCP server error: %v", err)
			}

			return nil
		},
	}

	return cmd
}