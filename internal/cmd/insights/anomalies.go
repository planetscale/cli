package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// AnomalyRow is an anomaly formatted for table output.
type AnomalyRow struct {
	ID                 string `header:"id" json:"id"`
	Active             bool   `header:"active" json:"active"`
	PeriodStart        string `header:"started" json:"period_start"`
	PeriodEnd          string `header:"ended" json:"period_end"`
	MinutesInViolation int64  `header:"minutes in violation" json:"minutes_in_violation"`
}

// AnomaliesCmd lists detected resource anomalies for a branch.
func AnomaliesCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "anomalies <database> <branch>",
		Short: "List detected resource anomalies (CPU, memory, IOPS, rows read/written)",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching anomalies for %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			anomalies, err := client.QueryInsights.ListAnomalies(ctx, &ps.ListAnomaliesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return notFoundError(ch, err, database, branch)
			}
			end()

			if len(anomalies) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No anomalies detected for %s in %s.\n",
					printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(anomalies)
			}

			rows := make([]*AnomalyRow, 0, len(anomalies))
			for _, a := range anomalies {
				end := ""
				if !a.PeriodEnd.IsZero() {
					end = a.PeriodEnd.Format("2006-01-02 15:04")
				}
				rows = append(rows, &AnomalyRow{
					ID:                 a.ID,
					Active:             a.Active,
					PeriodStart:        a.PeriodStart.Format("2006-01-02 15:04"),
					PeriodEnd:          end,
					MinutesInViolation: a.MinutesInViolation,
				})
			}
			return ch.Printer.PrintResource(rows)
		},
	}

	return cmd
}
