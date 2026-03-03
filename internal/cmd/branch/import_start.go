package branch

import (
	"fmt"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/postgres"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/roleutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// ImportStartCmd returns the command for starting a PostgreSQL import.
func ImportStartCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		source         string
		host           string
		port           int
		username       string
		password       string
		sourceDatabase string
		dbName         string
		sslMode        string
		publication    string
		schemas        []string
		excludeSchemas []string
		skipTables     string
		includeTables  string
		timeout        time.Duration
		dryRun         bool
		force          bool
	}

	cmd := &cobra.Command{
		Use:   "start <database> <branch> --source <uri>",
		Short: "Start importing a PostgreSQL database",
		Long: `Start importing a PostgreSQL database into a PlanetScale branch using logical replication.

This command sets up logical replication from an external PostgreSQL database to a
PlanetScale PostgreSQL branch. The import process:

1. Validates the source database connection and permissions
2. Creates a temporary replication role on the destination
3. Creates a publication on the source database
4. Imports the schema using pg_dump
5. Creates a subscription to replicate data

The source database must have:
- wal_level = logical
- Available replication slots
- A user with replication permissions

After starting the import, use 'pscale branch import status' to monitor progress.`,
		Example: `  # Import using a connection URI
  pscale branch import start mydb main --source "postgresql://user:pass@host:5432/db"

  # Import with explicit connection parameters
  pscale branch import start mydb main --host db.example.com --username admin --source-database production

  # Import specific schemas (avoiding Supabase system schemas)
  pscale branch import start mydb main --source "..." --schemas public,app

  # Exclude specific schemas
  pscale branch import start mydb main --source "..." --exclude-schemas auth,storage,realtime

  # Import specific tables only
  pscale branch import start mydb main --source "..." --include-tables "users,orders,products"

  # Dry run to validate without making changes
  pscale branch import start mydb main --source "..." --dry-run`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			// Check if this is a PostgreSQL database
			db, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if db.Kind != "postgresql" {
				return fmt.Errorf("database %s is not a PostgreSQL database (kind: %s). This command only works with PostgreSQL databases",
					printer.BoldBlue(database), db.Kind)
			}

			// Verify branch exists
			_, err = client.PostgresBranches.Get(ctx, &ps.GetPostgresBranchRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s",
						printer.BoldBlue(branch), printer.BoldBlue(database))
				default:
					return cmdutil.HandleError(err)
				}
			}

			// Build source connection config
			var srcCfg *postgres.Config
			if flags.source != "" {
				srcCfg, err = postgres.ParseConnectionURI(flags.source)
				if err != nil {
					return fmt.Errorf("failed to parse source URI: %w", err)
				}
			} else {
				srcCfg = &postgres.Config{
					Port:    5432,
					SSLMode: "require",
					Options: make(map[string]string),
				}
			}

			// Override with explicit flags
			if flags.host != "" {
				srcCfg.Host = flags.host
			}
			if flags.port != 0 {
				srcCfg.Port = flags.port
			}
			if flags.username != "" {
				srcCfg.User = flags.username
			}
			if flags.password != "" {
				srcCfg.Password = flags.password
			}
			if flags.sourceDatabase != "" {
				srcCfg.Database = flags.sourceDatabase
			}
			if flags.sslMode != "" {
				srcCfg.SSLMode = flags.sslMode
			}

			// Validate required fields
			if srcCfg.Host == "" {
				return fmt.Errorf("source host is required (use --source or --host)")
			}
			if srcCfg.User == "" {
				return fmt.Errorf("source username is required (use --source or --username)")
			}
			if srcCfg.Database == "" {
				return fmt.Errorf("source database is required (use --source or --source-database)")
			}

			// Prompt for password if not provided
			if srcCfg.Password == "" {
				prompt := &survey.Password{
					Message: "Source database password:",
				}
				err := survey.AskOne(prompt, &srcCfg.Password)
				if err != nil {
					return fmt.Errorf("failed to read password: %w", err)
				}
			}

			src := postgres.BuildConnectionString(srcCfg)

			// Validate source configuration for common issues
			if err := validateSourceConfig(srcCfg); err != nil {
				return err
			}

			// Check psql/pg_dump availability
			ch.Printer.Printf("Checking PostgreSQL client tools...\n")
			psqlMajor, psqlMinor, err := postgres.CheckPsqlVersion(10)
			if err != nil {
				return err
			}
			ch.Printer.Printf("  psql version: %d.%d\n", psqlMajor, psqlMinor)

			pgDumpMajor, pgDumpMinor, err := postgres.CheckPgDumpVersion(10)
			if err != nil {
				return err
			}
			ch.Printer.Printf("  pg_dump version: %d.%d\n", pgDumpMajor, pgDumpMinor)

			// Test source connection
			end := ch.Printer.PrintProgress("Testing source connection...")
			srcDB, err := postgres.OpenConnection(src)
			if err != nil {
				end()
				return fmt.Errorf("failed to connect to source database: %w", err)
			}
			defer srcDB.Close()

			if err := srcDB.PingContext(ctx); err != nil {
				end()
				return fmt.Errorf("failed to ping source database: %w", err)
			}
			end()
			ch.Printer.Printf("Source connection: %s\n", printer.BoldGreen("OK"))

			// Run pre-flight checks
			end = ch.Printer.PrintProgress("Running pre-flight checks...")
			checks, err := postgres.RunPreflightChecks(ctx, srcDB)
			if err != nil {
				end()
				return fmt.Errorf("pre-flight checks failed: %w", err)
			}
			end()

			// Display pre-flight results
			ch.Printer.Printf("\nPre-flight checks:\n")
			if !checks.WALLevelOK {
				ch.Printer.Printf("  wal_level: %s %s (requires 'logical')\n", checks.WALLevel, printer.BoldRed("[FAIL]"))
				return fmt.Errorf("source database wal_level must be 'logical', currently '%s'", checks.WALLevel)
			}
			ch.Printer.Printf("  wal_level: %s %s\n", checks.WALLevel, printer.BoldGreen("[OK]"))

			if checks.SlotsAvailable == 0 {
				ch.Printer.Printf("  replication slots: none available %s\n", printer.BoldRed("[FAIL]"))
				return fmt.Errorf("no replication slots available (max: %d)", checks.MaxReplicationSlots)
			}
			ch.Printer.Printf("  replication slots: %d available %s\n", checks.SlotsAvailable, printer.BoldGreen("[OK]"))

			if !checks.HasReplicationPermission {
				ch.Printer.Printf("  replication permission: %s\n", printer.BoldYellow("[WARN]"))
				ch.Printer.Printf("    Note: User may not have replication permission\n")
			} else {
				ch.Printer.Printf("  replication permission: %s\n", printer.BoldGreen("[OK]"))
			}

			// Display extensions
			if len(checks.Extensions) > 0 {
				ch.Printer.Printf("  extensions: %s\n", strings.Join(checks.Extensions, ", "))
				ch.Printer.Printf("    See available extensions: https://planetscale.com/docs/postgres/extensions\n")
			} else {
				ch.Printer.Printf("  extensions: none\n")
			}

			// Generate unique names
			timestamp := time.Now().Format("20060102150405")
			pubName := flags.publication
			if pubName == "" {
				pubName = fmt.Sprintf("_planetscale_import_%s", timestamp)
			}
			subName := fmt.Sprintf("_planetscale_import_%s", timestamp)
			roleName := fmt.Sprintf("pscale_import_%s", timestamp)

			// Parse table filters
			var skipTables, includeTables []string
			if flags.skipTables != "" {
				skipTables = strings.Split(flags.skipTables, ",")
				for i := range skipTables {
					skipTables[i] = strings.TrimSpace(skipTables[i])
				}
			}
			if flags.includeTables != "" {
				includeTables = strings.Split(flags.includeTables, ",")
				for i := range includeTables {
					includeTables[i] = strings.TrimSpace(includeTables[i])
				}
			}

			// Show summary
			ch.Printer.Printf("\n%s\n", printer.Bold("Import Summary:"))
			ch.Printer.Printf("  Source: %s\n", postgres.RedactPassword(src))
			ch.Printer.Printf("  Destination: %s/%s (branch: %s)\n", ch.Config.Organization, database, branch)
			ch.Printer.Printf("  Publication: %s\n", pubName)
			ch.Printer.Printf("  Subscription: %s\n", subName)
			if len(includeTables) > 0 {
				ch.Printer.Printf("  Include tables: %s\n", strings.Join(includeTables, ", "))
			}
			if len(skipTables) > 0 {
				ch.Printer.Printf("  Skip tables: %s\n", strings.Join(skipTables, ", "))
			}

			if flags.dryRun {
				ch.Printer.Printf("\n%s Dry run mode - no changes will be made\n", printer.BoldYellow("[DRY RUN]"))
				return nil
			}

			// Confirm
			if !flags.force {
				ch.Printer.Printf("\n")
				if err := ch.Printer.ConfirmCommand(branch, "import start", "import"); err != nil {
					return err
				}
			}

			// Create replication role on destination
			// Note: Import operations require admin privileges (postgres role)
			// Creating subscriptions needs both pg_create_subscription role AND CREATE on database
			end = ch.Printer.PrintProgress("Creating replication role on destination...")
			role, err := roleutil.New(ctx, client, roleutil.Options{
				Organization:   ch.Config.Organization,
				Database:       database,
				Branch:         branch,
				Name:           roleName,
				TTL:            24 * time.Hour,
				InheritedRoles: []string{"postgres"},
			})
			if err != nil {
				end()
				return fmt.Errorf("failed to create replication role: %w", err)
			}
			end()
			ch.Printer.Printf("Replication role created: %s\n", printer.BoldGreen(roleName))

			// Build destination connection string
			dbName := flags.dbName
			if dbName == "" {
				dbName = "postgres"
			}
			dstCfg := &postgres.Config{
				Host:     role.Role.AccessHostURL,
				Port:     5432,
				User:     role.Role.Username,
				Password: role.Role.Password,
				Database: dbName,
				SSLMode:  "require",
				Options:  make(map[string]string),
			}
			dst := postgres.BuildConnectionString(dstCfg)

			// Test destination connection
			end = ch.Printer.PrintProgress("Testing destination connection...")
			dstDB, err := postgres.OpenConnection(dst)
			if err != nil {
				end()
				role.Cleanup(ctx, "postgres")
				return fmt.Errorf("failed to connect to destination: %w", err)
			}
			defer dstDB.Close()

			if err := dstDB.PingContext(ctx); err != nil {
				end()
				role.Cleanup(ctx, "postgres")
				return fmt.Errorf("failed to ping destination: %w", err)
			}
			end()
			ch.Printer.Printf("Destination connection: %s\n", printer.BoldGreen("OK"))

			// Check for existing tables in destination
			end = ch.Printer.PrintProgress("Checking destination for existing tables...")
			var existingTables []string
			for _, schema := range flags.schemas {
				query := `SELECT tablename FROM pg_tables WHERE schemaname = $1 ORDER BY tablename`
				rows, err := dstDB.QueryContext(ctx, query, schema)
				if err != nil {
					end()
					role.Cleanup(ctx, "postgres")
					return fmt.Errorf("failed to check existing tables: %w", err)
				}
				defer rows.Close()

				for rows.Next() {
					var tableName string
					if err := rows.Scan(&tableName); err != nil {
						end()
						role.Cleanup(ctx, "postgres")
						return fmt.Errorf("failed to scan table name: %w", err)
					}
					existingTables = append(existingTables, schema+"."+tableName)
				}
			}
			end()

			if len(existingTables) > 0 {
				ch.Printer.Printf("Found %d existing table(s): %s\n", len(existingTables), printer.BoldYellow(strings.Join(existingTables, ", ")))

				if !flags.force {
					// Prompt user what to do
					var cleanTables bool
					prompt := &survey.Confirm{
						Message: "Drop existing tables and continue?",
						Default: false,
					}
					if err := survey.AskOne(prompt, &cleanTables); err != nil {
						role.Cleanup(ctx, "postgres")
						return fmt.Errorf("prompt failed: %w", err)
					}
					if !cleanTables {
						role.Cleanup(ctx, "postgres")
						return fmt.Errorf("import cancelled")
					}
				}

				// Drop existing tables
				ch.Printer.Printf("Dropping existing tables...\n")
				for _, table := range existingTables {
					parts := strings.SplitN(table, ".", 2)
					var quotedTable string
					if len(parts) == 2 {
						quotedTable = postgres.QuoteIdentifier(parts[0]) + "." + postgres.QuoteIdentifier(parts[1])
					} else {
						quotedTable = postgres.QuoteIdentifier(table)
					}

					end = ch.Printer.PrintProgress(fmt.Sprintf("Dropping %s...", table))
					_, err := dstDB.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", quotedTable))
					if err != nil {
						end()
						ch.Printer.Printf("%s Failed to drop table %s: %v\n", printer.BoldYellow("[WARN]"), table, err)
					} else {
						end()
					}
				}
				ch.Printer.Printf("Existing tables dropped: %s\n", printer.BoldGreen("OK"))
			} else {
				ch.Printer.Printf("Destination is empty: %s\n", printer.BoldGreen("OK"))
			}

			// Filter schemas based on include/exclude flags
			schemas := flags.schemas
			if len(flags.excludeSchemas) > 0 {
				excludeMap := make(map[string]bool)
				for _, s := range flags.excludeSchemas {
					excludeMap[s] = true
				}
				filtered := make([]string, 0, len(schemas))
				for _, s := range schemas {
					if !excludeMap[s] {
						filtered = append(filtered, s)
					}
				}
				schemas = filtered
			}

			if len(schemas) == 0 {
				return fmt.Errorf("no schemas to import after applying filters")
			}

			// Create publication on source
			end = ch.Printer.PrintProgress("Creating publication on source...")
			pubOpts := postgres.PublicationOptions{
				Name:      pubName,
				AllTables: len(includeTables) == 0,
				Schemas:   schemas,
				Tables:    includeTables,
			}
			if err := postgres.CreatePublication(ctx, srcDB, pubOpts); err != nil {
				end()
				role.Cleanup(ctx, "postgres")
				return fmt.Errorf("failed to create publication: %w", err)
			}
			end()
			ch.Printer.Printf("Publication created: %s\n", printer.BoldGreen(pubName))

			// Import schema via pg_dump | psql
			end = ch.Printer.PrintProgress("Importing schema...")
			if err := postgres.PipeSchemaImport(ctx, src, dst, schemas, skipTables); err != nil {
				end()
				// Try to clean up publication
				postgres.DropPublication(ctx, srcDB, pubName, true)
				role.Cleanup(ctx, "postgres")
				return fmt.Errorf("failed to import schema: %w", err)
			}
			end()
			ch.Printer.Printf("Schema imported: %s\n", printer.BoldGreen("OK"))

			// Create replication slot on source (manually, as some providers don't support automatic creation)
			end = ch.Printer.PrintProgress("Creating replication slot on source...")
			slotName := subName
			if err := postgres.CreateReplicationSlot(ctx, srcDB, slotName); err != nil {
				end()
				// Try to clean up
				postgres.DropPublication(ctx, srcDB, pubName, true)
				role.Cleanup(ctx, "postgres")
				return fmt.Errorf("failed to create replication slot: %w", err)
			}
			end()
			ch.Printer.Printf("Replication slot created: %s\n", printer.BoldGreen(slotName))

			// Create subscription on destination
			end = ch.Printer.PrintProgress("Creating subscription...")
			subOpts := postgres.SubscriptionOptions{
				Name:             subName,
				SourceConnString: src,
				PublicationName:  pubName,
				CopyData:         true,
				CreateSlot:       false, // Already created manually above
				SlotName:         slotName,
				Enabled:          true,
			}
			if err := postgres.CreateSubscription(ctx, dstDB, subOpts); err != nil {
				end()
				// Try to clean up
				postgres.DropReplicationSlot(ctx, srcDB, slotName)
				postgres.DropPublication(ctx, srcDB, pubName, true)
				role.Cleanup(ctx, "postgres")
				return fmt.Errorf("failed to create subscription: %w", err)
			}
			end()
			ch.Printer.Printf("Subscription created: %s\n", printer.BoldGreen(subName))

			// Store credentials for later use
			creds, err := postgres.NewImportCredentials()
			if err != nil {
				ch.Printer.Printf("%s Could not store credentials in keychain: %v\n", printer.BoldYellow("[WARN]"), err)
			} else {
				creds.StoreSourceCredentials(ch.Config.Organization, database, branch, src)
				creds.StoreRoleID(ch.Config.Organization, database, branch, role.Role.ID)
				creds.StoreRoleCredentials(ch.Config.Organization, database, branch, role.Role.Username, role.Role.Password, role.Role.AccessHostURL)
				creds.StorePublicationName(ch.Config.Organization, database, branch, pubName)
				creds.StoreSubscriptionName(ch.Config.Organization, database, branch, subName)
				creds.StoreDBName(ch.Config.Organization, database, branch, dbName)
			}

			ch.Printer.Printf("\n%s Import started successfully!\n\n", printer.BoldGreen("Success!"))
			ch.Printer.Printf("Next steps:\n")
			ch.Printer.Printf("  1. Monitor progress: pscale branch import status %s %s --watch\n", database, branch)
			ch.Printer.Printf("  2. When ready, complete: pscale branch import complete %s %s\n", database, branch)
			ch.Printer.Printf("  3. Or cancel: pscale branch import cancel %s %s\n", database, branch)

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.source, "source", "", "Source PostgreSQL connection URI")
	cmd.Flags().StringVar(&flags.host, "host", "", "Source PostgreSQL hostname")
	cmd.Flags().IntVar(&flags.port, "port", 0, "Source PostgreSQL port (default: 5432)")
	cmd.Flags().StringVar(&flags.username, "username", "", "Source PostgreSQL username")
	cmd.Flags().StringVar(&flags.password, "password", "", "Source PostgreSQL password")
	cmd.Flags().StringVar(&flags.sourceDatabase, "source-database", "", "Source database name")
	cmd.Flags().StringSliceVar(&flags.schemas, "schemas", []string{"public"}, "Schemas to import (comma-separated, default: public)")
	cmd.Flags().StringSliceVar(&flags.excludeSchemas, "exclude-schemas", []string{}, "Schemas to exclude (comma-separated)")
	cmd.Flags().StringVar(&flags.dbName, "dbname", "postgres", "PostgreSQL database name on destination (default: postgres)")
	cmd.Flags().StringVar(&flags.sslMode, "ssl-mode", "", "Source SSL mode (disable, require, verify-ca, verify-full)")
	cmd.Flags().StringVar(&flags.publication, "publication", "", "Custom publication name")
	cmd.Flags().StringVar(&flags.skipTables, "skip-tables", "", "Comma-separated list of tables to exclude")
	cmd.Flags().StringVar(&flags.includeTables, "include-tables", "", "Comma-separated list of tables to include (default: all)")
	cmd.Flags().DurationVar(&flags.timeout, "timeout", 30*time.Second, "Connection timeout")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Validate without making changes")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Skip confirmation prompt")

	return cmd
}

