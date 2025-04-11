package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/passwordutil"
	"github.com/planetscale/cli/internal/proxyutil"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/planetscale/psdbproxy"
	"github.com/spf13/cobra"
	"vitess.io/vitess/go/mysql"
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

// getOrganization extracts the organization from the request parameters or falls back to defaults
func getOrganization(request mcp.CallToolRequest, ch *cmdutil.Helper) (string, error) {
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
		return "", fmt.Errorf("no organization specified and no default organization set")
	}
	
	return orgName, nil
}

// HandleListDatabases implements the list_databases tool
func HandleListDatabases(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
	// Get the PlanetScale client
	client, err := ch.Client()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PlanetScale client: %w", err)
	}

	// Get the organization
	orgName, err := getOrganization(request, ch)
	if err != nil {
		return nil, err
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

	// Get the organization
	orgName, err := getOrganization(request, ch)
	if err != nil {
		return nil, err
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

// HandleListKeyspaces implements the list_keyspaces tool
func HandleListKeyspaces(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
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

	// Get the required branch parameter
	branchArg, ok := request.Params.Arguments["branch"]
	if !ok || branchArg == "" {
		return nil, fmt.Errorf("branch parameter is required")
	}
	branch := branchArg.(string)

	// Get the organization
	orgName, err := getOrganization(request, ch)
	if err != nil {
		return nil, err
	}

	// Get the list of keyspaces
	keyspaces, err := client.Keyspaces.List(ctx, &planetscale.ListKeyspacesRequest{
		Organization: orgName,
		Database:     database,
		Branch:       branch,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case planetscale.ErrNotFound:
			return nil, fmt.Errorf("database %s or branch %s does not exist in organization %s", database, branch, orgName)
		default:
			return nil, fmt.Errorf("failed to list keyspaces: %w", err)
		}
	}

	// Extract the keyspace names
	keyspaceNames := make([]string, 0, len(keyspaces))
	for _, keyspace := range keyspaces {
		keyspaceNames = append(keyspaceNames, keyspace.Name)
	}

	// Convert to JSON
	keyspaceNamesJSON, err := json.Marshal(keyspaceNames)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal keyspace names: %w", err)
	}

	// Return the JSON array as text
	return mcp.NewToolResultText(string(keyspaceNamesJSON)), nil
}

// DatabaseConnection represents a connection to a PlanetScale database
type DatabaseConnection struct {
	db         *sql.DB
	proxy      *psdbproxy.Server
	listener   net.Listener
	password   *passwordutil.Password
	keyspace   string
	localAddr  string
}

// createDatabaseConnection establishes a connection to a PlanetScale database
func createDatabaseConnection(ctx context.Context, client *planetscale.Client, orgName, database, branch, keyspace string, ch *cmdutil.Helper) (*DatabaseConnection, error) {
	// Check if database and branch exist
	dbBranch, err := client.DatabaseBranches.Get(ctx, &planetscale.GetDatabaseBranchRequest{
		Organization: orgName,
		Database:     database,
		Branch:       branch,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case planetscale.ErrNotFound:
			return nil, fmt.Errorf("database %s and branch %s does not exist in organization %s",
				database, branch, orgName)
		default:
			return nil, fmt.Errorf("failed to get database branch: %w", err)
		}
	}

	if !dbBranch.Ready {
		return nil, fmt.Errorf("database branch is not ready yet")
	}

	// Create a temporary password with reader role
	pw, err := passwordutil.New(ctx, client, passwordutil.Options{
		Organization: orgName,
		Database:     database,
		Branch:       branch,
		Role:         cmdutil.ReaderRole, // Use reader role for safety
		Name:         passwordutil.GenerateName("pscale-cli-mcp-query"),
		TTL:          5 * time.Minute,
		Replica:      true, // Use replica for read-only queries
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary password: %w", err)
	}

	// Create a proxy for the connection
	proxy := proxyutil.New(proxyutil.Config{
		Logger:       cmdutil.NewZapLogger(ch.Debug()),
		UpstreamAddr: pw.Password.Hostname,
		Username:     pw.Password.Username,
		Password:     pw.Password.PlainText,
	})

	// Create a local listener
	l, err := net.Listen("tcp", "127.0.0.1:0") // Use random port
	if err != nil {
		pw.Cleanup(ctx)
		proxy.Close()
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	// Start serving the proxy
	errCh := make(chan error, 1)
	go func() {
		errCh <- proxy.Serve(l, mysql.CachingSha2Password)
	}()

	// Get the local address
	localAddr := l.Addr().String()

	// Create a MySQL connection to the local proxy
	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(%s)/%s", localAddr, keyspace))
	if err != nil {
		pw.Cleanup(ctx)
		proxy.Close()
		l.Close()
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	// Set a timeout for the connection
	db.SetConnMaxLifetime(30 * time.Second)

	return &DatabaseConnection{
		db:        db,
		proxy:     proxy,
		listener:  l,
		password:  pw,
		keyspace:  keyspace,
		localAddr: localAddr,
	}, nil
}

// close closes all resources associated with the database connection
func (c *DatabaseConnection) close() {
	if c.db != nil {
		c.db.Close()
	}
	if c.listener != nil {
		c.listener.Close()
	}
	if c.proxy != nil {
		c.proxy.Close()
	}
	if c.password != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.password.Cleanup(ctx)
	}
}

// executeQuery executes a SQL query and returns the results as an array of maps
func executeQuery(ctx context.Context, conn *DatabaseConnection, query string) ([]map[string]interface{}, error) {
	// Execute the query
	rows, err := conn.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %w", err)
	}

	// Prepare a slice of interface{} to hold the row values
	values := make([]interface{}, len(columns))
	scanArgs := make([]interface{}, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Convert rows to map objects
	var results []map[string]interface{}
	for rows.Next() {
		// Scan the row into the values slice
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create a map for the row
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]

			// Handle different types
			switch v := val.(type) {
			case []byte:
				// Try to convert to string
				rowMap[col] = string(v)
			case nil:
				rowMap[col] = nil
			default:
				rowMap[col] = v
			}
		}

		results = append(results, rowMap)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error while iterating over query results: %w", err)
	}

	return results, nil
}

// HandleRunQuery implements the run_query tool
func HandleRunQuery(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
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

	// Get the required branch parameter
	branchArg, ok := request.Params.Arguments["branch"]
	if !ok || branchArg == "" {
		return nil, fmt.Errorf("branch parameter is required")
	}
	branch := branchArg.(string)

	// Get the required keyspace parameter
	keyspaceArg, ok := request.Params.Arguments["keyspace"]
	if !ok || keyspaceArg == "" {
		return nil, fmt.Errorf("keyspace parameter is required")
	}
	keyspace := keyspaceArg.(string)

	// Get the required query parameter
	queryArg, ok := request.Params.Arguments["query"]
	if !ok || queryArg == "" {
		return nil, fmt.Errorf("query parameter is required")
	}
	query := queryArg.(string)

	// Get the organization
	orgName, err := getOrganization(request, ch)
	if err != nil {
		return nil, err
	}

	// Create a database connection
	conn, err := createDatabaseConnection(ctx, client, orgName, database, branch, keyspace, ch)
	if err != nil {
		return nil, err
	}
	defer conn.close()

	// Execute the query
	results, err := executeQuery(ctx, conn, query)
	if err != nil {
		return nil, err
	}

	// Convert to JSON
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query results: %w", err)
	}

	// Return the JSON array as text
	return mcp.NewToolResultText(string(resultsJSON)), nil
}

