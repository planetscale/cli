package connections

import (
	"context"
	"errors"
	"fmt"
	"sync"

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

// notFoundExplainer replaces the connections endpoint's ambiguous 404 with
// whether the branch is missing or just lacks live connections. The answer is
// cached so a refreshing TUI does not repeat the lookups every interval.
type notFoundExplainer struct {
	ch       *cmdutil.Helper
	database string
	branch   string

	once      sync.Once
	explained error
}

func newNotFoundExplainer(ch *cmdutil.Helper, database, branch string) *notFoundExplainer {
	return &notFoundExplainer{ch: ch, database: database, branch: branch}
}

func (e *notFoundExplainer) explain(ctx context.Context, err error) error {
	var notFound *live.NotFoundError
	if e == nil || !errors.As(err, &notFound) {
		return err
	}
	e.once.Do(func() {
		e.explained = e.resolve(ctx, err)
	})
	return e.explained
}

func (e *notFoundExplainer) resolve(ctx context.Context, err error) error {
	client, clientErr := e.ch.Client()
	if clientErr != nil {
		return err
	}
	org := e.ch.Config.Organization
	db, dbErr := client.Databases.Get(ctx, &ps.GetDatabaseRequest{Organization: org, Database: e.database})
	if cmdutil.ErrCode(dbErr) == ps.ErrNotFound {
		return databaseNotFoundError(e.ch, e.database)
	}
	if dbErr != nil || db == nil {
		return err
	}

	var branchErr error
	if db.Kind == ps.DatabaseEngineMySQL {
		_, branchErr = client.DatabaseBranches.Get(ctx, &ps.GetDatabaseBranchRequest{Organization: org, Database: e.database, Branch: e.branch})
	} else {
		_, branchErr = client.PostgresBranches.Get(ctx, &ps.GetPostgresBranchRequest{Organization: org, Database: e.database, Branch: e.branch})
	}
	switch {
	case cmdutil.ErrCode(branchErr) == ps.ErrNotFound:
		return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
			printer.BoldBlue(e.branch), printer.BoldBlue(e.database), printer.BoldBlue(org))
	case branchErr != nil:
		return err
	default:
		return fmt.Errorf("live connections are not enabled for branch %s in database %s (organization: %s)",
			printer.BoldBlue(e.branch), printer.BoldBlue(e.database), printer.BoldBlue(org))
	}
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
