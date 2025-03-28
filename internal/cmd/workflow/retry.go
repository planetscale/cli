package workflow

import (
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func RetryCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retry <database> <number>",
		Short: "Retry a workflow that has been stopped or failed.",
		Long:  `Retry a workflow that has been stopped or failed. If the errors are restartable, the workflow will continue from where it left off. Otherwise, it will start over from the beginning.`,
		Args:  cmdutil.RequiredArgs("database", "number"),
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Retrying workflow %s in database %s…", printer.BoldBlue(printer.Number(number)), printer.BoldBlue(db)))
			defer end()

			workflow, err := client.Workflows.Retry(ctx, &ps.RetryWorkflowRequest{
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
				ch.Printer.Printf("Workflow %s in database %s has been retried.\n",
					printer.BoldBlue(printer.Number(workflow.Number)),
					printer.BoldBlue(db),
				)
				return nil
			}

			return ch.Printer.PrintResource(toWorkflow(workflow))
		},
	}

	return cmd
}
