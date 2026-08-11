package cmdutil

import (
	"context"
	"fmt"

	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// RequirePostgresDatabase fetches the database and errors if it is missing or
// not PostgreSQL. Use this before calling Postgres-only APIs that return a
// generic not_found for Vitess/MySQL databases.
func RequirePostgresDatabase(ctx context.Context, client *ps.Client, org, database, resourcePlural string) error {
	db, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
		Organization: org,
		Database:     database,
	})
	if err != nil {
		switch ErrCode(err) {
		case ps.ErrNotFound:
			return fmt.Errorf("database %s does not exist in organization %s",
				printer.BoldBlue(database), printer.BoldBlue(org))
		default:
			return HandleError(err)
		}
	}
	if db.Kind != ps.DatabaseEnginePostgres {
		return fmt.Errorf("%s are only available for PostgreSQL databases; %s is %s",
			resourcePlural, printer.BoldBlue(database), printer.BoldBlue(string(db.Kind)))
	}
	return nil
}
