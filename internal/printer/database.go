package printer

import (
	"time"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

// Database returns a table-serializable database model.
type Database struct {
	Name      string `header:"name" json:"name"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
	Notes     string `header:"notes" json:"notes"`
}

func NewDatabasePrinter(db *ps.Database) *ObjectPrinter {
	return &ObjectPrinter{
		Source:  db,
		Printer: newDatabasePrinter(db),
	}
}

func NewDatabaseSlicePrinter(dbs []*ps.Database) *ObjectPrinter {
	return &ObjectPrinter{
		Source:  dbs,
		Printer: newDatabaseSlicePrinter(dbs),
	}
}

// newDatabaseSlicePrinter returns a slice of printable databases.
func newDatabaseSlicePrinter(databases []*ps.Database) []*Database {
	dbs := make([]*Database, 0, len(databases))

	for _, db := range databases {
		dbs = append(dbs, newDatabasePrinter(db))
	}

	return dbs
}

// newDatabasePrinter returns a struct that prints out the various fields of a
// database model.
func newDatabasePrinter(db *ps.Database) *Database {
	return &Database{
		Name:      db.Name,
		Notes:     db.Notes,
		CreatedAt: db.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt: db.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
	}
}
