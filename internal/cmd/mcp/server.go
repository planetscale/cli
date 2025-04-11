package mcp

import (
	"context"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// Tool handler function type
type ToolHandler func(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error)

// Tool definition
type ToolDef struct {
	tool    mcp.Tool
	handler ToolHandler
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
		{
			tool: mcp.NewTool("list_keyspaces",
				mcp.WithDescription("List all keyspaces within a branch"),
				mcp.WithString("database",
					mcp.Description("The database name"),
					mcp.Required(),
				),
				mcp.WithString("branch",
					mcp.Description("The branch name"),
					mcp.Required(),
				),
				mcp.WithString("org",
					mcp.Description("The organization name (uses default organization if not specified)"),
				),
			),
			handler: HandleListKeyspaces,
		},
		{
			tool: mcp.NewTool("list_tables",
				mcp.WithDescription("List all tables in a keyspace"),
				mcp.WithString("database",
					mcp.Description("The database name"),
					mcp.Required(),
				),
				mcp.WithString("branch",
					mcp.Description("The branch name"),
					mcp.Required(),
				),
				mcp.WithString("keyspace",
					mcp.Description("The keyspace name"),
					mcp.Required(),
				),
				mcp.WithString("org",
					mcp.Description("The organization name (uses default organization if not specified)"),
				),
			),
			handler: HandleListTables,
		},
		{
			tool: mcp.NewTool("get_schema",
				mcp.WithDescription("Get the SQL schema for tables in a keyspace"),
				mcp.WithString("database",
					mcp.Description("The database name"),
					mcp.Required(),
				),
				mcp.WithString("branch",
					mcp.Description("The branch name"),
					mcp.Required(),
				),
				mcp.WithString("keyspace",
					mcp.Description("The keyspace name"),
					mcp.Required(),
				),
				mcp.WithString("tables",
					mcp.Description("Tables to get schemas for (single name, comma-separated list, or '*' for all tables)"),
					mcp.Required(),
				),
				mcp.WithString("org",
					mcp.Description("The organization name (uses default organization if not specified)"),
				),
			),
			handler: HandleGetSchema,
		},
		{
			tool: mcp.NewTool("run_query",
				mcp.WithDescription("Run a SQL query against a database branch keyspace"),
				mcp.WithString("database",
					mcp.Description("The database name"),
					mcp.Required(),
				),
				mcp.WithString("branch",
					mcp.Description("The branch name"),
					mcp.Required(),
				),
				mcp.WithString("keyspace",
					mcp.Description("The keyspace name"),
					mcp.Required(),
				),
				mcp.WithString("query",
					mcp.Description("The SQL query to run (read-only queries only)"),
					mcp.Required(),
				),
				mcp.WithString("org",
					mcp.Description("The organization name (uses default organization if not specified)"),
				),
			),
			handler: HandleRunQuery,
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