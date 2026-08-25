package insights

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// sortMetrics are the sort keys the insights API accepts that are useful for
// ranking queries from the CLI.
var sortMetrics = []string{
	"totalTime",
	"count",
	"errorCount",
	"rowsRead",
	"rowsReturned",
	"rowsAffected",
	"rowsReadPerReturned",
	"p50Latency",
	"p99Latency",
	"maxLatency",
	"cpuTime",
	"ioTime",
	"lastRun",
}

// QueryRow is a query statistics row formatted for table output.
type QueryRow struct {
	Fingerprint   string  `header:"fingerprint" json:"fingerprint"`
	Keyspace      string  `header:"keyspace" json:"keyspace"`
	Count         int64   `header:"count" json:"query_count"`
	TotalTimeMs   float64 `header:"total time (ms)" json:"sum_total_duration_millis"`
	TimePerQryMs  float64 `header:"per query (ms)" json:"time_per_query"`
	P50Ms         float64 `header:"p50 (ms)" json:"p50_latency"`
	P99Ms         float64 `header:"p99 (ms)" json:"p99_latency"`
	RowsRead      int64   `header:"rows read" json:"sum_rows_read"`
	ReadPerReturn float64 `header:"read/returned" json:"rows_read_per_returned"`
	Errors        int64   `header:"errors" json:"error_count"`
	LastRunAt     string  `header:"last run" json:"last_run_at"`
	Query         string  `header:"query" json:"normalized_sql"`
}

// QueriesCmd lists aggregated query statistics for a branch.
func QueriesCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		sort   string
		dir    string
		limit  int
		period string
	}

	cmd := &cobra.Command{
		Use:   "queries <database> <branch>",
		Short: "List the top queries by a performance metric",
		Example: `  # Queries taking the most cumulative time
  pscale insights queries mydb main --org myorg

  # Most expensive read patterns (rows read vs returned)
  pscale insights queries mydb main --org myorg --sort rowsReadPerReturned

  # Highest p99 latency over the last hour
  pscale insights queries mydb main --org myorg --sort p99Latency --period 1h

  # Recent executions for a fingerprint from the list (keyspace is required)
  pscale insights queries samples mydb main b129e8fa --org myorg --keyspace mydb`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			if !slices.Contains(sortMetrics, flags.sort) {
				return fmt.Errorf("invalid --sort %q, must be one of: %s", flags.sort, strings.Join(sortMetrics, ", "))
			}
			if flags.dir != "asc" && flags.dir != "desc" {
				return fmt.Errorf("invalid --dir %q, must be asc or desc", flags.dir)
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching query insights for %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			insights, err := client.QueryInsights.ListQueries(ctx, &ps.ListQueryInsightsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			}, ps.WithPerPage(flags.limit), ps.WithSort(flags.sort, flags.dir), ps.WithPeriod(flags.period))
			if err != nil {
				return notFoundError(ch, err, database, branch)
			}
			end()

			if len(insights) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No query statistics recorded for %s in %s.\n",
					printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(insights)
			}
			return ch.Printer.PrintResource(toQueryRows(insights))
		},
	}

	cmd.Flags().StringVar(&flags.sort, "sort", "totalTime",
		fmt.Sprintf("Metric to rank queries by, one of: %s", strings.Join(sortMetrics, ", ")))
	cmd.Flags().StringVar(&flags.dir, "dir", "desc", "Sort direction: asc or desc")
	cmd.Flags().IntVar(&flags.limit, "limit", 15, "Number of queries to return")
	cmd.Flags().StringVar(&flags.period, "period", "", "Time period to aggregate over (e.g. 1h, 1d)")

	cmd.AddCommand(QuerySamplesCmd(ch))
	cmd.AddCommand(QueryTrafficBudgetsCmd(ch))

	return cmd
}

func toQueryRows(insights []*ps.QueryInsight) []*QueryRow {
	rows := make([]*QueryRow, 0, len(insights))
	for _, in := range insights {
		rows = append(rows, &QueryRow{
			Fingerprint:   in.Fingerprint,
			Keyspace:      in.Keyspace,
			Count:         in.QueryCount,
			TotalTimeMs:   round2(in.SumTotalDurationMillis),
			TimePerQryMs:  round2(in.TimePerQuery),
			P50Ms:         round2(in.P50Latency),
			P99Ms:         round2(in.P99Latency),
			RowsRead:      in.SumRowsRead,
			ReadPerReturn: round2(in.RowsReadPerReturned),
			Errors:        in.ErrorCount,
			LastRunAt:     in.LastRunAt.Format("2006-01-02 15:04"),
			Query:         truncate(in.NormalizedSQL, 80),
		})
	}
	return rows
}

func round2(f float64) float64 {
	return float64(int64(f*100+0.5)) / 100
}
