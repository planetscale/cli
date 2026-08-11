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

// TagSummaryRow is a tag-grouped query stats row for table output.
type TagSummaryRow struct {
	Dimensions    string  `header:"dimensions" json:"dimensions"`
	Count         int64   `header:"count" json:"query_count"`
	TotalTimeMs   float64 `header:"total time (ms)" json:"sum_total_duration_millis"`
	TimePerQryMs  float64 `header:"per query (ms)" json:"time_per_query"`
	P99Ms         float64 `header:"p99 (ms)" json:"p99_latency"`
	RowsRead      int64   `header:"rows read" json:"sum_rows_read"`
	ReadPerReturn float64 `header:"read/returned" json:"rows_read_per_returned"`
	Errors        int64   `header:"errors" json:"error_count"`
}

// TagSummariesCmd lists query statistics grouped by tag keys.
func TagSummariesCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		tags   []string
		sort   string
		dir    string
		limit  int
		period string
	}

	cmd := &cobra.Command{
		Use:   "summaries <database> <branch>",
		Short: "List query statistics grouped by tag keys",
		Long: `Group query statistics by one or more tag keys.

--tags takes friendly names from 'pscale insights tags' / the Insights UI Key
picker (e.g. app, username, controller) — not the internal S/B ids.

If a name exists as both sql and system, disambiguate with sql:name or
system:name.`,
		Example: `  # List keys first, then summarize (matches the app Tags page)
  pscale insights tags mydb main --org myorg
  pscale insights tags summaries mydb main --org myorg --tags username

  # Group by multiple keys
  pscale insights tags summaries mydb main --org myorg --tags app --tags controller --sort totalTime`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			if len(flags.tags) == 0 {
				return fmt.Errorf("--tags is required (friendly names from 'pscale insights tags', e.g. --tags username)")
			}
			if !slices.Contains(sortMetrics, flags.sort) && flags.sort != "dimensions" {
				// summaries also allow "dimensions"; reuse query sort list + dimensions
				allowed := append([]string{"dimensions"}, sortMetrics...)
				return fmt.Errorf("invalid --sort %q, must be one of: %s", flags.sort, strings.Join(allowed, ", "))
			}
			if flags.dir != "asc" && flags.dir != "desc" {
				return fmt.Errorf("invalid --dir %q, must be asc or desc", flags.dir)
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching tag summaries for %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			tags, err := client.QueryInsights.ListTags(ctx, &ps.ListQueryTagsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			}, ps.WithPeriod(flags.period))
			if err != nil {
				return notFoundError(ch, err, database, branch)
			}

			ids, err := resolveTagIDs(tags, flags.tags)
			if err != nil {
				end()
				return err
			}

			summaries, err := client.QueryInsights.ListTagSummaries(ctx, &ps.ListTagSummariesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Tags:         ids,
			}, ps.WithPerPage(flags.limit), ps.WithSort(flags.sort, flags.dir), ps.WithPeriod(flags.period))
			if err != nil {
				return notFoundError(ch, err, database, branch)
			}
			end()

			if len(summaries) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No tag summaries for %s in %s.\n",
					printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(summaries)
			}

			rows := make([]*TagSummaryRow, 0, len(summaries))
			for _, s := range summaries {
				rows = append(rows, &TagSummaryRow{
					Dimensions:    formatDimensions(s.Dimensions),
					Count:         s.QueryCount,
					TotalTimeMs:   round2(s.SumTotalDurationMillis),
					TimePerQryMs:  round2(s.TimePerQuery),
					P99Ms:         round2(s.P99Latency),
					RowsRead:      s.SumRowsRead,
					ReadPerReturn: round2(s.RowsReadPerReturned),
					Errors:        s.ErrorCount,
				})
			}
			return ch.Printer.PrintResource(rows)
		},
	}

	cmd.Flags().StringArrayVar(&flags.tags, "tags", nil,
		"Tag key name from 'pscale insights tags' (repeatable; e.g. --tags username --tags app)")
	cmd.Flags().StringVar(&flags.sort, "sort", "totalTime",
		fmt.Sprintf("Metric to rank by, one of: dimensions, %s", strings.Join(sortMetrics, ", ")))
	cmd.Flags().StringVar(&flags.dir, "dir", "desc", "Sort direction: asc or desc")
	cmd.Flags().IntVar(&flags.limit, "limit", 25, "Number of summary rows to return")
	cmd.Flags().StringVar(&flags.period, "period", "", "Time period to aggregate over (e.g. 1h, 1d)")
	cmd.MarkFlagRequired("tags") // nolint:errcheck

	return cmd
}

func formatDimensions(dims map[string]string) string {
	if len(dims) == 0 {
		return ""
	}
	parts := make([]string, 0, len(dims))
	for k, v := range dims {
		label := k
		if len(k) > 1 {
			switch k[0] {
			case 'S':
				label = "sql:" + k[1:]
			case 'B':
				label = "system:" + k[1:]
			}
		}
		parts = append(parts, label+"="+v)
	}
	slices.Sort(parts)
	return strings.Join(parts, " ")
}
