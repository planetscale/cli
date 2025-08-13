package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/passwordutil"
	"github.com/planetscale/cli/internal/proxyutil"
	"github.com/planetscale/cli/internal/roleutil"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/planetscale/psdbproxy"
	"vitess.io/vitess/go/mysql"
)

// DatabaseConnection represents a connection to a PlanetScale database
type DatabaseConnection struct {
	db        *sql.DB
	proxy     *psdbproxy.Server
	listener  net.Listener
	password  *passwordutil.Password
	keyspace  string
	localAddr string
}

// PostgresConnection represents a connection to a PostgreSQL database
type PostgresConnection struct {
	db   *sql.DB
	role *roleutil.Role
}

// getOrganization extracts the organization from the request parameters or falls back to defaults
func getOrganization(request mcp.CallToolRequest, ch *cmdutil.Helper) (string, error) {
	args := request.GetArguments()

	// Get the organization from the parameters or use the default
	var orgName string
	if org, ok := args["org"].(string); ok && org != "" {
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

// getDatabaseKind returns the kind of a database (e.g. "mysql", "postgresql") for the given organization and database name
func getDatabaseKind(ctx context.Context, ch *cmdutil.Helper, orgName, database string) (string, error) {
	// Get the PlanetScale client
	client, err := ch.Client()
	if err != nil {
		return "", fmt.Errorf("failed to initialize PlanetScale client: %w", err)
	}

	// Get database info to determine the database kind
	dbInfo, err := client.Databases.Get(ctx, &planetscale.GetDatabaseRequest{
		Organization: orgName,
		Database:     database,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case planetscale.ErrNotFound:
			return "", fmt.Errorf("database %s does not exist in organization %s", database, orgName)
		default:
			handledErr := cmdutil.HandleError(err)
			if handledErr != err {
				return "", handledErr
			}
			return "", fmt.Errorf("failed to get database info: %w", err)
		}
	}

	return string(dbInfo.Kind), nil
}

// createMySQLConnection establishes a connection to a PlanetScale MySQL database
// It extracts all required parameters (org, database, branch, keyspace) from the request
func createMySQLConnection(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*DatabaseConnection, error) {
	// Get the PlanetScale client
	client, err := ch.Client()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PlanetScale client: %w", err)
	}

	args := request.GetArguments()

	// Extract the required database parameter
	dbArg, ok := args["database"]
	if !ok || dbArg == "" {
		return nil, fmt.Errorf("database parameter is required")
	}
	database := dbArg.(string)

	// Extract the required branch parameter
	branchArg, ok := args["branch"]
	if !ok || branchArg == "" {
		return nil, fmt.Errorf("branch parameter is required")
	}
	branch := branchArg.(string)

	// Extract the required keyspace parameter
	keyspaceArg, ok := args["keyspace"]
	if !ok || keyspaceArg == "" {
		return nil, fmt.Errorf("keyspace parameter is required")
	}
	keyspace := keyspaceArg.(string)

	// Get the organization
	orgName, err := getOrganization(request, ch)
	if err != nil {
		return nil, err
	}

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
			handledErr := cmdutil.HandleError(err)
			if handledErr != err {
				return nil, handledErr
			}
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
		Replica:      dbBranch.Production, // Use replica if branch is production
	})
	if err != nil {
		handledErr := cmdutil.HandleError(err)
		if handledErr != err {
			return nil, handledErr
		}
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

// executeQueryMySQL executes a SQL query against a MySQL database and returns the results as an array of maps
func executeQueryMySQL(ctx context.Context, conn *DatabaseConnection, query string) ([]map[string]interface{}, error) {
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

// close closes the PostgreSQL connection and cleans up the role
func (c *PostgresConnection) close() {
	if c.db != nil {
		c.db.Close()
	}
	if c.role != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.role.Cleanup(ctx, ""); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("failed to cleanup Postgres role: %v\n", err)
		}
	}
}

// executeQueryPostgres executes a SQL query against a PostgreSQL database using an ephemeral read-only role with a specific keyspace/database
func executeQueryPostgres(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper, query string, keyspace string, params ...interface{}) ([]map[string]interface{}, error) {
	// Get the PlanetScale client
	client, err := ch.Client()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PlanetScale client: %w", err)
	}

	args := request.GetArguments()

	// Extract the required database parameter
	dbArg, ok := args["database"]
	if !ok || dbArg == "" {
		return nil, fmt.Errorf("database parameter is required")
	}
	database := dbArg.(string)

	// Extract the required branch parameter
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
			handledErr := cmdutil.HandleError(err)
			if handledErr != err {
				return nil, handledErr
			}
			return nil, fmt.Errorf("failed to get database branch: %w", err)
		}
	}

	if !dbBranch.Ready {
		return nil, fmt.Errorf("database branch is not ready yet")
	}

	// Create a temporary role for Postgres with read-only access
	pgRole, err := roleutil.New(ctx, client, roleutil.Options{
		Organization:   orgName,
		Database:       database,
		Branch:         branch,
		Name:           passwordutil.GenerateName("pscale-cli-mcp-query"),
		TTL:            5 * time.Minute,
		InheritedRoles: []string{"pg_read_all_data"}, // Read-only role
	})
	if err != nil {
		handledErr := cmdutil.HandleError(err)
		if handledErr != err {
			return nil, handledErr
		}
		return nil, fmt.Errorf("failed to create temporary Postgres role: %w", err)
	}

	// Create PostgreSQL connection
	conn := &PostgresConnection{
		role: pgRole,
	}
	defer conn.close()

	// Get connection details
	remoteHost, remotePort, err := net.SplitHostPort(pgRole.Role.AccessHostURL)
	if err != nil {
		// If no port specified, default to 5432
		remoteHost = pgRole.Role.AccessHostURL
		remotePort = "5432"
	}

	// Create PostgreSQL connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		remoteHost, remotePort, pgRole.Role.Username, pgRole.Role.Password, keyspace)

	// Open PostgreSQL connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}
	conn.db = db

	// Set connection timeout
	db.SetConnMaxLifetime(30 * time.Second)

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	// Execute the query with optional parameters
	rows, err := db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute PostgreSQL query: %w", err)
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
		return nil, fmt.Errorf("error while iterating over PostgreSQL query results: %w", err)
	}

	return results, nil
}
