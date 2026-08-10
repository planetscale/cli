package database

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"

	ps "github.com/planetscale/cli/internal/planetscale"
)

// DatabaseCmd encapsulates the commands for creating a database
func DatabaseCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "database <command>",
		Short:             "Create, read, update, delete, and dump/restore databases",
		Aliases:           []string{"db"},
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(DeleteCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(UpdateCmd(ch))
	cmd.AddCommand(DumpCmd(ch))
	cmd.AddCommand(RestoreCmd(ch))

	return cmd
}

// Databases represents a slice of database list rows.
type Databases []*databaseListItem

// databaseListItem is the slim human table for `database list`.
type databaseListItem struct {
	Name      string `header:"name" json:"name"`
	Kind      string `header:"kind" json:"kind"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.Database
}

func (d *databaseListItem) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func (d *databaseListItem) MarshalCSVValue() interface{} {
	return []*databaseListItem{d}
}

// Database is the human table for show/create/update, including settings.
type Database struct {
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
	CreatedAt                  int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt                  int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.Database
}

// toDatabase returns a struct that prints out the various fields of a
// database model, including settings.
func toDatabase(db *ps.Database) *Database {
	out := &Database{
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
		CreatedAt:                  db.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt:                  db.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		orig:                       db,
	}

	if db.AutomaticMigrations != nil {
		out.AutomaticMigrations = fmt.Sprintf("%t", *db.AutomaticMigrations)
	}
	if db.MigrationFramework != nil {
		out.MigrationFramework = *db.MigrationFramework
	}
	if db.MigrationTableName != nil {
		out.MigrationTableName = *db.MigrationTableName
	}

	return out
}

// toDatabases returns a slice of printable list rows.
func toDatabases(databases []*ps.Database) Databases {
	dbs := make([]*databaseListItem, 0, len(databases))

	for _, db := range databases {
		dbs = append(dbs, &databaseListItem{
			Name:      db.Name,
			Kind:      string(db.Kind),
			CreatedAt: db.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
			UpdatedAt: db.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
			orig:      db,
		})
	}

	return dbs
}

func (d *Database) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func (d *Database) MarshalCSVValue() interface{} {
	return []*Database{d}
}

// printDatabase prints a database in the configured format. Human output uses a
// vertical key/value layout so settings remain readable.
func printDatabase(ch *cmdutil.Helper, db *ps.Database) error {
	view := toDatabase(db)
	if ch.Printer.Format() != printer.Human {
		return ch.Printer.PrintResource(view)
	}

	printDatabaseHuman(ch.Printer, view)
	return nil
}

func printDatabaseHuman(p *printer.Printer, db *Database) {
	vitess := db.Kind == string(ps.DatabaseEngineMySQL)

	p.Printf("%-32s %s\n", "Name", db.Name)
	p.Printf("%-32s %s\n", "Kind", db.Kind)
	p.Printf("%-32s %s\n", "Default Branch", db.DefaultBranch)
	p.Printf("%-32s %t\n", "Restrict Branch Region", db.RestrictBranchRegion)
	p.Printf("%-32s %t\n", "Insights Enabled", db.InsightsEnabled)
	p.Printf("%-32s %t\n", "Production Branch Web Console", db.ProductionBranchWebConsole)

	if vitess {
		p.Printf("%-32s %t\n", "Insights Raw Queries", db.InsightsRawQueries)
		p.Printf("%-32s %t\n", "Require Approval For Deploy", db.RequireApprovalForDeploy)
		p.Printf("%-32s %t\n", "Allow Data Branching", db.AllowDataBranching)
		p.Printf("%-32s %t\n", "Foreign Keys Enabled", db.ForeignKeysEnabled)
		p.Printf("%-32s %s\n", "Automatic Migrations", emptyAsDash(db.AutomaticMigrations))
		p.Printf("%-32s %s\n", "Migration Framework", emptyAsDash(db.MigrationFramework))
		p.Printf("%-32s %s\n", "Migration Table Name", emptyAsDash(db.MigrationTableName))
	}

	p.Printf("%-32s %s\n", "Created At", formatUnixMilli(db.CreatedAt))
	p.Printf("%-32s %s\n", "Updated At", formatUnixMilli(db.UpdatedAt))
}

func emptyAsDash(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

func formatUnixMilli(ms int64) string {
	if ms == 0 {
		return "-"
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}
