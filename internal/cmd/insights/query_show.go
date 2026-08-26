package insights

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

type QueryShowRow struct {
	ID           string  `header:"id" json:"id"`
	Fingerprint  string  `header:"fingerprint" json:"fingerprint"`
	StartedAt    string  `header:"started" json:"started_at"`
	Statement    string  `header:"statement" json:"statement_type"`
	Keyspace     string  `header:"keyspace" json:"keyspace"`
	Tables       string  `header:"tables" json:"tables"`
	Username     string  `header:"user" json:"username"`
	Remote       string  `header:"remote address" json:"remote_address"`
	ShardQueries int64   `header:"shard queries" json:"shard_queries"`
	RowsRead     int64   `header:"rows read" json:"rows_read"`
	RowsAffected int64   `header:"rows affected" json:"rows_affected"`
	RowsReturned int64   `header:"rows returned" json:"rows_returned"`
	DurationMs   float64 `header:"duration (ms)" json:"total_duration_millis"`
	Error        string  `header:"error" json:"error_message"`
	Query        string  `header:"query" json:"normalized_sql"`
}

func QueryShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <branch> <query-id>",
		Short: "Show an individual query execution",
		Long: `Show one query execution using an ID returned by
'pscale insights queries samples'. The query ID is an execution/sample ID,
not the fingerprint used by the samples and summary commands.`,
		Example: `  pscale insights queries show mydb main exec-1 --org myorg`,
		Args:    cmdutil.RequiredArgs("database", "branch", "query-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, queryID := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching query execution %s on %s/%s...",
				printer.BoldBlue(queryID), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			query, err := client.QueryInsights.GetQuery(cmd.Context(), &ps.GetQueryRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				QueryID:      queryID,
			})
			if err != nil {
				if cmdutil.ErrCode(err) == ps.ErrNotFound {
					return fmt.Errorf("query execution %s does not exist on branch %s in database %s (organization: %s)",
						printer.BoldBlue(queryID), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				}
				return notFoundError(ch, err, database, branch)
			}
			end()

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(query)
			}

			startedAt := ""
			if query.StartedAt != nil {
				startedAt = query.StartedAt.Format("2006-01-02 15:04:05")
			}

			return ch.Printer.PrintResource([]*QueryShowRow{{
				ID:           query.ID,
				Fingerprint:  query.Fingerprint,
				StartedAt:    startedAt,
				Statement:    query.StatementType,
				Keyspace:     query.Keyspace,
				Tables:       strings.Join(query.Tables, ", "),
				Username:     query.Username,
				Remote:       query.RemoteAddress,
				ShardQueries: query.ShardQueries,
				RowsRead:     query.RowsRead,
				RowsAffected: query.RowsAffected,
				RowsReturned: query.RowsReturned,
				DurationMs:   round2(query.TotalDurationMillis),
				Error:        query.ErrorMessage,
				Query:        query.NormalizedSQL,
			}})
		},
	}

	return cmd
}
