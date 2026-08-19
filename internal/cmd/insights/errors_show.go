package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// ErrorQueryRow is a failed query execution formatted for table output.
type ErrorQueryRow struct {
	StartedAt  string  `header:"started" json:"started_at"`
	DurationMs float64 `header:"duration (ms)" json:"total_duration_millis"`
	Username   string  `header:"user" json:"username"`
	Keyspace   string  `header:"keyspace" json:"keyspace"`
	Error      string  `header:"error" json:"error_message"`
	Query      string  `header:"query" json:"normalized_sql"`
}

// ErrorsShowCmd lists the individual queries behind an error fingerprint.
func ErrorsShowCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		limit  int
		period string
	}

	cmd := &cobra.Command{
		Use:   "show <database> <branch> <fingerprint>",
		Short: "Show the queries behind an error fingerprint",
		Long: `Show the individual query executions that failed with an error fingerprint.

Use the fingerprint column from 'pscale insights errors <database> <branch>'
(error_fingerprint in JSON), not the truncated id.

Useful for seeing which users, keyspaces, and statements produced the error.`,
		Example: `  pscale insights errors show mydb main b129e8fa --org myorg
  pscale insights errors show mydb main b129e8fa --org myorg --period 1h --limit 50`,
		Args: cmdutil.RequiredArgs("database", "branch", "fingerprint"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, fingerprint := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching queries for error fingerprint %s on %s/%s...",
				printer.BoldBlue(fingerprint), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			queries, err := client.QueryInsights.ListErrorQueries(ctx, &ps.ListErrorQueriesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Fingerprint:  fingerprint,
			}, ps.WithPerPage(flags.limit), ps.WithPeriod(flags.period))
			if err != nil {
				return notFoundError(ch, err, database, branch)
			}
			end()

			if len(queries) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No queries recorded for error fingerprint %s on %s/%s.\n",
					printer.BoldBlue(fingerprint), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(queries)
			}

			rows := make([]*ErrorQueryRow, 0, len(queries))
			for _, q := range queries {
				rows = append(rows, &ErrorQueryRow{
					StartedAt:  q.StartedAt.Format("2006-01-02 15:04:05"),
					DurationMs: round2(q.TotalDurationMillis),
					Username:   q.Username,
					Keyspace:   q.Keyspace,
					Error:      truncate(q.ErrorMessage, 60),
					Query:      truncate(q.NormalizedSQL, 60),
				})
			}
			return ch.Printer.PrintResource(rows)
		},
	}

	cmd.Flags().IntVar(&flags.limit, "limit", 25, "Number of queries to return")
	cmd.Flags().StringVar(&flags.period, "period", "", "Time period to look back (e.g. 1h, 1d)")

	return cmd
}
