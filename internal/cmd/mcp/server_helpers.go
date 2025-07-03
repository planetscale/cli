package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/passwordutil"
	"github.com/planetscale/cli/internal/proxyutil"
	"github.com/planetscale/planetscale-go/planetscale"
	"vitess.io/vitess/go/mysql"
)

// DatabaseConnection represents a connection to a PlanetScale database
type DatabaseConnection struct {
	db        *sql.DB
	proxy     proxyutil.Proxy
	listener  net.Listener
	password  *passwordutil.Password
	keyspace  string
	localAddr string
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

// createDatabaseConnection establishes a connection to a PlanetScale database
// It extracts all required parameters (org, database, branch, keyspace) from the request
func createDatabaseConnection(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*DatabaseConnection, error) {
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
