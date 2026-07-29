package connections

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	live "github.com/planetscale/cli/internal/connections"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// DatabaseEngine resolves the engine backing a database so connection commands
// can pick the Vitess or Postgres code path.
func DatabaseEngine(ctx context.Context, ch *cmdutil.Helper, database string) (ps.DatabaseEngine, error) {
	client, err := ch.Client()
	if err != nil {
		return "", err
	}

	db, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
		Organization: ch.Config.Organization,
		Database:     database,
	})
	if err != nil {
		if cmdutil.ErrCode(err) == ps.ErrNotFound {
			return "", databaseNotFoundError(ch, database)
		}
		return "", cmdutil.HandleError(err)
	}
	if db == nil {
		return "", databaseNotFoundError(ch, database)
	}
	return db.Kind, nil
}

func databaseNotFoundError(ch *cmdutil.Helper, database string) error {
	return fmt.Errorf("database %s does not exist in organization %s",
		printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
}

func newConnectionsClient(ch *cmdutil.Helper, database, branch string, target ConnectionTarget) (*live.Client, error) {
	return live.NewClient(live.ClientConfig{
		BaseURL:        ch.Config.BaseURL,
		Organization:   ch.Config.Organization,
		Database:       database,
		Branch:         branch,
		Keyspace:       target.Keyspace,
		Shard:          target.Shard,
		AccessToken:    ch.Config.AccessToken,
		ServiceTokenID: ch.Config.ServiceTokenID,
		ServiceToken:   ch.Config.ServiceToken,
	})
}