// HandleListTables implements the list_tables tool
func HandleListTables(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
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

	// Get the required branch parameter
	branchArg, ok := request.Params.Arguments["branch"]
	if !ok || branchArg == "" {
		return nil, fmt.Errorf("branch parameter is required")
	}
	branch := branchArg.(string)

	// Get the required keyspace parameter
	keyspaceArg, ok := request.Params.Arguments["keyspace"]
	if !ok || keyspaceArg == "" {
		return nil, fmt.Errorf("keyspace parameter is required")
	}
	keyspace := keyspaceArg.(string)

	// Get the organization
	orgName, err := getOrganization(request, ch)
	if err != nil {
		return nil, err
	}

	// Create a database connection
	conn, err := createDatabaseConnection(ctx, client, orgName, database, branch, keyspace, ch)
	if err != nil {
		return nil, err
	}
	defer conn.close()

	// Execute the SHOW TABLES query
	results, err := executeQuery(ctx, conn, "SHOW TABLES")
	if err != nil {
		return nil, err
	}

	// Extract just the table names from the results
	tableNames := make([]string, 0, len(results))
	for _, row := range results {
		// Each row has only one value, so we can just take the first value we find
		for _, value := range row {
			if tableName, ok := value.(string); ok {
				tableNames = append(tableNames, tableName)
				break // Only need the first value
			}
		}
	}

	// Convert to JSON
	tableNamesJSON, err := json.Marshal(tableNames)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal table names: %w", err)
	}

	// Return the JSON array as text
	return mcp.NewToolResultText(string(tableNamesJSON)), nil
}

