package printer

import (
	"time"

	"github.com/planetscale/planetscale-go"
)

type Database struct {
	Name      string     `header:"name"`
	Notes     string     `header:"notes"`
	CreatedAt *time.Time `header:"created_at,timestamp"`
	UpdatedAt *time.Time `header:"updated_at,timestamp"`
}

// DatabasePrinter returns a struct that prints out the various fields of a
// database model.
func NewDatabasePrinter(db *planetscale.Database) *Database {
	return &Database{
		Name:      db.Name,
		Notes:     db.Notes,
		CreatedAt: &db.CreatedAt,
		UpdatedAt: &db.UpdatedAt,
	}
}
