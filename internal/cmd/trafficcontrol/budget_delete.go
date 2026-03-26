package trafficcontrol

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func BudgetDeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database> <branch> <budget-id>",
		Short:   "Delete a traffic budget",
		Args:    cmdutil.RequiredArgs("database", "branch", "budget-id"),
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			budgetID := args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !force {
				if err := ch.Printer.ConfirmCommand(budgetID, "delete budget", "budget deletion"); err != nil {
					return err
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting traffic budget %s from %s/%s",
				printer.BoldBlue(budgetID), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			err = client.TrafficBudgets.Delete(ctx, &ps.DeleteTrafficBudgetRequest{
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

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Traffic budget %s was successfully deleted from %s/%s.\n",
					printer.BoldBlue(budgetID), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(
				map[string]string{
					"result":    "budget deleted",
					"budget_id": budgetID,
					"database":  database,
					"branch":    branch,
				},
			)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete a budget without confirmation")
	return cmd
}
