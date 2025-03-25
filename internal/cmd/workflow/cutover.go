package workflow

import (
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func CutoverCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cutover <database> <number>",
		Short: "Completes the workflow, cutting over all traffic to the target keyspace.",
		Long:  `Completes the workflow, cutting over all traffic to the target keyspace. This will delete the moved tables from the source keyspace and replication between the source and target keyspace will end. If your application has errors related to the moved tables, use the reverse-cutover command to temporarily restore the routing rules.`,
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Completing workflow workflow %s in database %s…", printer.BoldBlue(number), printer.BoldBlue(db)))
			defer end()

			workflow, err := client.Workflows.Cutover(ctx, &ps.CutoverWorkflowRequest{
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

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Workflow %s successfully completed.\n",
					printer.BoldBlue(workflow.Name),
					printer.BoldBlue(db),
				)
				return nil
			}

			return ch.Printer.PrintResource(toWorkflow(workflow))
		},
	}

	return cmd
}
