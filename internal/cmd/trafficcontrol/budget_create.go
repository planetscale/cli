package trafficcontrol

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func BudgetCreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name             string
		mode             string
		capacity         int
		rate             int
		burst            int
		concurrency      int
		warningThreshold int
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create a traffic budget",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating traffic budget %s for %s/%s",
				printer.BoldBlue(flags.name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			req := &ps.CreateTrafficBudgetRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Name:         flags.name,
				Mode:         flags.mode,
			}

			if cmd.Flags().Changed("capacity") {
				req.Capacity = &flags.capacity
			}
			if cmd.Flags().Changed("rate") {
				req.Rate = &flags.rate
			}
			if cmd.Flags().Changed("burst") {
				req.Burst = &flags.burst
			}
			if cmd.Flags().Changed("concurrency") {
				req.Concurrency = &flags.concurrency
			}
			if cmd.Flags().Changed("warning-threshold") {
				req.WarningThreshold = &flags.warningThreshold
			}

			budget, err := client.TrafficBudgets.Create(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s or branch %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Traffic budget %s was successfully created in %s/%s.\n",
					printer.BoldBlue(budget.Name), printer.BoldBlue(database), printer.BoldBlue(branch))
			}

			return ch.Printer.PrintResource(toTrafficBudget(budget))
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the traffic budget (required)")
	cmd.Flags().StringVar(&flags.mode, "mode", "warn", "Mode of the budget: enforce, warn, or off")
	cmd.Flags().IntVar(&flags.capacity, "capacity", 0, "Maximum capacity that can be banked (0-6000). Unlimited when not set.")
	cmd.Flags().IntVar(&flags.rate, "rate", 0, "Rate at which capacity refills, as a percentage of server resources (0-100). Unlimited when not set.")
	cmd.Flags().IntVar(&flags.burst, "burst", 0, "Maximum capacity a single query can consume (0-6000). Unlimited when not set.")
	cmd.Flags().IntVar(&flags.concurrency, "concurrency", 0, "Percentage of available worker processes (0-100). Unlimited when not set.")
	cmd.Flags().IntVar(&flags.warningThreshold, "warning-threshold", 0, "Percentage (0-100) of capacity, burst, or concurrency at which to emit warnings for enforced budgets.`")

	cmd.MarkFlagRequired("name") // nolint:errcheck

	return cmd
}
