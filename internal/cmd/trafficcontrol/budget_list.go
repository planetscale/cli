package trafficcontrol

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func BudgetListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		fingerprint string
	}

	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List traffic budgets for a branch",
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching traffic budgets for %s/%s",
				printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			budgets, err := client.TrafficBudgets.List(ctx, &ps.ListTrafficBudgetsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Fingerprint:  flags.fingerprint,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if len(budgets) == 0 && ch.Printer.Format() == printer.Human {
				if flags.fingerprint != "" {
					ch.Printer.Printf("No traffic budgets in %s/%s match fingerprint %s.\n",
						printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(flags.fingerprint))
				} else {
					ch.Printer.Printf("No traffic budgets exist in %s/%s.\n",
						printer.BoldBlue(database), printer.BoldBlue(branch))
				}
				return nil
			}

			return ch.Printer.PrintResource(ToTrafficBudgets(budgets))
		},
	}

	cmd.Flags().StringVar(&flags.fingerprint, "fingerprint", "",
		"Only list budgets with a rule for this query fingerprint")

	return cmd
}
