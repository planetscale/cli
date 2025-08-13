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
	namingBlurb := ".  Two common naming conventions for PlanetScale databases are <org>/<database> and <org>/<database>/<branch>. When the user provides a database identifier in either of these formats, automatically parse and use the org, database, and branch parameters directly - do not perform discovery steps like list_orgs or list_databases.  Examples: `acme/widgets` -> org=acme, database=widgets.  `acme/widgets/main` -> org=acme, database=widgets, branch=main.  If the user provides an identifier like 'org/database' or 'org/database/branch', parse these components directly and skip organizational/database discovery steps."
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
					mcp.Description("The organization name"),
					mcp.Required(),
				),
			),
			handler: HandleListDatabases,
		},
		{
			tool: mcp.NewTool("list_branches",
				mcp.WithDescription("List all branches for a database" + namingBlurb),
				mcp.WithString("database",
					mcp.Description("The database name"),
					mcp.Required(),
				),
				mcp.WithString("org",
					mcp.Description("The organization name"),
					mcp.Required(),
				),
			),
			handler: HandleListBranches,
		},
		{
			tool: mcp.NewTool("list_keyspaces",
				mcp.WithDescription("List all keyspaces within a branch" + namingBlurb),
				mcp.WithString("database",
					mcp.Description("The database name"),
					mcp.Required(),
				),
				mcp.WithString("branch",
					mcp.Description("The branch name"),
					mcp.Required(),
				),
				mcp.WithString("org",
					mcp.Description("The organization name"),
					mcp.Required(),
				),
			),
			handler: HandleListKeyspaces,
		},
		{
			tool: mcp.NewTool("list_tables",
				mcp.WithDescription("List all tables in a keyspace/database" + namingBlurb),
				mcp.WithString("database",
					mcp.Description("The database name"),
					mcp.Required(),
				),
				mcp.WithString("branch",
					mcp.Description("The branch name"),
					mcp.Required(),
				),
				mcp.WithString("keyspace",
					mcp.Description("The keyspace name (for MySQL) or inner database name (for PostgreSQL)"),
				),
				mcp.WithString("schema",
					mcp.Description("Filter tables by schema (PostgreSQL only)"),
				),
				mcp.WithString("org",
					mcp.Description("The organization name"),
					mcp.Required(),
				),
			),
			handler: HandleListTables,
		},
		{
			tool: mcp.NewTool("get_schema",
				mcp.WithDescription("Get the SQL schema for tables in a keyspace/database" + namingBlurb),
				mcp.WithString("database",
					mcp.Description("The database name"),
					mcp.Required(),
				),
				mcp.WithString("branch",
					mcp.Description("The branch name"),
					mcp.Required(),
				),
				mcp.WithString("keyspace",
					mcp.Description("The keyspace name (for MySQL) or inner database name (for PostgreSQL)"),
				),
				mcp.WithString("tables",
					mcp.Description("Tables to get schemas for. MySQL: comma-separated list of table names, or '*' for all tables. PostgreSQL: comma-separated list of simple table names in the public schema, qualified schema.table_names, 'schema.*' for all tables in a schema, or '*' for all tables in all schemas"),
					mcp.Required(),
				),
				mcp.WithString("org",
					mcp.Description("The organization name"),
					mcp.Required(),
				),
			),
			handler: HandleGetSchema,
		},
		{
			tool: mcp.NewTool("run_query",
				mcp.WithDescription("Run a SQL query against a database branch keyspace/database" + namingBlurb),
				mcp.WithString("database",
					mcp.Description("The database name"),
					mcp.Required(),
				),
				mcp.WithString("branch",
					mcp.Description("The branch name"),
					mcp.Required(),
				),
				mcp.WithString("keyspace",
					mcp.Description("The keyspace name (for MySQL) or inner database name (for PostgreSQL)"),
				),
				mcp.WithString("query",
					mcp.Description("The SQL query to run (read-only queries only)"),
					mcp.Required(),
				),
				mcp.WithString("org",
					mcp.Description("The organization name"),
					mcp.Required(),
				),
			),
			handler: HandleRunQuery,
		},
		{
			tool: mcp.NewTool("get_insights",
				mcp.WithDescription("Get recent performance data for a database branch" + namingBlurb),
				mcp.WithString("database",
					mcp.Description("The database name"),
					mcp.Required(),
				),
				mcp.WithString("branch",
					mcp.Description("The branch name"),
					mcp.Required(),
				),
				mcp.WithString("org",
					mcp.Description("The organization name"),
					mcp.Required(),
				),
			),
			handler: HandleGetInsights,
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
				server.WithInstructions(`PlanetScale Database Management tools

These tools provide read-only access to MySQL (Vitess) and PostgreSQL
databases hosted by PlanetScale.  The naming convention for PlanetScale
databases is <org>/<database> or <org>/<database>/<branch>, so if the user
refers to databases in this format, they are specifying the org, database,
and possibly branch needed to invoke the tools in this MCP.  Usually if
the branch is omitted, the default is "main".

When PlanetScale uses the word "database", it refers to a collection of
database branches (production, development, staging, etc.), each with one
or more replicas.  This is distinct from PostgreSQL's notion of a database
as a namespace on a single server, which PlanetScale calls a "keyspace"
for both MySQL and PostgreSQL.`),
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
