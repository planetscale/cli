package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/planetscale-go/planetscale"
	"golang.org/x/oauth2"
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

	// Extract database names and kinds as JSON objects
	dbObjects := make([]map[string]string, 0, len(databases))
	for _, db := range databases {
		dbObjects = append(dbObjects, map[string]string{
			"name": db.Name,
			"kind": string(db.Kind),
		})
	}

	// Convert to JSON
	dbObjectsJSON, err := json.Marshal(dbObjects)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal database objects: %w", err)
	}

	// Return the JSON array as text
	return mcp.NewToolResultText(string(dbObjectsJSON)), nil
}

// HandleListBranches implements the list_branches tool
func HandleListBranches(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
	// Get the PlanetScale client
	client, err := ch.Client()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PlanetScale client: %w", err)
	}

	args := request.GetArguments()

	// Get the required database parameter
	dbArg, ok := args["database"]
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
	args := request.GetArguments()

	// Get the required database parameter
	dbArg, ok := args["database"]
	if !ok || dbArg == "" {
		return nil, fmt.Errorf("database parameter is required")
	}
	database := dbArg.(string)

	// Get the required branch parameter
	branchArg, ok := args["branch"]
	if !ok || branchArg == "" {
		return nil, fmt.Errorf("branch parameter is required")
	}
	branch := branchArg.(string)

	// Get the organization
	orgName, err := getOrganization(request, ch)
	if err != nil {
		return nil, err
	}

	// Get database kind
	dbKind, err := getDatabaseKind(ctx, ch, orgName, database)
	if err != nil {
		return nil, err
	}

	var keyspaceNames []string

	// Handle different database types
	switch dbKind {
	case "postgresql", "horizon":
		// For PostgreSQL, query pg_database to get database names (keyspaces)
		query := "SELECT datname FROM pg_database WHERE datistemplate = false AND datallowconn = true ORDER BY datname;"
		results, err := executeQueryPostgres(ctx, request, ch, query, "pscale_admin")
		if err != nil {
			return nil, fmt.Errorf("failed to query PostgreSQL databases: %w", err)
		}

		// Extract database names from results
		keyspaceNames = make([]string, 0, len(results))
		for _, row := range results {
			if datname, ok := row["datname"].(string); ok {
				keyspaceNames = append(keyspaceNames, datname)
			}
		}

	case "mysql":
		// Get the PlanetScale client for MySQL operations
		client, err := ch.Client()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize PlanetScale client: %w", err)
		}

		// For MySQL, use the existing API to get keyspaces
		keyspaces, err := client.Keyspaces.List(ctx, &planetscale.ListKeyspacesRequest{
			Organization: orgName,
			Database:     database,
			Branch:       branch,
		})
		if err != nil {
			switch cmdutil.ErrCode(err) {
			case planetscale.ErrNotFound:
				return nil, fmt.Errorf("branch %s does not exist in database %s/%s", branch, database, orgName)
			default:
				handledErr := cmdutil.HandleError(err)
				if handledErr != err {
					return nil, handledErr
				}
				return nil, fmt.Errorf("failed to list keyspaces: %w", err)
			}
		}

		// Extract the keyspace names
		keyspaceNames = make([]string, 0, len(keyspaces))
		for _, keyspace := range keyspaces {
			keyspaceNames = append(keyspaceNames, keyspace.Name)
		}

	default:
		return nil, fmt.Errorf("unsupported database kind: %s. Only 'mysql' and 'postgresql' are supported", dbKind)
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
	args := request.GetArguments()

	// Get the required database parameter
	dbArg, ok := args["database"]
	if !ok || dbArg == "" {
		return nil, fmt.Errorf("database parameter is required")
	}
	database := dbArg.(string)

	// Get the required query parameter
	queryArg, ok := args["query"]
	if !ok || queryArg == "" {
		return nil, fmt.Errorf("query parameter is required")
	}
	query := queryArg.(string)

	// Get the organization
	orgName, err := getOrganization(request, ch)
	if err != nil {
		return nil, err
	}

	// Get database kind
	dbKind, err := getDatabaseKind(ctx, ch, orgName, database)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}

	// Handle different database types
	switch dbKind {
	case "postgresql", "horizon":
		// For PostgreSQL, get keyspace parameter (database name) with default
		keyspace := "postgres" // default database
		if keyspaceArg, ok := args["keyspace"].(string); ok && keyspaceArg != "" {
			keyspace = keyspaceArg
		}

		// Execute the query using PostgreSQL connection
		results, err = executeQueryPostgres(ctx, request, ch, query, keyspace)
		if err != nil {
			return nil, fmt.Errorf("failed to execute PostgreSQL query: %w", err)
		}

	case "mysql":
		// For MySQL, keyspace parameter is required
		keyspaceArg, ok := args["keyspace"]
		if !ok || keyspaceArg == "" {
			return nil, fmt.Errorf("keyspace parameter is required for MySQL databases")
		}

		// For MySQL, use the existing approach with MySQL connection
		conn, err := createMySQLConnection(ctx, request, ch)
		if err != nil {
			handledErr := cmdutil.HandleError(err)
			if handledErr != err {
				return nil, handledErr
			}
			return nil, err
		}
		defer conn.close()

		// Execute the query
		results, err = executeQueryMySQL(ctx, conn, query)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported database kind: %s. Only 'mysql' and 'postgresql' are supported", dbKind)
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
	args := request.GetArguments()

	// Get the required database parameter
	dbArg, ok := args["database"]
	if !ok || dbArg == "" {
		return nil, fmt.Errorf("database parameter is required")
	}
	database := dbArg.(string)

	// Get the optional schema parameter
	var schemaFilter string
	if schemaArg, ok := args["schema"].(string); ok && schemaArg != "" {
		schemaFilter = schemaArg
	}

	// Get the organization
	orgName, err := getOrganization(request, ch)
	if err != nil {
		return nil, err
	}

	// Get database kind
	dbKind, err := getDatabaseKind(ctx, ch, orgName, database)
	if err != nil {
		return nil, err
	}

	// Handle different database types
	switch dbKind {
	case "postgresql", "horizon":
		// For PostgreSQL, get keyspace parameter (database name) with default
		keyspace := "postgres" // default database
		if keyspaceArg, ok := args["keyspace"].(string); ok && keyspaceArg != "" {
			keyspace = keyspaceArg
		}

		// Build PostgreSQL query with optional schema filter
		var query string
		var results []map[string]interface{}
		if schemaFilter != "" {
			// Filter by specific schema using parameterized query
			query = "SELECT schemaname, tablename, tableowner FROM pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema') AND schemaname = $1 ORDER BY tablename;"
			results, err = executeQueryPostgres(ctx, request, ch, query, keyspace, schemaFilter)
		} else {
			// No schema filter, get all tables
			query = "SELECT schemaname, tablename, tableowner FROM pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema') ORDER BY schemaname, tablename;"
			results, err = executeQueryPostgres(ctx, request, ch, query, keyspace)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query PostgreSQL tables: %w", err)
		}

		// Convert results to the expected format with schema, name, and owner fields
		tableObjects := make([]map[string]interface{}, 0, len(results))
		for _, row := range results {
			tableObj := make(map[string]interface{})

			if schema, ok := row["schemaname"]; ok {
				tableObj["schema"] = schema
			}
			if name, ok := row["tablename"]; ok {
				tableObj["name"] = name
			}
			if owner, ok := row["tableowner"]; ok {
				tableObj["owner"] = owner
			}

			tableObjects = append(tableObjects, tableObj)
		}

		// Convert to JSON
		tableObjectsJSON, err := json.Marshal(tableObjects)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal PostgreSQL table objects: %w", err)
		}

		// Return the JSON array as text
		return mcp.NewToolResultText(string(tableObjectsJSON)), nil

	case "mysql":
		// Check if schema parameter is provided - not supported for MySQL
		if schemaFilter != "" {
			return nil, fmt.Errorf("schema parameter is not supported for MySQL databases")
		}

		// For MySQL, keyspace parameter is required
		keyspaceArg, ok := args["keyspace"]
		if !ok || keyspaceArg == "" {
			return nil, fmt.Errorf("keyspace parameter is required for MySQL databases")
		}

		// For MySQL, use the existing approach with MySQL connection
		conn, err := createMySQLConnection(ctx, request, ch)
		if err != nil {
			handledErr := cmdutil.HandleError(err)
			if handledErr != err {
				return nil, handledErr
			}
			return nil, err
		}
		defer conn.close()

		// Execute the SHOW TABLES query
		results, err := executeQueryMySQL(ctx, conn, "SHOW TABLES")
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

	default:
		return nil, fmt.Errorf("unsupported database kind: %s. Only 'mysql' and 'postgresql' are supported", dbKind)
	}
}

