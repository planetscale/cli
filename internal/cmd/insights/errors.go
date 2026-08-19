package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// ErrorRow is a query error row formatted for table output. The fingerprint is
// the full error_fingerprint rather than the truncated id, so it can be passed
// to 'pscale insights errors show'.
type ErrorRow struct {
	Count       int64  `header:"count" json:"error_count"`
	LastSeen    string `header:"last seen" json:"started_at"`
	Message     string `header:"message" json:"error_message"`
	Fingerprint string `header:"fingerprint" json:"error_fingerprint"`
}

// ErrorsCmd lists aggregated query errors for a branch.
func ErrorsCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		limit  int
		period string
	}

	cmd := &cobra.Command{
		Use:   "errors <database> <branch>",
		Short: "List queries that are failing with errors",
		Long: `List aggregated query errors for a branch.

Pass a value from the fingerprint column to 'pscale insights errors show' to
see the individual queries behind an error.`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching query errors for %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			errs, err := client.QueryInsights.ListErrors(ctx, &ps.ListQueryInsightsErrorsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			}, ps.WithPerPage(flags.limit), ps.WithPeriod(flags.period))
			if err != nil {
				return notFoundError(ch, err, database, branch)
			}
			end()

			if len(errs) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No query errors recorded for %s in %s.\n",
					printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(errs)
			}

			rows := make([]*ErrorRow, 0, len(errs))
			for _, e := range errs {
				rows = append(rows, &ErrorRow{
					Count:       e.ErrorCount,
					LastSeen:    e.StartedAt.Format("2006-01-02 15:04"),
					Message:     truncate(e.ErrorMessage, 100),
					Fingerprint: e.ErrorFingerprint,
				})
			}
			return ch.Printer.PrintResource(rows)
		},
	}

	cmd.Flags().IntVar(&flags.limit, "limit", 15, "Number of errors to return")
	cmd.Flags().StringVar(&flags.period, "period", "", "Time period to aggregate over (e.g. 1h, 1d)")

	cmd.AddCommand(ErrorsShowCmd(ch))

	return cmd
}
