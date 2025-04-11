package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/planetscale-go/planetscale"
)

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
		handledErr := cmdutil.HandleError(err)
		if handledErr != err {
			return nil, handledErr
		}
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
			handledErr := cmdutil.HandleError(err)
			if handledErr != err {
				return nil, handledErr
			}
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
			handledErr := cmdutil.HandleError(err)
			if handledErr != err {
				return nil, handledErr
			}
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
			handledErr := cmdutil.HandleError(err)
			if handledErr != err {
				return nil, handledErr
			}
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

// HandleRunQuery implements the run_query tool
func HandleRunQuery(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
	// Get the required query parameter
	queryArg, ok := request.Params.Arguments["query"]
	if !ok || queryArg == "" {
		return nil, fmt.Errorf("query parameter is required")
	}
	query := queryArg.(string)

	// Create a database connection
	conn, err := createDatabaseConnection(ctx, request, ch)
	if err != nil {
		handledErr := cmdutil.HandleError(err)
		if handledErr != err {
			return nil, handledErr
		}
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
	// Create a database connection
	conn, err := createDatabaseConnection(ctx, request, ch)
	if err != nil {
		handledErr := cmdutil.HandleError(err)
		if handledErr != err {
			return nil, handledErr
		}
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
	// Get the required tables parameter
	tablesArg, ok := request.Params.Arguments["tables"]
	if !ok || tablesArg == "" {
		return nil, fmt.Errorf("tables parameter is required")
	}
	tables := tablesArg.(string)

	// Create a database connection
	conn, err := createDatabaseConnection(ctx, request, ch)
	if err != nil {
		handledErr := cmdutil.HandleError(err)
		if handledErr != err {
			return nil, handledErr
		}
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
