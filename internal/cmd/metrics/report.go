package metrics

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

type metricsReport struct {
	Type         string                  `json:"type"`
	Organization string                  `json:"organization"`
	Database     string                  `json:"database"`
	Branch       string                  `json:"branch"`
	Engine       ps.DatabaseEngine       `json:"engine"`
	Period       string                  `json:"period,omitempty"`
	From         string                  `json:"from,omitempty"`
	To           string                  `json:"to,omitempty"`
	Steps        int                     `json:"steps,omitempty"`
	Sections     []*metricsReportSection `json:"sections"`
}

type metricsReportSection struct {
	Name   string            `json:"name"`
	Kind   reportSectionKind `json:"kind"`
	Result any               `json:"result"`
}

type metricsReportCSVRow struct {
	Section    string `csv:"section"`
	Kind       string `csv:"kind"`
	Timestamp  string `csv:"timestamp"`
	Metric     string `csv:"metric"`
	Series     string `csv:"series"`
	Dimensions string `csv:"dimensions"`
	Value      string `csv:"value"`
}

// ReportCmd produces an engine-aware, grouped metrics report for a branch.
func ReportCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		period string
		from   string
		to     string
		steps  int
	}

	cmd := &cobra.Command{
		Use:   "report <database> <branch>",
		Short: "Produce a grouped performance metrics report",
		Long: `Produce a curated performance report for a database branch.

The database engine is detected automatically. MySQL and PostgreSQL reports use
different metric sections, including current-value sections where applicable.
Section headings are bold in human output and plain text with --no-color.`,
		Example: `  # Daily human-readable performance report
  pscale metrics report mydb main --org myorg --period 1d

  # Weekly report without terminal styling
  pscale metrics report mydb main --org myorg --period 7d --no-color

  # Composite JSON report for automation
  pscale metrics report mydb main --org myorg --period 1d --format json`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateRangeFlags(cmd, flags.from, flags.to); err != nil {
				return err
			}
			if cmd.Flags().Changed("steps") && flags.steps <= 0 {
				return fmt.Errorf("--steps must be greater than zero")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			database, branch := args[0], args[1]
			progress := ch.Printer.StartProgress(fmt.Sprintf("Preparing metrics report for %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer progress.Stop()

			db, err := client.Databases.Get(cmd.Context(), &ps.GetDatabaseRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			definitions, err := reportSectionsForEngine(db.Kind)
			if err != nil {
				return err
			}

			period := flags.period
			if flags.from != "" {
				period = ""
			}
			report := &metricsReport{
				Type:         "MetricsReport",
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Engine:       db.Kind,
				Period:       period,
				From:         flags.from,
				To:           flags.to,
				Steps:        flags.steps,
				Sections:     make([]*metricsReportSection, 0, len(definitions)),
			}

			for _, definition := range definitions {
				progress.Update(fmt.Sprintf("Fetching %s...", definition.Name))
				section := &metricsReportSection{Name: definition.Name, Kind: definition.Kind}
				switch definition.Kind {
				case reportSeriesSection:
					result, err := client.Metrics.GetSeries(cmd.Context(), &ps.GetMetricSeriesRequest{
						Organization: ch.Config.Organization,
						Database:     database,
						Branch:       branch,
						Metrics:      definition.Metrics,
						Period:       period,
						From:         flags.from,
						To:           flags.to,
						Steps:        flags.steps,
					})
					if err != nil {
						return fmt.Errorf("fetching report section %q: %w", definition.Name, cmdutil.HandleError(err))
					}
					section.Result = result
				case reportInstantSection:
					result, err := client.Metrics.GetInstant(cmd.Context(), &ps.GetInstantMetricsRequest{
						Organization: ch.Config.Organization,
						Database:     database,
						Branch:       branch,
						Metrics:      definition.Metrics,
					})
					if err != nil {
						return fmt.Errorf("fetching report section %q: %w", definition.Name, cmdutil.HandleError(err))
					}
					section.Result = result
				default:
					return fmt.Errorf("unsupported metrics report section kind %q", definition.Kind)
				}
				report.Sections = append(report.Sections, section)
			}
			progress.Stop()

			switch ch.Printer.Format() {
			case printer.JSON:
				return ch.Printer.PrintJSON(report)
			case printer.CSV:
				return ch.Printer.PrintResource(metricsReportCSVRows(report))
			default:
				return printMetricsReport(ch, report)
			}
		},
	}

	cmd.Flags().StringVar(&flags.period, "period", "1d", "Named report period (15m, 1h, 3h, 6h, 12h, 1d, 2d, 7d, or 8d)")
	cmd.Flags().StringVar(&flags.from, "from", "", "Start of a custom time range as an ISO 8601 timestamp")
	cmd.Flags().StringVar(&flags.to, "to", "", "End of a custom time range as an ISO 8601 timestamp")
	cmd.Flags().IntVar(&flags.steps, "steps", 0, "Requested number of historical data points")

	return cmd
}

func printMetricsReport(ch *cmdutil.Helper, report *metricsReport) error {
	ch.Printer.Printf("%s\n", printer.Bold(fmt.Sprintf("Metrics report for %s/%s/%s", report.Organization, report.Database, report.Branch)))
	if start, end, interval, ok := reportRange(report); ok {
		ch.Printer.Printf("Range: %s · interval: %ds\n", formatMetricRange(start, end), interval)
	} else if report.Period != "" {
		ch.Printer.Printf("Period: %s\n", report.Period)
	} else {
		ch.Printer.Printf("Range: %s–%s\n", report.From, report.To)
	}

	for _, section := range report.Sections {
		ch.Printer.Printf("\n%s\n\n", printer.Bold(section.Name))
		switch result := section.Result.(type) {
		case *ps.MetricSeries:
			rows := seriesSummaryRows(result)
			if len(rows) == 0 {
				ch.Printer.Printf("No metrics returned.\n")
				continue
			}
			if err := ch.Printer.PrintResource(rows); err != nil {
				return err
			}
		case *ps.InstantMetrics:
			rows := instantMetricHumanRows(result)
			if len(rows) == 0 {
				ch.Printer.Printf("No metrics returned.\n")
				continue
			}
			if err := ch.Printer.PrintResource(rows); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported result type %T for report section %q", section.Result, section.Name)
		}
	}

	return nil
}

func metricsReportCSVRows(report *metricsReport) []*metricsReportCSVRow {
	rows := make([]*metricsReportCSVRow, 0)
	for _, section := range report.Sections {
		switch result := section.Result.(type) {
		case *ps.MetricSeries:
			for _, series := range result.Series {
				dimensions, _ := json.Marshal(series.Labels)
				for _, point := range series.Points {
					if len(point) < 2 {
						continue
					}
					rows = append(rows, &metricsReportCSVRow{
						Section:    section.Name,
						Kind:       string(section.Kind),
						Timestamp:  time.Unix(int64(point[0]), 0).UTC().Format(time.RFC3339),
						Metric:     series.Metric,
						Series:     series.Label,
						Dimensions: string(dimensions),
						Value:      formatRawValue(point[1]),
					})
				}
			}
		case *ps.InstantMetrics:
			for _, metric := range result.Metrics {
				for _, value := range metric.Values {
					rows = append(rows, &metricsReportCSVRow{
						Section:    section.Name,
						Kind:       string(section.Kind),
						Metric:     metric.Metric,
						Series:     metric.Label,
						Dimensions: formatInstantDimensions(value, ""),
						Value:      formatRawValue(value["value"]),
					})
				}
			}
		}
	}
	return rows
}

func reportRange(report *metricsReport) (time.Time, time.Time, int, bool) {
	for _, section := range report.Sections {
		series, ok := section.Result.(*ps.MetricSeries)
		if ok && !series.StartDate.IsZero() && !series.EndDate.IsZero() {
			return series.StartDate, series.EndDate, series.Interval, true
		}
	}
	return time.Time{}, time.Time{}, 0, false
}
