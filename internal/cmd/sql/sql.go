package sql

import (
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/sqlquery"
)

// SQLCmd runs queries without an interactive shell.
func SQLCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		query      string
		keyspace   string
		postgresDB string
		role       string
		replica    bool
		force      bool
		vertical   bool
	}

	cmd := &cobra.Command{
		Use:   "sql <database> <branch>",
		Short: "Execute a SQL query without an interactive shell",
		Long: `Execute a single SQL query against a database branch using ephemeral credentials.

Use --format json for machine-readable output. This command is intended for agents and scripts;
for interactive sessions use pscale shell instead.

Access flags match pscale shell: --role (reader, writer, readwriter, admin) and --replica.
Unlike shell, the default role is reader. Pass --role admin (or writer/readwriter) for writes.

Destructive SQL containing DELETE, DROP, or TRUNCATE is blocked unless --force is passed.
Agents must ask the user for approval before using --force.

MySQL (Vitess) databases use the primary keyspace by default (same as pscale shell -D @primary).
Pass --keyspace when targeting a specific keyspace in a multi-keyspace database. A keyspace may
include a shard and tablet type (mykeyspace/-80, mykeyspace/-80@replica) to pin the connection to
one shard; enumerate shards with SHOW VITESS_SHARDS.

PostgreSQL databases use --dbname (default postgres).

Human output prints rows as a table. Pass --vertical (or end the query with \G,
like the mysql client) to print one column per line, which is easier to read for
wide results such as SHOW REPLICA STATUS.

Place flags after positional arguments (see Usage). --org is required:

  pscale sql <database> <branch> --org <org> --format json --query "SELECT 1"`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		Example: `  # Read query (default reader role)
  pscale sql <database> <branch> --org <org> --format json --query "SELECT 1"

  # Read from replica
  pscale sql <database> <branch> --org <org> --format json --replica --query "SELECT 1"

  # MySQL — keyspace optional (@primary default)
  pscale sql <database> <branch> --org <org> --format json --keyspace <keyspace> --query "SELECT 1"

  # Vertical output for wide rows (--vertical, or end the query with \G)
  pscale sql <database> <branch> --org <org> --replica --query "SHOW REPLICA STATUS\G"`,
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
		RunE: func(cmd *cobra.Command, args []string) error {
			query, verticalTerminator := stripVerticalTerminator(flags.query)
			vertical := flags.vertical || verticalTerminator

			result, err := sqlquery.Execute(cmd.Context(), ch, sqlquery.Options{
				Organization: ch.Config.Organization,
				Database:     args[0],
				Branch:       args[1],
				Query:        query,
				Keyspace:     flags.keyspace,
				PostgresDB:   flags.postgresDB,
				Role:         flags.role,
				Replica:      flags.replica,
				Force:        flags.force,
			})
			if err != nil {
				return handleExecuteError(ch, err, args[0], args[1])
			}

			switch ch.Printer.Format() {
			case printer.JSON:
				return ch.Printer.PrintJSON(result)
			case printer.Human:
				printHumanResult(ch, result, vertical)
				return nil
			default:
				return ch.Printer.PrintResource(result.Rows)
			}
		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.Flags().StringVar(&flags.query, "query", "", "SQL query to execute")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Vitess keyspace, optionally with a shard and tablet type (e.g. mykeyspace, mykeyspace/-80, mykeyspace/-80@replica). List shards with --query \"SHOW VITESS_SHARDS\". Defaults to @primary, same as pscale shell.")
	cmd.Flags().StringVar(&flags.postgresDB, "dbname", "postgres", "PostgreSQL database name")
	cmd.Flags().StringVar(&flags.role, "role",
		"", "Role defines the access level, allowed values are: reader, writer, readwriter, admin. Defaults to reader (use --role admin for writes).")
	cmd.Flags().BoolVar(&flags.replica, "replica", false,
		"When enabled, the password will route all reads to the branch's primary replicas and all read-only regions.")
	cmd.Flags().BoolVar(&flags.force, "force", false,
		"Allow destructive SQL (DELETE, DROP, TRUNCATE). Only use after the user explicitly approves.")
	cmd.Flags().BoolVar(&flags.vertical, "vertical", false,
		"Print each row vertically, one column per line. Same as ending the query with \\G in the mysql client.")
	cmd.MarkFlagRequired("query")         // nolint:errcheck
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	return cmd
}
