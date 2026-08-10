package database

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// SettingsCmd shows database settings.
func SettingsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settings <database>",
		Short: "Show the settings for a database",
		Args:  cmdutil.RequiredArgs("database"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching settings for database %s...", printer.BoldBlue(name)))
			defer end()

			database, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
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
			end()

			return ch.Printer.PrintResource(toDatabaseSettings(database))
		},
	}

	return cmd
}

// DatabaseSettings is a table-serializable view of configurable database settings.
type DatabaseSettings struct {
	Name                       string `header:"name" json:"name"`
	Kind                       string `header:"kind" json:"kind"`
	DefaultBranch              string `header:"default_branch" json:"default_branch"`
	RequireApprovalForDeploy   bool   `header:"require_approval_for_deploy" json:"require_approval_for_deploy"`
	RestrictBranchRegion       bool   `header:"restrict_branch_region" json:"restrict_branch_region"`
	AllowDataBranching         bool   `header:"allow_data_branching" json:"allow_data_branching"`
	ForeignKeysEnabled         bool   `header:"foreign_keys_enabled" json:"foreign_keys_enabled"`
	AutomaticMigrations        string `header:"automatic_migrations" json:"automatic_migrations"`
	MigrationFramework         string `header:"migration_framework" json:"migration_framework"`
	MigrationTableName         string `header:"migration_table_name" json:"migration_table_name"`
	InsightsRawQueries         bool   `header:"insights_raw_queries" json:"insights_raw_queries"`
	InsightsEnabled            bool   `header:"insights_enabled" json:"insights_enabled"`
	ProductionBranchWebConsole bool   `header:"production_branch_web_console" json:"production_branch_web_console"`

	orig *ps.Database
}

func toDatabaseSettings(db *ps.Database) *DatabaseSettings {
	settings := &DatabaseSettings{
		Name:                       db.Name,
		Kind:                       string(db.Kind),
		DefaultBranch:              db.DefaultBranch,
		RequireApprovalForDeploy:   db.RequireApprovalForDeploy,
		RestrictBranchRegion:       db.RestrictBranchRegion,
		AllowDataBranching:         db.AllowDataBranching,
		ForeignKeysEnabled:         db.ForeignKeysEnabled,
		InsightsRawQueries:         db.InsightsRawQueries,
		InsightsEnabled:            db.InsightsEnabled,
		ProductionBranchWebConsole: db.ProductionBranchWebConsole,
		orig:                       db,
	}

	if db.AutomaticMigrations != nil {
		settings.AutomaticMigrations = fmt.Sprintf("%t", *db.AutomaticMigrations)
	}
	if db.MigrationFramework != nil {
		settings.MigrationFramework = *db.MigrationFramework
	}
	if db.MigrationTableName != nil {
		settings.MigrationTableName = *db.MigrationTableName
	}

	return settings
}

func (d *DatabaseSettings) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func (d *DatabaseSettings) MarshalCSVValue() interface{} {
	return []*DatabaseSettings{d}
}
