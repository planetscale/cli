package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// AgentRunCancelCmd cancels a pending or running Insights Agent run.
func AgentRunCancelCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "cancel <database> <branch> <number>",
		Short:   "Cancel a pending or running Insights Agent run",
		Example: `  pscale insights agent runs cancel mydb main 12 --org myorg --force`,
		Args:    cmdutil.RequiredArgs("database", "branch", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]
			number, err := parseRunNumber(args[2])
			if err != nil {
				return err
			}

			if !force {
				if err := ch.Printer.ConfirmCommand(args[2], "cancel Insights Agent run", "cancellation of Insights Agent run"); err != nil {
					return err
				}
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Cancelling Insights Agent run #%d on %s/%s...",
				number, printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			run, err := client.InsightsAgent.CancelRun(ctx, &ps.CancelInsightsAgentRunRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Number:       number,
			})
			if err != nil {
				return agentRunNotFoundError(ch, err, database, branch, number)
			}
			end()

			switch ch.Printer.Format() {
			case printer.JSON:
				return ch.Printer.PrintJSON(run)
			case printer.Human:
				ch.Printer.Printf("Insights Agent run %s on %s/%s was cancelled.\n",
					printer.BoldBlue(fmt.Sprintf("#%d", number)), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			default:
				return ch.Printer.PrintResource([]*AgentRunRow{agentRunRow(run)})
			}
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Cancel without confirmation")

	return cmd
}
