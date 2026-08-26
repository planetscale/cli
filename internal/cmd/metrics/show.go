package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// ShowCmd queries historical metric series for a branch.
func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		metrics     []string
		period      string
		from        string
		to          string
		steps       int
		tabletType  string
		keyspace    string
		shard       string
		role        string
		container   string
		pod         string
		pods        []string
		queryIDs    []string
		fingerprint string
		budgetID    string
		ruleID      string
		search      string
	}

	cmd := &cobra.Command{
		Use:   "show <database> <branch>",
		Short: "Show historical metric series",
		Example: `  # Summarize query volume and p99 latency over the last hour
  pscale metrics show mydb main --org myorg --metric queries --metric latency_p99 --period 1h

  # Export every sample as CSV
  pscale metrics show mydb main --org myorg --metric queries --period 1h --format csv

  # Preserve the complete metrics API response
  pscale metrics show mydb main --org myorg --metric queries --period 1h --format json`,
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
			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching metrics for %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			series, err := client.Metrics.GetSeries(cmd.Context(), &ps.GetMetricSeriesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Metrics:      flags.metrics,
				Period:       flags.period,
				From:         flags.from,
				To:           flags.to,
				Steps:        flags.steps,
				TabletType:   flags.tabletType,
				Keyspace:     flags.keyspace,
				Shard:        flags.shard,
				Role:         flags.role,
				Container:    flags.container,
				Pod:          flags.pod,
				Pods:         flags.pods,
				QueryIDs:     flags.queryIDs,
				Fingerprint:  flags.fingerprint,
				BudgetID:     flags.budgetID,
				RuleID:       flags.ruleID,
				Search:       flags.search,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			end()

			switch ch.Printer.Format() {
			case printer.JSON:
				return ch.Printer.PrintJSON(series)
			case printer.CSV:
				return ch.Printer.PrintResource(metricPointRows(series))
			default:
				return printSeriesSummary(ch, database, branch, series)
			}
		},
	}

	cmd.Flags().StringSliceVar(&flags.metrics, "metric", nil, "Metric to query (repeat or comma-separate)")
	cmd.Flags().StringVar(&flags.period, "period", "", "Named time period to query (for example 1h, 12h, or 1d; defaults to 12h)")
	cmd.Flags().StringVar(&flags.from, "from", "", "Start of a custom time range as an ISO 8601 timestamp")
	cmd.Flags().StringVar(&flags.to, "to", "", "End of a custom time range as an ISO 8601 timestamp")
	cmd.Flags().IntVar(&flags.steps, "steps", 0, "Requested number of data points")
	cmd.Flags().StringVar(&flags.tabletType, "tablet-type", "", "Filter by tablet type")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Filter by keyspace")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Filter by shard")
	cmd.Flags().StringVar(&flags.role, "role", "", "Filter by Postgres role")
	cmd.Flags().StringVar(&flags.container, "container", "", "Filter by container")
	cmd.Flags().StringVar(&flags.pod, "pod", "", "Filter by one pod")
	cmd.Flags().StringSliceVar(&flags.pods, "pods", nil, "Filter by pods (repeat or comma-separate)")
	cmd.Flags().StringSliceVar(&flags.queryIDs, "query-id", nil, "Filter by query pattern ID (repeat or comma-separate)")
	cmd.Flags().StringVar(&flags.fingerprint, "fingerprint", "", "Filter by query fingerprint")
	cmd.Flags().StringVar(&flags.budgetID, "budget-id", "", "Filter by traffic budget ID")
	cmd.Flags().StringVar(&flags.ruleID, "rule-id", "", "Filter by traffic rule ID")
	cmd.Flags().StringVarP(&flags.search, "search", "q", "", "Filter by search terms")
	cmd.MarkFlagRequired("metric") // nolint:errcheck

	return cmd
}

func validateRangeFlags(cmd *cobra.Command, from, to string) error {
	if (from == "") != (to == "") {
		return fmt.Errorf("--from and --to must be used together")
	}
	if from != "" && cmd.Flags().Changed("period") {
		return fmt.Errorf("--period cannot be combined with --from and --to")
	}
	return nil
}
