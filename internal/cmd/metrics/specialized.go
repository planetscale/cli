package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

type specializedSeriesFlags struct {
	metrics []string
	period  string
	from    string
	to      string
	steps   int
}

type queryDimensionFlags struct {
	tabletType string
	budgetID   string
	ruleID     string
	search     string
}

func QueriesCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		specializedSeriesFlags
		queryDimensionFlags
		queryIDs    []string
		fingerprint string
		keyspace    string
	}

	cmd := &cobra.Command{
		Use:   "queries <database> <branch>",
		Short: "Show metrics for SQL queries",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateSpecializedSeriesFlags(cmd, flags.specializedSeriesFlags); err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			database, branch := args[0], args[1]
			end := specializedMetricsProgress(ch, database, branch, "query")
			defer end()

			series, err := client.Metrics.GetQuerySeries(cmd.Context(), &ps.GetQueryMetricSeriesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Metrics:      flags.metrics,
				QueryIDs:     flags.queryIDs,
				Fingerprint:  flags.fingerprint,
				Keyspace:     flags.keyspace,
				Period:       flags.period,
				From:         flags.from,
				To:           flags.to,
				Steps:        flags.steps,
				TabletType:   flags.tabletType,
				BudgetID:     flags.budgetID,
				RuleID:       flags.ruleID,
				Search:       flags.search,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			end()

			return printSpecializedSeries(ch, database, branch, series)
		},
	}

	addSpecializedSeriesFlags(cmd, &flags.specializedSeriesFlags)
	addQueryDimensionFlags(cmd, &flags.queryDimensionFlags)
	cmd.Flags().StringSliceVar(&flags.queryIDs, "query-id", nil, "Filter by query pattern ID (repeat or comma-separate)")
	cmd.Flags().StringVar(&flags.fingerprint, "fingerprint", "", "Filter by query fingerprint")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace for the query fingerprint")

	return cmd
}

func TablesCmd(ch *cmdutil.Helper) *cobra.Command {
	return storageMetricsCmd(ch, "tables", "Show table storage metrics", func(client *ps.Client, cmd *cobra.Command, req *ps.GetBranchMetricsRequest) ([]byte, error) {
		return client.Metrics.GetTables(cmd.Context(), req)
	})
}

func KeyspaceTablesCmd(ch *cmdutil.Helper) *cobra.Command {
	return storageMetricsCmd(ch, "keyspace-tables", "Show table storage metrics by keyspace", func(client *ps.Client, cmd *cobra.Command, req *ps.GetBranchMetricsRequest) ([]byte, error) {
		return client.Metrics.GetKeyspaceTables(cmd.Context(), req)
	})
}

func TabletsCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		specializedSeriesFlags
		keyspace string
		shard    string
		pod      string
		workflow string
	}

	cmd := &cobra.Command{
		Use:   "tablets <database> <branch>",
		Short: "Show tablet metric series",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateSpecializedSeriesFlags(cmd, flags.specializedSeriesFlags); err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			database, branch := args[0], args[1]
			end := specializedMetricsProgress(ch, database, branch, "tablet")
			defer end()

			series, err := client.Metrics.GetTabletSeries(cmd.Context(), &ps.GetTabletMetricSeriesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Metrics:      flags.metrics,
				Period:       flags.period,
				From:         flags.from,
				To:           flags.to,
				Steps:        flags.steps,
				Keyspace:     flags.keyspace,
				Shard:        flags.shard,
				Pod:          flags.pod,
				Workflow:     flags.workflow,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			end()

			return printSpecializedSeries(ch, database, branch, series)
		},
	}

	addSpecializedSeriesFlags(cmd, &flags.specializedSeriesFlags)
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Filter by keyspace")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Filter by shard")
	cmd.Flags().StringVar(&flags.pod, "pod", "", "Filter by pod")
	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Filter by VReplication workflow")
	cmd.AddCommand(InstantTabletsCmd(ch))

	return cmd
}

func InstantTabletsCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		metrics  []string
		keyspace string
		shard    string
	}

	cmd := &cobra.Command{
		Use:   "instant <database> <branch>",
		Short: "Show current tablet metric values",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}

			database, branch := args[0], args[1]
			end := specializedMetricsProgress(ch, database, branch, "current tablet")
			defer end()

			metrics, err := client.Metrics.GetInstantTablets(cmd.Context(), &ps.GetInstantTabletMetricsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Metrics:      flags.metrics,
				Keyspace:     flags.keyspace,
				Shard:        flags.shard,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			end()

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(metrics)
			}
			if ch.Printer.Format() == printer.CSV {
				return ch.Printer.PrintResource(instantMetricCSVRows(metrics))
			}
			return ch.Printer.PrintResource(instantMetricHumanRows(metrics))
		},
	}

	cmd.Flags().StringSliceVar(&flags.metrics, "metric", nil, "Metric to query (repeat or comma-separate)")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Filter by keyspace")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Filter by shard")

	return cmd
}

func TagsCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		specializedSeriesFlags
		queryDimensionFlags
		tagSets []string
	}

	cmd := &cobra.Command{
		Use:   "tags <database> <branch>",
		Short: "Show metrics grouped by query tags",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateSpecializedSeriesFlags(cmd, flags.specializedSeriesFlags); err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			database, branch := args[0], args[1]
			end := specializedMetricsProgress(ch, database, branch, "query tag")
			defer end()

			series, err := client.Metrics.GetTagSeries(cmd.Context(), &ps.GetTagMetricSeriesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Metrics:      flags.metrics,
				TagSets:      flags.tagSets,
				Period:       flags.period,
				From:         flags.from,
				To:           flags.to,
				Steps:        flags.steps,
				TabletType:   flags.tabletType,
				BudgetID:     flags.budgetID,
				RuleID:       flags.ruleID,
				Search:       flags.search,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			end()

			return printSpecializedSeries(ch, database, branch, series)
		},
	}

	addSpecializedSeriesFlags(cmd, &flags.specializedSeriesFlags)
	addQueryDimensionFlags(cmd, &flags.queryDimensionFlags)
	cmd.Flags().StringSliceVar(&flags.tagSets, "tag-set", nil, "Filter by tag set (repeat or comma-separate)")

	return cmd
}

func addSpecializedSeriesFlags(cmd *cobra.Command, flags *specializedSeriesFlags) {
	cmd.Flags().StringSliceVar(&flags.metrics, "metric", nil, "Metric to query (repeat or comma-separate)")
	cmd.Flags().StringVar(&flags.period, "period", "", "Named time period to query (for example 1h, 12h, or 1d; defaults to 12h)")
	cmd.Flags().StringVar(&flags.from, "from", "", "Start of a custom time range as an ISO 8601 timestamp")
	cmd.Flags().StringVar(&flags.to, "to", "", "End of a custom time range as an ISO 8601 timestamp")
	cmd.Flags().IntVar(&flags.steps, "steps", 0, "Requested number of data points")
}

func addQueryDimensionFlags(cmd *cobra.Command, flags *queryDimensionFlags) {
	cmd.Flags().StringVar(&flags.tabletType, "tablet-type", "", "Filter by tablet type")
	cmd.Flags().StringVar(&flags.budgetID, "budget-id", "", "Filter by traffic budget ID")
	cmd.Flags().StringVar(&flags.ruleID, "rule-id", "", "Filter by traffic rule ID")
	cmd.Flags().StringVarP(&flags.search, "search", "q", "", "Filter by search terms")
}

func validateSpecializedSeriesFlags(cmd *cobra.Command, flags specializedSeriesFlags) error {
	if err := validateRangeFlags(cmd, flags.from, flags.to); err != nil {
		return err
	}
	if cmd.Flags().Changed("steps") && flags.steps <= 0 {
		return fmt.Errorf("--steps must be greater than zero")
	}
	return nil
}

func specializedMetricsProgress(ch *cmdutil.Helper, database, branch, kind string) func() {
	return ch.Printer.PrintProgress(fmt.Sprintf("Fetching %s metrics for %s in %s...",
		kind, printer.BoldBlue(branch), printer.BoldBlue(database)))
}

func printSpecializedSeries(ch *cmdutil.Helper, database, branch string, series *ps.MetricSeries) error {
	switch ch.Printer.Format() {
	case printer.JSON:
		return ch.Printer.PrintJSON(series)
	case printer.CSV:
		return ch.Printer.PrintResource(metricPointRows(series))
	default:
		return printSeriesSummary(ch, database, branch, series)
	}
}

func storageMetricsCmd(ch *cmdutil.Helper, use, short string, fetch func(*ps.Client, *cobra.Command, *ps.GetBranchMetricsRequest) ([]byte, error)) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <database> <branch>",
		Short: short,
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}

			database, branch := args[0], args[1]
			end := specializedMetricsProgress(ch, database, branch, use)
			defer end()

			response, err := fetch(client, cmd, &ps.GetBranchMetricsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			end()

			return ch.Printer.PrettyPrintJSON(response)
		},
	}
}
