package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// QuerySampleRow is an individual query execution for table output.
type QuerySampleRow struct {
	ID           string  `header:"id" json:"id"`
	StartedAt    string  `header:"started" json:"started_at"`
	DurationMs   float64 `header:"duration (ms)" json:"total_duration_millis"`
	Username     string  `header:"user" json:"username"`
	RowsRead     int64   `header:"rows read" json:"rows_read"`
	RowsReturned int64   `header:"rows returned" json:"rows_returned"`
	Error        string  `header:"error" json:"error_message"`
	Query        string  `header:"query" json:"normalized_sql"`
}

// QuerySamplesCmd lists recent individual executions for a query fingerprint.
func QuerySamplesCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		limit    int
		period   string
		keyspace string
	}

	cmd := &cobra.Command{
		Use:   "samples <database> <branch> <fingerprint>",
		Short: "List recent executions for a query fingerprint",
		Long: `List individual query executions for a fingerprint from
'pscale insights queries <database> <branch>'.

--keyspace is required (use the keyspace column from the queries list).
Useful for seeing who ran the query, when, duration, and any error message.`,
		Example: `  pscale insights queries samples mydb main b129e8fa --org myorg --keyspace mydb
  pscale insights queries samples mydb main b129e8fa --org myorg --keyspace mydb --period 1h --limit 25`,
		Args: cmdutil.RequiredArgs("database", "branch", "fingerprint"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, fingerprint := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching query samples for fingerprint %s on %s/%s...",
				printer.BoldBlue(fingerprint), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			samples, err := client.QueryInsights.ListQuerySamples(ctx, &ps.ListQuerySamplesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Fingerprint:  fingerprint,
			}, ps.WithPerPage(flags.limit), ps.WithPeriod(flags.period), ps.WithKeyspace(flags.keyspace))
			if err != nil {
				return notFoundError(ch, err, database, branch)
			}
			end()

			if len(samples) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No query samples for fingerprint %s on %s/%s.\n",
					printer.BoldBlue(fingerprint), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(samples)
			}

			rows := make([]*QuerySampleRow, 0, len(samples))
			for _, s := range samples {
				rows = append(rows, &QuerySampleRow{
					ID:           s.ID,
					StartedAt:    s.StartedAt.Format("2006-01-02 15:04:05"),
					DurationMs:   round2(s.TotalDurationMillis),
					Username:     s.Username,
					RowsRead:     s.RowsRead,
					RowsReturned: s.RowsReturned,
					Error:        truncate(s.ErrorMessage, 60),
					Query:        truncate(s.NormalizedSQL, 60),
				})
			}
			return ch.Printer.PrintResource(rows)
		},
	}

	cmd.Flags().IntVar(&flags.limit, "limit", 25, "Number of samples to return")
	cmd.Flags().StringVar(&flags.period, "period", "", "Time period to look back (e.g. 1h, 1d)")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace for the fingerprint (required; from insights queries)")
	cmd.MarkFlagRequired("keyspace") // nolint:errcheck

	return cmd
}
