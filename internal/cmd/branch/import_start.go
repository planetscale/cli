package branch

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/postgres"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/roleutil"
)

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

			_, src, err := buildSourceConfig(flags.source, flags.host, flags.port, flags.username, flags.password, flags.sourceDatabase, flags.sslMode)
			if err != nil {
				return err
			}

			srcDB, err := checkPrerequisites(ctx, ch, src)
			if err != nil {
				return err
			}
			defer srcDB.Close()

			checks, err := postgres.RunPreflightChecks(ctx, srcDB)
			if err != nil {
				return fmt.Errorf("pre-flight checks failed: %w", err)
			}

			if err := printPreflightResults(ch, checks); err != nil {
				return err
			}

			timestamp := time.Now().Format("20060102150405")
			pubName := flags.publication
			if pubName == "" {
				pubName = fmt.Sprintf("_planetscale_import_%s", timestamp)
			}
			subName := fmt.Sprintf("_planetscale_import_%s", timestamp)
			roleName := fmt.Sprintf("pscale_import_%s", timestamp)

			includeTables, skipTables := parseTableFilters(flags.includeTables, flags.skipTables, flags.schemas)

			if err := checkForeignKeyDependencies(ctx, ch, srcDB, includeTables, flags.force); err != nil {
				return err
			}

			printStartSummary(ch, src, database, branch, pubName, subName, includeTables, skipTables, flags.dryRun)

			if flags.dryRun {
				ch.Printer.Printf("\n%s Dry run mode - no changes will be made\n", printer.BoldYellow("[DRY RUN]"))
				return nil
			}

			if !flags.force {
				ch.Printer.Printf("\n")
				if err := ch.Printer.ConfirmCommand(branch, "import start", "import"); err != nil {
					return err
				}
			}

			role, dstDB, err := setupDestinationRole(ctx, ch, client, ch.Config.Organization, database, branch, roleName, flags.dbName)
			if err != nil {
				return err
			}
			defer dstDB.Close()

			if err := handleConflictingTables(ctx, ch, dstDB, role, flags.schemas, includeTables, skipTables, flags.force); err != nil {
				return err
			}

			schemas, err := filterSchemas(flags.schemas, flags.excludeSchemas)
			if err != nil {
				return err
			}

			dbName := flags.dbName
			if dbName == "" {
				dbName = "postgres"
			}

			dst := postgres.BuildConnectionString(&postgres.Config{
				Host:     role.Role.AccessHostURL,
				Port:     5432,
				User:     role.Role.Username,
				Password: role.Role.Password,
				Database: dbName,
				SSLMode:  "require",
				Options:  make(map[string]string),
			})

			if err := setupReplication(ctx, ch, srcDB, dstDB, role, src, dst, schemas, includeTables, skipTables, pubName, subName); err != nil {
				return err
			}

			storeCredentials(ch, ch.Config.Organization, database, branch, subName, src, role, pubName, dbName)

			printStartSuccess(ch, database, branch)

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

func buildSourceConfig(sourceURI, host string, port int, username, password, sourceDatabase, sslMode string) (*postgres.Config, string, error) {
	var cfg *postgres.Config
	var err error

	if sourceURI != "" {
		cfg, err = postgres.ParseConnectionURI(sourceURI)
		if err != nil {
			return nil, "", fmt.Errorf("failed to parse source URI: %w", err)
		}
	} else {
		cfg = &postgres.Config{
			Port:    5432,
			SSLMode: "require",
			Options: make(map[string]string),
		}
	}

	if host != "" {
		cfg.Host = host
	}
	if port != 0 {
		cfg.Port = port
	}
	if username != "" {
		cfg.User = username
	}
	if password != "" {
		cfg.Password = password
	}
	if sourceDatabase != "" {
		cfg.Database = sourceDatabase
	}
	if sslMode != "" {
		cfg.SSLMode = sslMode
	}

	if cfg.Host == "" {
		return nil, "", fmt.Errorf("source host is required (use --source or --host)")
	}
	if cfg.User == "" {
		return nil, "", fmt.Errorf("source username is required (use --source or --username)")
	}
	if cfg.Database == "" {
		return nil, "", fmt.Errorf("source database is required (use --source or --source-database)")
	}

	if cfg.Password == "" {
		prompt := &survey.Password{
			Message: "Source database password:",
		}
		if err := survey.AskOne(prompt, &cfg.Password); err != nil {
			return nil, "", fmt.Errorf("failed to read password: %w", err)
		}
	}

	if err := validateSourceConfig(cfg); err != nil {
		return nil, "", err
	}

	return cfg, postgres.BuildConnectionString(cfg), nil
}

