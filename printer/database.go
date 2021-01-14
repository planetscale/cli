package printer

import (
	"time"

	"github.com/planetscale/planetscale-go"
)

type Database struct {
	Name      string `header:"name"`
	Notes     string `header:"notes"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)"`
}

// DatabasePrinter returns a struct that prints out the various fields of a
// database model.
func NewDatabasePrinter(db *planetscale.Database) *Database {
	return &Database{
		Name:      db.Name,
		Notes:     db.Notes,
		CreatedAt: db.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt: db.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
	}
}
