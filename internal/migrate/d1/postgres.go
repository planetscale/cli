package d1

import (
	"database/sql"

	"github.com/planetscale/cli/internal/postgres"
)

// OpenPostgres opens a PostgreSQL connection.
func OpenPostgres(uri string) (*sql.DB, error) {
	return postgres.OpenConnection(uri)
}
