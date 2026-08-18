package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// InstantCmd queries the current values of branch metrics.
func InstantCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		metrics   []string
		role      string
		shard     string
		container string
		pod       string
	}

	cmd := &cobra.Command{
		Use:   "instant <database> <branch>",
		Short: "Show current metric values",
		Example: `  # Show current disk utilization for every Postgres pod
  pscale metrics instant mydb main --org myorg --metric planetscale_volume_usage_percentage

  # Preserve the complete instant metrics API response
  pscale metrics instant mydb main --org myorg --metric planetscale_volume_usage_percentage --format json`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}

			database, branch := args[0], args[1]
			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching current metrics for %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			metrics, err := client.Metrics.GetInstant(cmd.Context(), &ps.GetInstantMetricsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Metrics:      flags.metrics,
				Role:         flags.role,
				Shard:        flags.shard,
				Container:    flags.container,
				Pod:          flags.pod,
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

			rows := instantMetricHumanRows(metrics)
			if len(rows) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No current metric values returned for %s in %s.\n",
					printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}
			return ch.Printer.PrintResource(rows)
		},
	}

	cmd.Flags().StringSliceVar(&flags.metrics, "metric", nil, "Metric to query (repeat or comma-separate)")
	cmd.Flags().StringVar(&flags.role, "role", "", "Filter by Postgres role")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Filter by shard")
	cmd.Flags().StringVar(&flags.container, "container", "", "Filter by container")
	cmd.Flags().StringVar(&flags.pod, "pod", "", "Filter by pod")
	cmd.MarkFlagRequired("metric") // nolint:errcheck

	return cmd
}
