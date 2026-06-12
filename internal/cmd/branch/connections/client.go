package connections

import (
	"github.com/planetscale/cli/internal/cmdutil"
	live "github.com/planetscale/cli/internal/connections"
)

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
