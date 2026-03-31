package trafficcontrol

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func BudgetUpdateCmd(ch *cmdutil.Helper) *cobra.Command {
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
		Use:   "update <database> <branch> <budget-id>",
		Short: "Update a traffic budget",
		Args:  cmdutil.RequiredArgs("database", "branch", "budget-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			budgetID := args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			req := &ps.UpdateTrafficBudgetRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				BudgetID:     budgetID,
			}

			if cmd.Flags().Changed("name") {
				req.Name = &flags.name
			}
			if cmd.Flags().Changed("mode") {
				req.Mode = &flags.mode
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating traffic budget %s in %s/%s",
				printer.BoldBlue(budgetID), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			budget, err := client.TrafficBudgets.Update(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("traffic budget %s does not exist in %s/%s (organization: %s)",
						printer.BoldBlue(budgetID), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Traffic budget %s was successfully updated.\n", printer.BoldBlue(budgetID))
			}

			return ch.Printer.PrintResource(toTrafficBudget(budget))
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the traffic budget")
	cmd.Flags().StringVar(&flags.mode, "mode", "", "Mode of the budget: enforce, warn, or off")
	cmd.Flags().IntVar(&flags.capacity, "capacity", 0, "Maximum capacity that can be banked (0-6000). Unlimited when not set.")
	cmd.Flags().IntVar(&flags.rate, "rate", 0, "Rate at which capacity refills, as a percentage of server resources (0-100). Unlimited when not set.")
	cmd.Flags().IntVar(&flags.burst, "burst", 0, "Maximum capacity a single query can consume (0-6000). Unlimited when not set.")
	cmd.Flags().IntVar(&flags.concurrency, "concurrency", 0, "Percentage of available worker processes (0-100). Unlimited when not set.")
	cmd.Flags().IntVar(&flags.warningThreshold, "warning-threshold", 0, "Percentage (0-100) of capacity, burst, or concurrency at which to emit warnings for enforced budgets.")

	return cmd
}
