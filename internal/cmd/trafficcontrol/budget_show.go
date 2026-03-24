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
			return ch.Printer.PrintResource(toTrafficBudget(budget))
		},
	}

	return cmd
}
