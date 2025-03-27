package workflow

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func CancelCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "cancel <database> <number>",
		Short: "Cancel a workflow that is in progress",
		Long: `Cancel a workflow that is in progress. This stops the workflow and 
marks it as cancelled, allowing you to start a new workflow if needed.`,
		Args: cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db, num := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			var number uint64
			number, err = strconv.ParseUint(num, 10, 64)
			if err != nil {
				return err
			}

			if !force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf("cannot cancel workflow with the output format %q (run with -force to override)", ch.Printer.Format())
				}

				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm cancellation (run with -force to override)")
				}

				prompt := &survey.Confirm{
					Message: "Are you sure you want to cancel this workflow?",
					Default: false,
				}

				var confirm bool
				err = survey.AskOne(prompt, &confirm)
				if err != nil {
					if err == terminal.InterruptErr {
						os.Exit(0)
					} else {
						return err
					}
				}

				if !confirm {
					return errors.New("cancellation not confirmed, skipping workflow cancellation")
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Cancelling workflow %s in database %s…", printer.BoldBlue(number), printer.BoldBlue(db)))
			defer end()

			workflow, err := client.Workflows.Cancel(ctx, &ps.CancelWorkflowRequest{
				Organization:   ch.Config.Organization,
				Database:       db,
				WorkflowNumber: number,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s or workflow %s does not exist in organization %s",
						printer.BoldBlue(db), printer.BoldBlue(number), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Workflow %s in database %s has been cancelled.\n",
					printer.BoldBlue(workflow.Name),
					printer.BoldBlue(db),
				)
				return nil
			}

			return ch.Printer.PrintResource(toWorkflow(workflow))
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force cancel the workflow without confirmation")

	return cmd
}