// HandleGetSchema implements the get_schema tool
func HandleGetSchema(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
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

	// Get the required branch parameter
	branchArg, ok := request.Params.Arguments["branch"]
	if !ok || branchArg == "" {
		return nil, fmt.Errorf("branch parameter is required")
	}
	branch := branchArg.(string)

	// Get the required keyspace parameter
	keyspaceArg, ok := request.Params.Arguments["keyspace"]
	if !ok || keyspaceArg == "" {
		return nil, fmt.Errorf("keyspace parameter is required")
	}
	keyspace := keyspaceArg.(string)

	// Get the required tables parameter
	tablesArg, ok := request.Params.Arguments["tables"]
	if !ok || tablesArg == "" {
		return nil, fmt.Errorf("tables parameter is required")
	}
	tables := tablesArg.(string)

	// Get the organization
	orgName, err := getOrganization(request, ch)
	if err != nil {
		return nil, err
	}

	// Create a database connection
	conn, err := createDatabaseConnection(ctx, client, orgName, database, branch, keyspace, ch)
	if err != nil {
		return nil, err
	}
	defer conn.close()

	// Define a list of tables to get schemas for
	var tableList []string

	// If tables is "*", fetch all tables in the keyspace
	if tables == "*" {
		// Execute the SHOW TABLES query
		results, err := executeQuery(ctx, conn, "SHOW TABLES")
		if err != nil {
			return nil, err
		}

		// Extract the table names from the results
		for _, row := range results {
			for _, value := range row {
				if tableName, ok := value.(string); ok {
					tableList = append(tableList, tableName)
					break // Only need the first value
				}
			}
		}
	} else {
		// Split the comma-separated list of tables
		for _, table := range strings.Split(tables, ",") {
			trimmedTable := strings.TrimSpace(table)
			if trimmedTable != "" {
				tableList = append(tableList, trimmedTable)
			}
		}
	}

	// Create a map to store table schemas
	schemas := make(map[string]string)

	// For each table, get the schema
	for _, table := range tableList {
		query := fmt.Sprintf("SHOW CREATE TABLE `%s`", table)
		results, err := executeQuery(ctx, conn, query)
		if err != nil {
			return nil, fmt.Errorf("failed to get schema for table %s: %w", table, err)
		}

		// Extract the schema from the results
		if len(results) > 0 {
			// The second column has the CREATE TABLE statement
			for colName, value := range results[0] {
				if colName == "Create Table" {
					if schema, ok := value.(string); ok {
						schemas[table] = schema
						break
					}
				}
			}
		}
	}

	// Convert to JSON
	schemasJSON, err := json.Marshal(schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schemas: %w", err)
	}

	// Return the JSON object as text
	return mcp.NewToolResultText(string(schemasJSON)), nil
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