// HandleGetSchema implements the get_schema tool
func HandleGetSchema(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Get the required database parameter
	dbArg, ok := args["database"]
	if !ok || dbArg == "" {
		return nil, fmt.Errorf("database parameter is required")
	}
	database := dbArg.(string)

	// Get the required tables parameter
	tablesArg, ok := args["tables"]
	if !ok || tablesArg == "" {
		return nil, fmt.Errorf("tables parameter is required")
	}
	tables := tablesArg.(string)

	// Get the organization
	orgName, err := getOrganization(request, ch)
	if err != nil {
		return nil, err
	}

	// Get database kind
	dbKind, err := getDatabaseKind(ctx, ch, orgName, database)
	if err != nil {
		return nil, err
	}

	var schemas interface{}

	// Handle different database types
	switch dbKind {
	case "postgresql", "horizon":
		schemas, err = getPostgreSQLSchemas(ctx, request, ch, tables)
		if err != nil {
			return nil, fmt.Errorf("failed to get PostgreSQL schemas: %w", err)
		}

	case "mysql":
		schemas, err = getMySQLSchemas(ctx, request, ch, tables)
		if err != nil {
			return nil, fmt.Errorf("failed to get MySQL schemas: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported database kind: %s. Only 'mysql' and 'postgresql' are supported", dbKind)
	}

	// Convert to JSON
	schemasJSON, err := json.Marshal(schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schemas: %w", err)
	}

	// Return the JSON object as text
	return mcp.NewToolResultText(string(schemasJSON)), nil
}

// getPostgreSQLSchemas gets schema information for PostgreSQL tables
func getPostgreSQLSchemas(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper, tables string) (map[string][]map[string]interface{}, error) {
	args := request.GetArguments()

	// Get keyspace parameter (database name) with default
	keyspace := "postgres" // default database
	if keyspaceArg, ok := args["keyspace"].(string); ok && keyspaceArg != "" {
		keyspace = keyspaceArg
	}

	// Build WHERE clauses for each part of the UNION query using parameterized queries
	var columnsWhereClause, constraintsWhereClause, indexesWhereClause string
	var params []interface{}

	if tables == "*" {
		// All tables in all schemas (excluding system schemas)
		columnsWhereClause = "table_schema NOT IN ('pg_catalog', 'information_schema')"
		constraintsWhereClause = "tc.table_schema NOT IN ('pg_catalog', 'information_schema')"
		indexesWhereClause = "schemaname NOT IN ('pg_catalog', 'information_schema')"
	} else {
		// Parse comma-separated list and build conditions with parameterized queries
		var columnConditions, constraintConditions, indexConditions []string
		paramIndex := 1

		for _, table := range strings.Split(tables, ",") {
			table = strings.TrimSpace(table)
			if table == "" {
				continue
			}

			if strings.HasSuffix(table, ".*") {
				// Schema wildcard: "myschema.*"
				schemaName := strings.TrimSuffix(table, ".*")
				columnConditions = append(columnConditions, fmt.Sprintf("table_schema = $%d", paramIndex))
				constraintConditions = append(constraintConditions, fmt.Sprintf("tc.table_schema = $%d", paramIndex))
				indexConditions = append(indexConditions, fmt.Sprintf("schemaname = $%d", paramIndex))
				params = append(params, schemaName)
				paramIndex++
			} else if strings.Contains(table, ".") {
				// Qualified table name: "myschema.mytable"
				parts := strings.Split(table, ".")
				if len(parts) == 2 {
					schemaName := parts[0]
					tableName := parts[1]
					columnConditions = append(columnConditions, fmt.Sprintf("(table_schema = $%d AND table_name = $%d)", paramIndex, paramIndex+1))
					constraintConditions = append(constraintConditions, fmt.Sprintf("(tc.table_schema = $%d AND tc.table_name = $%d)", paramIndex, paramIndex+1))
					indexConditions = append(indexConditions, fmt.Sprintf("(schemaname = $%d AND tablename = $%d)", paramIndex, paramIndex+1))
					params = append(params, schemaName, tableName)
					paramIndex += 2
				}
			} else {
				// Unqualified table name: "mytable" -> "public.mytable"
				columnConditions = append(columnConditions, fmt.Sprintf("(table_schema = 'public' AND table_name = $%d)", paramIndex))
				constraintConditions = append(constraintConditions, fmt.Sprintf("(tc.table_schema = 'public' AND tc.table_name = $%d)", paramIndex))
				indexConditions = append(indexConditions, fmt.Sprintf("(schemaname = 'public' AND tablename = $%d)", paramIndex))
				params = append(params, table)
				paramIndex++
			}
		}

		if len(columnConditions) == 0 {
			return map[string][]map[string]interface{}{}, nil
		}

		columnsWhereClause = "(" + strings.Join(columnConditions, " OR ") + ")"
		constraintsWhereClause = "(" + strings.Join(constraintConditions, " OR ") + ")"
		indexesWhereClause = "(" + strings.Join(indexConditions, " OR ") + ")"
	}

	// Build the big query
	query := fmt.Sprintf(`
-- Column details
SELECT 
    table_schema || '.' || table_name as table_key,
    'COLUMN' as type,
    column_name as name,
    data_type,
    character_maximum_length,
    is_nullable,
    column_default as details
FROM information_schema.columns 
WHERE %s

UNION ALL

-- Constraints
SELECT 
    tc.table_schema || '.' || tc.table_name as table_key,
    'CONSTRAINT' as type,
    tc.constraint_name as name,
    tc.constraint_type as data_type,
    NULL::integer as character_maximum_length,
    NULL as is_nullable,
    string_agg(kcu.column_name, ', ') as details
FROM information_schema.table_constraints tc
LEFT JOIN information_schema.key_column_usage kcu 
    ON tc.constraint_name = kcu.constraint_name
WHERE %s
GROUP BY tc.table_schema, tc.table_name, tc.constraint_name, tc.constraint_type

UNION ALL

-- Indexes
SELECT 
    schemaname || '.' || tablename as table_key,
    'INDEX' as type,
    indexname as name,
    'btree' as data_type,
    NULL::integer as character_maximum_length,
    NULL as is_nullable,
    indexdef as details
FROM pg_indexes 
WHERE %s

ORDER BY table_key, type, name;`, columnsWhereClause, constraintsWhereClause, indexesWhereClause)

	results, err := executeQueryPostgres(ctx, request, ch, query, keyspace, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute PostgreSQL schema query: %w", err)
	}

	// Group results by table
	schemas := make(map[string][]map[string]interface{})
	for _, row := range results {
		if tableKey, ok := row["table_key"].(string); ok {
			// Remove table_key from the row data
			rowData := make(map[string]interface{})
			for k, v := range row {
				if k != "table_key" {
					rowData[k] = v
				}
			}
			schemas[tableKey] = append(schemas[tableKey], rowData)
		}
	}

	return schemas, nil
}

// getMySQLSchemas gets schema information for MySQL tables (existing functionality)
func getMySQLSchemas(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper, tables string) (map[string]string, error) {
	// Create a MySQL connection
	conn, err := createMySQLConnection(ctx, request, ch)
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
		results, err := executeQueryMySQL(ctx, conn, "SHOW TABLES")
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
		results, err := executeQueryMySQL(ctx, conn, query)
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

	return schemas, nil
}

// HandleGetInsights implements the get_insights tool
func HandleGetInsights(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Get the required parameters
	dbArg, ok := args["database"]
	if !ok || dbArg == "" {
		return nil, fmt.Errorf("database parameter is required")
	}
	database := dbArg.(string)

	branchArg, ok := args["branch"]
	if !ok || branchArg == "" {
		return nil, fmt.Errorf("branch parameter is required")
	}
	branch := branchArg.(string)

	// Get the organization
	orgName, err := getOrganization(request, ch)
	if err != nil {
		return nil, err
	}

	// Fetch the /insights endpoint several times, to get the top queries by
	// each of these metrics
	fetchMetrics := []string{
		"totalTime",
		"rowsReadPerReturned",
		"rowsRead",
		"p99Latency",
		"rowsAffected",
	}

	// Fields to include in the result
	resultFields := []string{
		"query_count",
		"sum_total_duration_millis",
		"rows_read_per_returned",
		"sum_rows_read",
		"p99_latency",
		"sum_rows_affected",
		"normalized_sql",
		"tables",
		"index_usages",
		"keyspace",
		"last_run_at",
	}

	// Create a set to track unique entries
	uniqueEntries := make(map[string]bool)
	var topEntries []map[string]interface{}

	for _, metric := range fetchMetrics {
		// Construct the API path
		apiPath := fmt.Sprintf("organizations/%s/databases/%s/branches/%s/insights?per_page=5&sort=%s&dir=desc", orgName, database, branch, metric)

		// Build the URL
		urlStr := fmt.Sprintf("%s/v1/%s", ch.Config.BaseURL, apiPath)

		// Create the request
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
		if err != nil {
			return nil, fmt.Errorf("creating HTTP request: %w", err)
		}

		// Add headers
		req.Header.Set("User-Agent", "pscale-cli-mcp")
		req.Header.Set("Accept", "application/json")

		// Create an HTTP client with authentication
		var cl *http.Client
		if ch.Config.AccessToken != "" {
			tok := &oauth2.Token{AccessToken: ch.Config.AccessToken}
			cl = oauth2.NewClient(ctx, oauth2.StaticTokenSource(tok))
		} else if ch.Config.ServiceToken != "" && ch.Config.ServiceTokenID != "" {
			req.Header.Set("Authorization", ch.Config.ServiceTokenID+":"+ch.Config.ServiceToken)
			cl = &http.Client{}
		} else {
			return nil, fmt.Errorf("not authenticated")
		}

		// Send the request
		resp, err := cl.Do(req)
		if err != nil {
			return nil, fmt.Errorf("sending HTTP request: %w", err)
		}
		defer resp.Body.Close()

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading HTTP response body: %w", err)
		}

		// Check for errors (anything above 299 is an error)
		if resp.StatusCode > 299 {
			return nil, fmt.Errorf("HTTP %s: %s", resp.Status, string(body))
		}

		// Parse the JSON response
		var response struct {
			Data []map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("parsing JSON response: %w", err)
		}

		for _, entry := range response.Data {
			id, _ := entry["id"].(string)
			if !uniqueEntries[id] {
				// Create a filtered entry with just the fields we want
				filteredEntry := make(map[string]interface{})
				for _, field := range resultFields {
					if value, ok := entry[field]; ok {
						filteredEntry[field] = value
					}
				}

				topEntries = append(topEntries, filteredEntry)
				uniqueEntries[id] = true
			}
		}
	}

	// Convert to JSON for the response
	resultJSON, err := json.MarshalIndent(topEntries, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling result to JSON: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}