func checkPrerequisites(ctx context.Context, ch *cmdutil.Helper, src string) (*sql.DB, error) {
	ch.Printer.Printf("Checking PostgreSQL client tools...\n")
	psqlMajor, psqlMinor, err := postgres.CheckPsqlVersion(10)
	if err != nil {
		return nil, err
	}
	ch.Printer.Printf("  psql version: %d.%d\n", psqlMajor, psqlMinor)

	pgDumpMajor, pgDumpMinor, err := postgres.CheckPgDumpVersion(10)
	if err != nil {
		return nil, err
	}
	ch.Printer.Printf("  pg_dump version: %d.%d\n", pgDumpMajor, pgDumpMinor)

	end := ch.Printer.PrintProgress("Testing source connection...")
	srcDB, err := postgres.OpenConnection(src)
	if err != nil {
		end()
		return nil, fmt.Errorf("failed to connect to source database: %w", err)
	}

	if err := srcDB.PingContext(ctx); err != nil {
		end()
		srcDB.Close()
		return nil, fmt.Errorf("failed to ping source database: %w", err)
	}
	end()
	ch.Printer.Printf("Source connection: %s\n", printer.BoldGreen("OK"))

	return srcDB, nil
}

func printPreflightResults(ch *cmdutil.Helper, checks *postgres.PreflightCheck) error {
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

	if len(checks.Extensions) > 0 {
		ch.Printer.Printf("  extensions: %s\n", strings.Join(checks.Extensions, ", "))
		ch.Printer.Printf("    See available extensions: https://planetscale.com/docs/postgres/extensions\n")
	} else {
		ch.Printer.Printf("  extensions: none\n")
	}

	return nil
}

func parseTableFilters(includeTablesStr, skipTablesStr string, schemas []string) ([]string, []string) {
	var includeTables, skipTables []string

	if skipTablesStr != "" {
		skipTables = strings.Split(skipTablesStr, ",")
		for i := range skipTables {
			skipTables[i] = strings.TrimSpace(skipTables[i])
		}
	}

	if includeTablesStr != "" {
		includeTables = strings.Split(includeTablesStr, ",")
		for i := range includeTables {
			includeTables[i] = strings.TrimSpace(includeTables[i])
		}
	}

	if len(includeTables) > 0 && len(schemas) > 0 {
		for i, table := range includeTables {
			if !strings.Contains(table, ".") {
				includeTables[i] = schemas[0] + "." + table
			}
		}
	}

	if len(skipTables) > 0 && len(schemas) > 0 {
		for i, table := range skipTables {
			if !strings.Contains(table, ".") {
				skipTables[i] = schemas[0] + "." + table
			}
		}
	}

	return includeTables, skipTables
}

func checkForeignKeyDependencies(ctx context.Context, ch *cmdutil.Helper, db *sql.DB, includeTables []string, force bool) error {
	if len(includeTables) == 0 {
		return nil
	}

	end := ch.Printer.PrintProgress("Checking foreign key dependencies...")
	fkDeps, err := postgres.GetForeignKeyDependencies(ctx, db, includeTables)
	end()

	if err != nil {
		ch.Printer.Printf("%s Could not check foreign keys: %v\n", printer.BoldYellow("[WARN]"), err)
		return nil
	}

	if len(fkDeps) == 0 {
		return nil
	}

	importingSet := make(map[string]bool)
	for _, t := range includeTables {
		importingSet[t] = true
	}

	missingTables := make(map[string]bool)
	for _, dep := range fkDeps {
		if !importingSet[dep.ReferencedTable] {
			missingTables[dep.ReferencedTable] = true
		}
	}

	if len(missingTables) == 0 {
		return nil
	}

	var missing []string
	for t := range missingTables {
		missing = append(missing, t)
	}

	ch.Printer.Printf("\n%s Foreign key dependencies detected!\n", printer.BoldYellow("Warning:"))
	ch.Printer.Printf("The tables you're importing reference these tables:\n")
	for _, t := range missing {
		ch.Printer.Printf("  - %s\n", printer.BoldBlue(t))
	}
	ch.Printer.Printf("\nFor best results, import all related tables together in one import.\n")
	ch.Printer.Printf("Example: --include-tables \"%s,%s\"\n\n",
		strings.Join(includeTables, ","), strings.Join(missing, ","))

	if force {
		return nil
	}

	var proceed bool
	prompt := &survey.Confirm{
		Message: "Continue anyway? (May cause replication issues)",
		Default: false,
	}
	if err := survey.AskOne(prompt, &proceed); err != nil {
		return err
	}
	if !proceed {
		return fmt.Errorf("import cancelled")
	}

	return nil
}

