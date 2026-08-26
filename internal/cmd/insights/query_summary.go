package insights

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

type QuerySummaryRow struct {
	Fingerprint   string  `header:"fingerprint" json:"fingerprint"`
	Keyspace      string  `header:"keyspace" json:"keyspace"`
	Statement     string  `header:"statement" json:"statement_type"`
	Count         int64   `header:"count" json:"query_count"`
	Errors        int64   `header:"errors" json:"error_count"`
	TotalTimeMs   float64 `header:"total time (ms)" json:"sum_total_duration_millis"`
	TimePerQryMs  float64 `header:"per query (ms)" json:"time_per_query"`
	P50Ms         float64 `header:"p50 (ms)" json:"p50_latency"`
	P99Ms         float64 `header:"p99 (ms)" json:"p99_latency"`
	MaxMs         float64 `header:"max (ms)" json:"max_latency"`
	RowsRead      int64   `header:"rows read" json:"sum_rows_read"`
	RowsReturned  int64   `header:"rows returned" json:"sum_rows_returned"`
	ReadPerReturn float64 `header:"read/returned" json:"rows_read_per_returned"`
	LastRunAt     string  `header:"last run" json:"last_run_at"`
	Tables        string  `header:"tables" json:"tables"`
	Multishard    bool    `header:"multishard" json:"multishard"`
	Query         string  `header:"query" json:"normalized_sql"`
}

func QuerySummaryCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
		period   string
		from     string
		to       string
	}

	cmd := &cobra.Command{
		Use:   "summary <database> <branch> <fingerprint>",
		Short: "Show aggregate statistics for a query fingerprint",
		Long: `Show aggregate statistics for a fingerprint from
'pscale insights queries <database> <branch>'.

--keyspace is required (use the keyspace column from the queries list).
The fingerprint identifies a query pattern; it is not an execution/sample ID.`,
		Example: `  pscale insights queries summary mydb main b129e8fa --org myorg --keyspace mydb
  pscale insights queries summary mydb main b129e8fa --org myorg --keyspace mydb --period 1h
  pscale insights queries summary mydb main b129e8fa --org myorg --keyspace mydb --from 2026-08-25T14:00:00Z --to 2026-08-25T15:00:00Z`,
		Args: cmdutil.RequiredArgs("database", "branch", "fingerprint"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateQuerySummaryRangeFlags(cmd, flags.from, flags.to); err != nil {
				return err
			}

			database, branch, fingerprint := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching query summary for fingerprint %s on %s/%s...",
				printer.BoldBlue(fingerprint), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			summary, err := client.QueryInsights.GetQuerySummary(cmd.Context(), &ps.GetQuerySummaryRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Fingerprint:  fingerprint,
			}, ps.WithKeyspace(flags.keyspace), ps.WithPeriod(flags.period), ps.WithTimeRange(flags.from, flags.to))
			if err != nil {
				if cmdutil.ErrCode(err) == ps.ErrNotFound {
					return fmt.Errorf("query fingerprint %s does not exist in keyspace %s on branch %s in database %s (organization: %s)",
						printer.BoldBlue(fingerprint), printer.BoldBlue(flags.keyspace), printer.BoldBlue(branch),
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				}
				return notFoundError(ch, err, database, branch)
			}
			end()

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(summary)
			}

			lastRunAt := ""
			if summary.LastRunAt != nil {
				lastRunAt = summary.LastRunAt.Format("2006-01-02 15:04:05")
			}

			return ch.Printer.PrintResource([]*QuerySummaryRow{{
				Fingerprint:   summary.Fingerprint,
				Keyspace:      summary.Keyspace,
				Statement:     summary.StatementType,
				Count:         summary.QueryCount,
				Errors:        summary.ErrorCount,
				TotalTimeMs:   round2(summary.SumTotalDurationMillis),
				TimePerQryMs:  round2(summary.TimePerQuery),
				P50Ms:         round2(summary.P50Latency),
				P99Ms:         round2(summary.P99Latency),
				MaxMs:         round2(summary.MaxLatency),
				RowsRead:      summary.SumRowsRead,
				RowsReturned:  summary.SumRowsReturned,
				ReadPerReturn: round2(summary.RowsReadPerReturned),
				LastRunAt:     lastRunAt,
				Tables:        strings.Join(summary.Tables, ", "),
				Multishard:    summary.Multishard,
				Query:         summary.NormalizedSQL,
			}})
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace for the fingerprint (required; from insights queries)")
	cmd.Flags().StringVar(&flags.period, "period", "", "Named time period to summarize (for example 1h, 12h, or 1d)")
	cmd.Flags().StringVar(&flags.from, "from", "", "Start of a custom time range as an ISO 8601 timestamp")
	cmd.Flags().StringVar(&flags.to, "to", "", "End of a custom time range as an ISO 8601 timestamp")
	cmd.MarkFlagRequired("keyspace") // nolint:errcheck

	return cmd
}

func validateQuerySummaryRangeFlags(cmd *cobra.Command, from, to string) error {
	if (from == "") != (to == "") {
		return fmt.Errorf("--from and --to must be used together")
	}
	if from != "" && cmd.Flags().Changed("period") {
		return fmt.Errorf("--period cannot be combined with --from and --to")
	}
	return nil
}
