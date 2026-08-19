package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// CorrelationRow is a query correlated with an anomaly, for table output.
type CorrelationRow struct {
	Correlation float64 `header:"correlation" json:"r"`
	Fingerprint string  `header:"fingerprint" json:"fingerprint"`
	Keyspace    string  `header:"keyspace" json:"keyspace"`
	TabletType  string  `header:"tablet type" json:"tablet_type"`
	Query       string  `header:"query" json:"normalized_sql"`
}

// AnomaliesShowCmd shows an anomaly and the queries correlated with it.
func AnomaliesShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <branch> <anomaly-id>",
		Short: "Show an anomaly and its correlated queries",
		Long: `Show a single anomaly from 'pscale insights anomalies <database> <branch>',
along with the queries whose activity correlates with it.`,
		Example: `  pscale insights anomalies show mydb main anomaly-id --org myorg`,
		Args:    cmdutil.RequiredArgs("database", "branch", "anomaly-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, anomalyID := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching anomaly %s on %s/%s...",
				printer.BoldBlue(anomalyID), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			anomaly, err := client.QueryInsights.GetAnomaly(ctx, &ps.GetAnomalyRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				AnomalyID:    anomalyID,
			})
			if err != nil {
				return notFoundError(ch, err, database, branch)
			}
			end()

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(anomaly)
			}

			periodEnd := ""
			if !anomaly.PeriodEnd.IsZero() {
				periodEnd = anomaly.PeriodEnd.Format("2006-01-02 15:04")
			}
			if err := ch.Printer.PrintResource([]*AnomalyRow{{
				ID:                 anomaly.ID,
				Active:             anomaly.Active,
				PeriodStart:        anomaly.PeriodStart.Format("2006-01-02 15:04"),
				PeriodEnd:          periodEnd,
				MinutesInViolation: anomaly.MinutesInViolation,
			}}); err != nil {
				return err
			}

			if ch.Printer.Format() != printer.Human {
				return nil
			}

			if len(anomaly.Correlations) == 0 {
				ch.Printer.Println("\nNo correlated queries for this anomaly.")
				return nil
			}

			ch.Printer.Printf("\n%s\n", printer.Bold("Correlated queries:"))
			rows := make([]*CorrelationRow, 0, len(anomaly.Correlations))
			for _, c := range anomaly.Correlations {
				rows = append(rows, &CorrelationRow{
					Correlation: round2(c.R),
					Fingerprint: c.Fingerprint,
					Keyspace:    c.Keyspace,
					TabletType:  c.TabletType,
					Query:       truncate(c.NormalizedSQL, 60),
				})
			}
			return ch.Printer.PrintResource(rows)
		},
	}

	return cmd
}