// validateSourceConfig checks for common misconfigurations with PostgreSQL providers.
func validateSourceConfig(cfg *postgres.Config) error {
	// Check for IPv6 addresses
	if strings.Contains(cfg.Host, ":") && !strings.Contains(cfg.Host, ".") {
		return fmt.Errorf(
			"source host appears to be an IPv6 address (%s).\n\n"+
				"If you're experiencing connection issues:\n"+
				"  1. Check your network supports IPv6 routing\n"+
				"  2. Try using your provider's IPv4 connection string if available\n\n"+
				"Note: This is separate from connection pooling. Make sure you're also using\n"+
				"a direct database connection (NOT a pooled connection).",
			cfg.Host)
	}

	// Check for connection pooler usage
	// 1. Port 6543 is commonly used by pgBouncer
	// 2. Some providers use specific hostname patterns for poolers
	hostLower := strings.ToLower(cfg.Host)
	isPoolerPort := cfg.Port == 6543
	isPoolerHostname := strings.HasPrefix(hostLower, "pooler-") ||
		strings.HasPrefix(hostLower, "pooler.") ||
		strings.Contains(hostLower, "-pooler.")

	if isPoolerPort || isPoolerHostname {
		return fmt.Errorf(
			"source appears to be using a connection pooler (port: %d, host: %s).\n\n"+
				"Logical replication does NOT work through connection poolers (pgBouncer, PgCat, etc).\n"+
				"You must use a direct database connection instead.\n\n"+
				"How to get a direct connection:\n\n"+
				"Supabase:\n"+
				"  1. Go to Project Settings > Database\n"+
				"  2. Find 'Direct connection' section (NOT 'Connection pooling')\n"+
				"  3. Use the connection string with port 5432\n\n"+
				"Other providers:\n"+
				"  Look for connection strings labeled as:\n"+
				"  - 'Direct connection' or 'Direct access'\n"+
				"  - Port 5432 (standard PostgreSQL port)\n"+
				"  - NOT labeled as 'pooled' or 'connection pooling'",
			cfg.Port, cfg.Host)
	}

	return nil
}
