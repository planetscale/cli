package workflow

import (
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func VerifyDataCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-data <database> <number>",
		Short: "Verify data consistency for a specific workflow in a PlanetScale database",
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Verifying data for workflow %s in database %s...", printer.BoldBlue(number), printer.BoldBlue(db)))
			defer end()

			workflow, err := client.Workflows.VerifyData(ctx, &ps.VerifyDataWorkflowRequest{
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
				ch.Printer.Println("Successfully started data verification for workflow %s in database %s.",
					printer.BoldBlue(workflow.Name),
					printer.BoldBlue(db),
					printer.Bold(workflow.State))
				return nil
			}

			return ch.Printer.PrintResource(toWorkflow(workflow))
		},
	}

	return cmd
}