func printStartSummary(ch *cmdutil.Helper, src, database, branch, pubName, subName string, includeTables, skipTables []string, dryRun bool) {
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
}

func setupDestinationRole(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, org, database, branch, roleName, dbName string) (*roleutil.Role, *sql.DB, error) {
	end := ch.Printer.PrintProgress("Creating replication role on destination...")
	role, err := roleutil.New(ctx, client, roleutil.Options{
		Organization:   org,
		Database:       database,
		Branch:         branch,
		Name:           roleName,
		TTL:            7 * 24 * time.Hour, // 7 days - imports can take a long time for large databases
		InheritedRoles: []string{"postgres"},
	})
	if err != nil {
		end()
		return nil, nil, fmt.Errorf("failed to create replication role: %w", err)
	}
	end()
	ch.Printer.Printf("Replication role created: %s\n", printer.BoldGreen(roleName))

	if dbName == "" {
		dbName = "postgres"
	}

	dst := postgres.BuildConnectionString(&postgres.Config{
		Host:     role.Role.AccessHostURL,
		Port:     5432,
		User:     role.Role.Username,
		Password: role.Role.Password,
		Database: dbName,
		SSLMode:  "require",
		Options:  make(map[string]string),
	})

	end = ch.Printer.PrintProgress("Testing destination connection...")
	dstDB, err := postgres.OpenConnection(dst)
	if err != nil {
		end()
		role.Cleanup(ctx, "postgres")
		return nil, nil, fmt.Errorf("failed to connect to destination: %w", err)
	}

	if err := dstDB.PingContext(ctx); err != nil {
		end()
		dstDB.Close()
		role.Cleanup(ctx, "postgres")
		return nil, nil, fmt.Errorf("failed to ping destination: %w", err)
	}
	end()
	ch.Printer.Printf("Destination connection: %s\n", printer.BoldGreen("OK"))

	return role, dstDB, nil
}

