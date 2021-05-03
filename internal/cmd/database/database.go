package database

import (
	"encoding/json"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

// DatabaseCmd encapsulates the commands for creating a database
func DatabaseCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "database <command>",
		Short:             "Create, read, destroy, and update databases",
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
	cmd.AddCommand(DumpCmd(ch))
	cmd.AddCommand(RestoreCmd(ch))

	return cmd
}

// Databases represents a slice of database's.
type Databases []*Database

// Database returns a table-serializable database model.
type Database struct {
	Name      string `header:"name" json:"name"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
	Notes     string `header:"notes" json:"notes"`

	orig *ps.Database
}

// toDatabase returns a struct that prints out the various fields of a
// database model.
func toDatabase(db *ps.Database) *Database {
	return &Database{
		Name:      db.Name,
		Notes:     db.Notes,
		CreatedAt: db.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt: db.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		orig:      db,
	}
}

// toDatabases returns a slice of printable databases.
func toDatabases(databases []*ps.Database) Databases {
	dbs := make([]*Database, 0, len(databases))

	for _, db := range databases {
		dbs = append(dbs, toDatabase(db))
	}

	return dbs
}

func (d *Database) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func (d *Database) MarshalCSVValue() interface{} {
	return []*Database{d}
}
