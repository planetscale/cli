package insights

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

var anomalyPeriods = []string{"15m", "1h", "3h", "6h", "12h", "1d", "2d", "7d", "8d"}

// AnomalyRow is an anomaly formatted for table output.
type AnomalyRow struct {
	ID                 string `header:"id" json:"id"`
	Active             bool   `header:"active" json:"active"`
	PeriodStart        string `header:"started" json:"period_start"`
	PeriodEnd          string `header:"ended" json:"period_end"`
	MinutesInViolation int64  `header:"minutes in violation" json:"minutes_in_violation"`
}

// AnomalyDetailRow is an anomaly formatted for human-readable detail output.
type AnomalyDetailRow struct {
	ID                 string  `header:"id"`
	Active             bool    `header:"active"`
	PeriodStart        string  `header:"started"`
	PeriodEnd          string  `header:"ended"`
	MinutesInViolation int64   `header:"minutes in violation"`
	DurationSeconds    float64 `header:"duration (seconds)"`
}

// AnomalyCorrelationRow is a correlated query formatted for human-readable output.
type AnomalyCorrelationRow struct {
	Correlation float64 `header:"correlation"`
	Tablet      string  `header:"tablet"`
	Keyspace    string  `header:"keyspace"`
	Fingerprint string  `header:"fingerprint"`
	Query       string  `header:"query"`
}

// AnomaliesCmd lists detected resource anomalies for a branch.
func AnomaliesCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		from   string
		to     string
		period string
	}

	cmd := &cobra.Command{
		Use:   "anomalies <database> <branch> [anomaly-id]",
		Short: "List detected anomalies or retrieve one anomaly with its correlated queries",
		Example: `  # Anomalies detected during the last 24 hours
  pscale insights anomalies mydb main --org myorg --period 1d

  # Anomalies that started in a date range (date-only --to includes the whole day)
  pscale insights anomalies mydb main --org myorg \
    --from 07/23 --to 07/25

  # Anomalies that started in an explicit timestamp range
  pscale insights anomalies mydb main --org myorg \
    --from 2026-07-24T00:00:00Z --to 2026-07-25T00:00:00Z

  # Retrieve one anomaly and its correlated query patterns
  pscale insights anomalies mydb main 4888442 --org myorg`,
		Args: cobra.MatchAll(cmdutil.RequiredArgs("database", "branch"), cobra.MaximumNArgs(3)),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			if len(args) == 3 {
				if flags.from != "" || flags.to != "" || flags.period != "" {
					return fmt.Errorf("--from, --to, and --period cannot be used when retrieving an anomaly by ID")
				}
				return printAnomalyDetail(ctx, ch, database, branch, args[2])
			}

			from, to, err := anomalyTimeRange(flags.from, flags.to, time.Now())
			if err != nil {
				return err
			}
			if flags.period != "" && !slices.Contains(anomalyPeriods, flags.period) {
				return fmt.Errorf("invalid --period %q, must be one of: %s", flags.period, strings.Join(anomalyPeriods, ", "))
			}

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
			}, ps.WithPeriod(flags.period), ps.WithTimeRange(from, to))
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

	cmd.Flags().StringVar(&flags.from, "from", "", "Start of a time range (RFC3339, YYYY-MM-DD, MM/DD/YYYY, or MM/DD; date-only values use local time)")
	cmd.Flags().StringVar(&flags.to, "to", "", "End of a time range (same formats as --from; a date includes the whole day; defaults to now)")
	cmd.Flags().StringVar(&flags.period, "period", "", fmt.Sprintf("Named time period, one of: %s", strings.Join(anomalyPeriods, ", ")))
	cmd.MarkFlagsMutuallyExclusive("period", "from")
	cmd.MarkFlagsMutuallyExclusive("period", "to")

	return cmd
}

func printAnomalyDetail(ctx context.Context, ch *cmdutil.Helper, database, branch, anomalyID string) error {
	client, err := ch.Client()
	if err != nil {
		return err
	}

	end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching anomaly %s for %s in %s...",
		printer.BoldBlue(anomalyID), printer.BoldBlue(branch), printer.BoldBlue(database)))
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
	if err := ch.Printer.PrintResource(&AnomalyDetailRow{
		ID:                 anomaly.ID,
		Active:             anomaly.Active,
		PeriodStart:        anomaly.PeriodStart.Format("2006-01-02 15:04"),
		PeriodEnd:          periodEnd,
		MinutesInViolation: anomaly.MinutesInViolation,
		DurationSeconds:    anomaly.Duration,
	}); err != nil {
		return err
	}

	if len(anomaly.Correlations) == 0 {
		ch.Printer.Println("\nNo correlated query patterns were identified.")
		return nil
	}

	ch.Printer.Println("\nCorrelated query patterns:")
	rows := make([]*AnomalyCorrelationRow, 0, len(anomaly.Correlations))
	for _, correlation := range anomaly.Correlations {
		rows = append(rows, &AnomalyCorrelationRow{
			Correlation: round2(correlation.CorrelationCoefficient),
			Tablet:      correlation.TabletType,
			Keyspace:    correlation.Keyspace,
			Fingerprint: correlation.Fingerprint,
			Query:       truncate(correlation.NormalizedSQL, 80),
		})
	}
	return ch.Printer.PrintResource(rows)
}

func anomalyTimeRange(fromValue, toValue string, now time.Time) (time.Time, time.Time, error) {
	if fromValue == "" {
		if toValue != "" {
			return time.Time{}, time.Time{}, fmt.Errorf("--to requires --from")
		}
		return time.Time{}, time.Time{}, nil
	}

	from, _, err := parseAnomalyTime(fromValue, now)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --from %q: use RFC3339, YYYY-MM-DD, MM/DD/YYYY, or MM/DD", fromValue)
	}
	if toValue == "" {
		return from, time.Time{}, nil
	}

	to, dateOnly, err := parseAnomalyTime(toValue, now)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --to %q: use RFC3339, YYYY-MM-DD, MM/DD/YYYY, or MM/DD", toValue)
	}
	if dateOnly {
		to = to.AddDate(0, 0, 1)
	}
	if !from.Before(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("--from must be before --to")
	}
	return from, to, nil
}

func parseAnomalyTime(value string, now time.Time) (time.Time, bool, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, false, nil
	}

	formats := []struct {
		layout string
		value  string
	}{
		{layout: "2006-01-02", value: value},
		{layout: "1/2/2006", value: value},
		{layout: "1/2/2006", value: fmt.Sprintf("%s/%d", value, now.Year())},
	}
	for _, format := range formats {
		if parsed, err := time.ParseInLocation(format.layout, format.value, now.Location()); err == nil {
			return parsed, true, nil
		}
	}

	return time.Time{}, false, fmt.Errorf("unsupported time format")
}
