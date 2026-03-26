package trafficcontrol

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func BudgetShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <branch> <budget-id>",
		Short: "Show a traffic budget",
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching traffic budget %s", printer.BoldBlue(budgetID)))
			defer end()

			budget, err := client.TrafficBudgets.Get(ctx, &ps.GetTrafficBudgetRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				BudgetID:     budgetID,
			})
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

			if err := ch.Printer.PrintResource(toTrafficBudget(budget)); err != nil {
				return err
			}

			if ch.Printer.Format() == printer.Human && len(budget.Rules) > 0 {
				ch.Printer.Printf("\n%s\n", printer.Bold("Rules:"))
				rules := make([]*TrafficRuleDisplay, len(budget.Rules))
				for i := range budget.Rules {
					rules[i] = toTrafficRule(&budget.Rules[i])
				}
				return ch.Printer.PrintResource(rules)
			}

			return nil
		},
	}

	return cmd
}
