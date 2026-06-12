package branch

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/planetscale/cli/internal/cmd/branch/connections"
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ConnectionsShowCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
		shard    string
		instance string
		role     string
	}

	cmd := &cobra.Command{
		Use:   "show <database> <branch>",
		Short: "Show branch connections once",
		Long: `Show branch connections once.

Use --format json when an agent or script needs to inspect query_id,
transaction_id, and connection_id fields. Human output uses vertical records so
query text and action IDs are not truncated.`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := connections.ConnectionFilter{Instance: flags.instance, Role: flags.role}
			if err := connections.ValidateConnectionFilter(filter); err != nil {
				return err
			}
			engine, err := databaseEngine(cmd.Context(), ch, args[0])
			if err != nil {
				return err
			}
			target := connections.ConnectionTarget{Keyspace: flags.keyspace, Shard: flags.shard}
			if err := connections.ValidateEngineFlags(engine, filter, target); err != nil {
				return err
			}
			return connections.RunList(cmd.Context(), ch, args[0], args[1], filter, target)
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Vitess keyspace to target")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Vitess shard to target")
	cmd.Flags().StringVar(&flags.instance, "instance", "", "Postgres instance to target")
	cmd.Flags().StringVar(&flags.role, "role", "", "Postgres instance role to target: primary or replica")

	return cmd
}

func ConnectionsKillCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
		shard    string
		query    bool
	}

	cmd := &cobra.Command{
		Use:   "kill <database> <branch> <id>",
		Short: "Kill a branch connection or query",
		Long: `Kill a branch connection or query.

This is destructive. Pass a connection_id from connections show to terminate a
connection, or pass --query with a query_id from connections show to cancel only
the current query.`,
		Args: cmdutil.RequiredArgs("database", "branch", "id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.query {
				if err := connections.ValidateQueryID(args[2]); err != nil {
					return err
				}
			} else {
				if err := connections.ValidateConnectionID(args[2]); err != nil {
					return err
				}
			}
			engine, err := databaseEngine(cmd.Context(), ch, args[0])
			if err != nil {
				return err
			}
			target := connections.ConnectionTarget{Keyspace: flags.keyspace, Shard: flags.shard}
			if err := connections.ValidateEngineFlags(engine, connections.ConnectionFilter{}, target); err != nil {
				return err
			}
			if flags.query {
				return connections.RunCancelQueryForEngine(cmd.Context(), ch, args[0], args[1], args[2], engine, target)
			}
			return connections.RunKillConnectionForEngine(cmd.Context(), ch, args[0], args[1], args[2], engine, target)
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Vitess keyspace to target")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Vitess shard to target")
	cmd.Flags().BoolVar(&flags.query, "query", false, "Cancel the query_id instead of terminating the connection_id")

	return cmd
}

func ConnectionsKillTransactionCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kill-transaction <database> <branch> <transaction-id>",
		Short: "Kill a Postgres branch transaction",
		Long: `Kill a Postgres branch transaction.

This is destructive. Pass a transaction_id from connections show to terminate
the matching Postgres connection.`,
		Args: cmdutil.RequiredArgs("database", "branch", "transaction-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := connections.ValidateTransactionID(args[2]); err != nil {
				return err
			}
			engine, err := databaseEngine(cmd.Context(), ch, args[0])
			if err != nil {
				return err
			}
			if engine != ps.DatabaseEnginePostgres {
				return errors.New("connections kill-transaction is only supported for Postgres databases")
			}
			return connections.RunKillTransactionForEngine(cmd.Context(), ch, args[0], args[1], args[2], engine, connections.ConnectionTarget{})
		},
	}

	return cmd
}

func databaseEngine(ctx context.Context, ch *cmdutil.Helper, database string) (ps.DatabaseEngine, error) {
	client, err := ch.Client()
	if err != nil {
		return "", err
	}

	db, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
		Organization: ch.Config.Organization,
		Database:     database,
	})
	if err != nil {
		return "", err
	}
	if db == nil {
		return "", errors.New("database not found")
	}
	return db.Kind, nil
}

func vitessConnectionID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("id must be a positive integer")
	}
	return id, nil
}

func isNegativeIDFlagError(err error) bool {
	const marker = " in -"

	_, suffix, ok := strings.Cut(err.Error(), marker)
	if !ok {
		return false
	}

	_, parseErr := strconv.ParseInt("-"+suffix, 10, 64)
	return parseErr == nil
}

func vitessConnectionIDFlagError(_ *cobra.Command, err error) error {
	if isNegativeIDFlagError(err) {
		return errors.New("id must be a positive integer")
	}
	return err
}
