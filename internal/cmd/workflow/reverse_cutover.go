package workflow

import (
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ReverseCutoverCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reverse-cutover <database> <number>",
		Short: "Reverse the cutover of a workflow back to the source keyspace by reverting the routing rules.",
		Long: `Reverse the cutover of a workflow, redirecting traffic back to the source keyspace.  
This is useful if your application has errors and you need to rollback after a cutover has been completed.`,
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Reversing cutover for workflow %s in database %s…", printer.BoldBlue(printer.Number(number)), printer.BoldBlue(db)))
			defer end()

			workflow, err := client.Workflows.ReverseCutover(ctx, &ps.ReverseCutoverWorkflowRequest{
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
				ch.Printer.Printf("Cutover reversed for workflow %s in database %s.\n",
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
