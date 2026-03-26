package trafficcontrol

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func RuleDeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database> <branch> <budget-id> <rule-id>",
		Short:   "Delete a traffic rule from a budget",
		Args:    cmdutil.RequiredArgs("database", "branch", "budget-id", "rule-id"),
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			budgetID := args[2]
			ruleID := args[3]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !force {
				if err := ch.Printer.ConfirmCommand(ruleID, "delete rule", "rule deletion"); err != nil {
					return err
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting traffic rule %s from budget %s",
				printer.BoldBlue(ruleID), printer.BoldBlue(budgetID)))
			defer end()

			err = client.TrafficRules.Delete(ctx, &ps.DeleteTrafficRuleRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				BudgetID:     budgetID,
				RuleID:       ruleID,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("traffic rule %s does not exist on budget %s in %s/%s (organization: %s)",
						printer.BoldBlue(ruleID), printer.BoldBlue(budgetID), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Traffic rule %s was successfully deleted from budget %s.\n",
					printer.BoldBlue(ruleID), printer.BoldBlue(budgetID))
				return nil
			}

			return ch.Printer.PrintResource(
				map[string]string{
					"result":    "rule deleted",
					"rule_id":   ruleID,
					"budget_id": budgetID,
					"database":  database,
					"branch":    branch,
				},
			)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete a rule without confirmation")
	return cmd
}
