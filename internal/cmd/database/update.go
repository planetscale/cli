package database

import (
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// UpdateCmd updates configurable database settings.
func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		newName                    string
		defaultBranch              string
		requireApprovalForDeploy   bool
		restrictBranchRegion       bool
		allowDataBranching         bool
		allowForeignKeyConstraints bool
		automaticMigrations        bool
		migrationFramework         string
		migrationTableName         string
		insightsRawQueries         bool
		productionBranchWebConsole bool
	}

	cmd := &cobra.Command{
		Use:   "update <database>",
		Short: "Update a database's settings",
		Long: `Update a database's settings.

Only flags you pass are sent to the API. Boolean flags must be set explicitly,
for example --insights-raw-queries=true or --insights-raw-queries=false.

Flags marked "Vitess only" are rejected for PostgreSQL databases.`,
		Args: cmdutil.RequiredArgs("database"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]

			req := &ps.UpdateDatabaseSettingsRequest{
				Organization: ch.Config.Organization,
				Database:     name,
			}

			changed := false
			vitessFlags := make([]string, 0, 6)

			if cmd.Flags().Changed("new-name") {
				req.NewName = &flags.newName
				changed = true
			}
			if cmd.Flags().Changed("default-branch") {
				req.DefaultBranch = &flags.defaultBranch
				changed = true
			}
			if cmd.Flags().Changed("restrict-branch-region") {
				req.RestrictBranchRegion = &flags.restrictBranchRegion
				changed = true
			}
			if cmd.Flags().Changed("insights-raw-queries") {
				req.InsightsRawQueries = &flags.insightsRawQueries
				changed = true
			}
			if cmd.Flags().Changed("production-branch-web-console") {
				req.ProductionBranchWebConsole = &flags.productionBranchWebConsole
				changed = true
			}

			if cmd.Flags().Changed("require-approval-for-deploy") {
				req.RequireApprovalForDeploy = &flags.requireApprovalForDeploy
				changed = true
				vitessFlags = append(vitessFlags, "--require-approval-for-deploy")
			}
			if cmd.Flags().Changed("allow-data-branching") {
				req.AllowDataBranching = &flags.allowDataBranching
				changed = true
				vitessFlags = append(vitessFlags, "--allow-data-branching")
			}
			if cmd.Flags().Changed("allow-foreign-key-constraints") {
				req.AllowForeignKeyConstraints = &flags.allowForeignKeyConstraints
				changed = true
				vitessFlags = append(vitessFlags, "--allow-foreign-key-constraints")
			}
			if cmd.Flags().Changed("automatic-migrations") {
				req.AutomaticMigrations = &flags.automaticMigrations
				changed = true
				vitessFlags = append(vitessFlags, "--automatic-migrations")
			}
			if cmd.Flags().Changed("migration-framework") {
				req.MigrationFramework = &flags.migrationFramework
				changed = true
				vitessFlags = append(vitessFlags, "--migration-framework")
			}
			if cmd.Flags().Changed("migration-table-name") {
				req.MigrationTableName = &flags.migrationTableName
				changed = true
				vitessFlags = append(vitessFlags, "--migration-table-name")
			}

			if !changed {
				return fmt.Errorf("at least one settings flag must be provided")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if len(vitessFlags) > 0 {
				existing, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
					Organization: ch.Config.Organization,
					Database:     name,
				})
				if err != nil {
					switch cmdutil.ErrCode(err) {
					case ps.ErrNotFound:
						return fmt.Errorf("database %s does not exist in organization %s",
							printer.BoldBlue(name), printer.BoldBlue(ch.Config.Organization))
					default:
						return cmdutil.HandleError(err)
					}
				}
				if existing.Kind != ps.DatabaseEngineMySQL {
					return fmt.Errorf("%s %s only valid for Vitess (MySQL) databases (database %s is %s)",
						strings.Join(vitessFlags, ", "),
						pluralFlags(len(vitessFlags)),
						printer.BoldBlue(name),
						printer.BoldBlue(string(existing.Kind)))
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating database %s...", printer.BoldBlue(name)))
			defer end()

			database, err := client.Databases.UpdateSettings(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(name), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return printDatabase(ch, database)
		},
	}

	cmd.Flags().StringVar(&flags.newName, "new-name", "", "Rename the database (PostgreSQL and Vitess)")
	cmd.Flags().StringVar(&flags.defaultBranch, "default-branch", "", "The default branch of the database (PostgreSQL and Vitess)")
	cmd.Flags().BoolVar(&flags.restrictBranchRegion, "restrict-branch-region", false, "Limit branch creation to the database region (PostgreSQL and Vitess)")
	cmd.Flags().BoolVar(&flags.insightsRawQueries, "insights-raw-queries", false, "Collect full SQL queries for Insights (PostgreSQL and Vitess)")
	cmd.Flags().BoolVar(&flags.productionBranchWebConsole, "production-branch-web-console", false, "Allow the web console on the production branch (PostgreSQL and Vitess)")

	cmd.Flags().BoolVar(&flags.requireApprovalForDeploy, "require-approval-for-deploy", false, "Require admin approval for deploy requests (Vitess only)")
	cmd.Flags().BoolVar(&flags.allowDataBranching, "allow-data-branching", false, "Allow seeding branches with data (Vitess only)")
	cmd.Flags().BoolVar(&flags.allowForeignKeyConstraints, "allow-foreign-key-constraints", false, "Allow foreign key constraints (Vitess only)")
	cmd.Flags().BoolVar(&flags.automaticMigrations, "automatic-migrations", false, "Copy migration data to new branches and deploy requests (Vitess only)")
	cmd.Flags().StringVar(&flags.migrationFramework, "migration-framework", "", "Migration framework for the database (Vitess only)")
	cmd.Flags().StringVar(&flags.migrationTableName, "migration-table-name", "", "Migration table name for the database (Vitess only)")

	return cmd
}

func pluralFlags(n int) string {
	if n == 1 {
		return "is"
	}
	return "are"
}