func handleConflictingTables(ctx context.Context, ch *cmdutil.Helper, db *sql.DB, role *roleutil.Role, schemas, includeTables, skipTables []string, force bool) error {
	end := ch.Printer.PrintProgress("Checking destination for existing tables...")

	var existingTables []string
	for _, schema := range schemas {
		query := `SELECT tablename FROM pg_tables WHERE schemaname = $1 ORDER BY tablename`
		rows, err := db.QueryContext(ctx, query, schema)
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

	var conflictingTables []string
	if len(includeTables) > 0 {
		includeMap := make(map[string]bool)
		for _, t := range includeTables {
			includeMap[t] = true
			if !strings.Contains(t, ".") {
				for _, schema := range schemas {
					includeMap[schema+"."+t] = true
				}
			}
		}
		for _, existing := range existingTables {
			if includeMap[existing] {
				conflictingTables = append(conflictingTables, existing)
			}
		}
	} else {
		skipMap := make(map[string]bool)
		for _, t := range skipTables {
			skipMap[t] = true
			if !strings.Contains(t, ".") {
				for _, schema := range schemas {
					skipMap[schema+"."+t] = true
				}
			}
		}
		for _, existing := range existingTables {
			if !skipMap[existing] {
				conflictingTables = append(conflictingTables, existing)
			}
		}
	}

	if len(conflictingTables) == 0 {
		ch.Printer.Printf("No conflicting tables: %s\n", printer.BoldGreen("OK"))
		return nil
	}

	ch.Printer.Printf("Found %d conflicting table(s): %s\n", len(conflictingTables), printer.BoldYellow(strings.Join(conflictingTables, ", ")))

	if !force {
		var cleanTables bool
		prompt := &survey.Confirm{
			Message: "Drop conflicting tables and continue?",
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

	ch.Printer.Printf("Dropping conflicting tables...\n")
	for _, table := range conflictingTables {
		parts := strings.SplitN(table, ".", 2)
		var quotedTable string
		if len(parts) == 2 {
			quotedTable = postgres.QuoteIdentifier(parts[0]) + "." + postgres.QuoteIdentifier(parts[1])
		} else {
			quotedTable = postgres.QuoteIdentifier(table)
		}

		end = ch.Printer.PrintProgress(fmt.Sprintf("Dropping %s...", table))
		_, err := db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", quotedTable))
		if err != nil {
			end()
			ch.Printer.Printf("%s Failed to drop table %s: %v\n", printer.BoldYellow("[WARN]"), table, err)
		} else {
			end()
		}
	}
	ch.Printer.Printf("Conflicting tables dropped: %s\n", printer.BoldGreen("OK"))

	return nil
}

func filterSchemas(schemas, excludeSchemas []string) ([]string, error) {
	if len(excludeSchemas) == 0 {
		return schemas, nil
	}

	excludeMap := make(map[string]bool)
	for _, s := range excludeSchemas {
		excludeMap[s] = true
	}

	filtered := make([]string, 0, len(schemas))
	for _, s := range schemas {
		if !excludeMap[s] {
			filtered = append(filtered, s)
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no schemas to import after applying filters")
	}

	return filtered, nil
}

func setupReplication(ctx context.Context, ch *cmdutil.Helper, srcDB, dstDB *sql.DB, role *roleutil.Role, src, dst string, schemas, includeTables, skipTables []string, pubName, subName string) error {
	end := ch.Printer.PrintProgress("Creating publication on source...")
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

	end = ch.Printer.PrintProgress("Importing schema...")
	if err := postgres.PipeSchemaImport(ctx, src, dst, schemas, includeTables, skipTables); err != nil {
		end()
		postgres.DropPublication(ctx, srcDB, pubName, true)
		role.Cleanup(ctx, "postgres")
		return fmt.Errorf("failed to import schema: %w", err)
	}
	end()
	ch.Printer.Printf("Schema imported: %s\n", printer.BoldGreen("OK"))

	end = ch.Printer.PrintProgress("Creating replication slot on source...")
	slotName := subName
	if err := postgres.CreateReplicationSlot(ctx, srcDB, slotName); err != nil {
		end()
		postgres.DropPublication(ctx, srcDB, pubName, true)
		role.Cleanup(ctx, "postgres")
		return fmt.Errorf("failed to create replication slot: %w", err)
	}
	end()
	ch.Printer.Printf("Replication slot created: %s\n", printer.BoldGreen(slotName))

	end = ch.Printer.PrintProgress("Creating subscription...")
	subOpts := postgres.SubscriptionOptions{
		Name:             subName,
		SourceConnString: src,
		PublicationName:  pubName,
		CopyData:         true,
		CreateSlot:       false,
		SlotName:         slotName,
		Enabled:          true,
	}
	if err := postgres.CreateSubscription(ctx, dstDB, subOpts); err != nil {
		end()
		postgres.DropReplicationSlot(ctx, srcDB, slotName)
		postgres.DropPublication(ctx, srcDB, pubName, true)
		role.Cleanup(ctx, "postgres")
		return fmt.Errorf("failed to create subscription: %w", err)
	}
	end()
	ch.Printer.Printf("Subscription created: %s\n", printer.BoldGreen(subName))

	return nil
}

func storeCredentials(ch *cmdutil.Helper, org, database, branch, subName, src string, role *roleutil.Role, pubName, dbName string) {
	creds, err := postgres.NewImportCredentials()
	if err != nil {
		ch.Printer.Printf("%s Could not store credentials in keychain: %v\n", printer.BoldYellow("[WARN]"), err)
		return
	}

	if dbName == "" {
		dbName = "postgres"
	}

	err = creds.StoreImportForSubscription(
		org,
		database,
		branch,
		subName,
		&postgres.ImportInfo{
			SourceConnStr:   src,
			RoleID:          role.Role.ID,
			RoleUsername:    role.Role.Username,
			RolePassword:    role.Role.Password,
			RoleHost:        role.Role.AccessHostURL,
			PublicationName: pubName,
			DBName:          dbName,
		},
	)
	if err != nil {
		ch.Printer.Printf("%s Could not store credentials: %v\n", printer.BoldYellow("[WARN]"), err)
	}
}

func printStartSuccess(ch *cmdutil.Helper, database, branch string) {
	ch.Printer.Printf("\n%s Import started successfully!\n\n", printer.BoldGreen("Success!"))
	ch.Printer.Printf("Next steps:\n")
	ch.Printer.Printf("  1. Monitor progress: pscale branch import status %s %s --watch\n", database, branch)
	ch.Printer.Printf("  2. When ready, complete: pscale branch import complete %s %s\n", database, branch)
	ch.Printer.Printf("  3. Or cancel: pscale branch import cancel %s %s\n", database, branch)
}

func validateSourceConfig(cfg *postgres.Config) error {
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
