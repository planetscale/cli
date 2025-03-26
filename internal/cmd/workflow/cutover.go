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

func CutoverCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

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

			if !force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf("cannot cutover with the output format %q (run with -force to override)", ch.Printer.Format())
				}

				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm cutover (run with -force to override)")
				}

				workflow, err := client.Workflows.Get(ctx, &ps.GetWorkflowRequest{
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

				confirmationMessage := fmt.Sprintf("Are you sure you want to the cutover? This will delete the moved tables for %s and replication between %s and %s will end.", workflow.SourceKeyspace.Name, workflow.SourceKeyspace.Name, workflow.TargetKeyspace.Name)
				prompt := &survey.Confirm{
					Message: confirmationMessage,
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
					return errors.New("cancelled cutover")
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Completing workflow %s in database %s…", printer.BoldBlue(number), printer.BoldBlue(db)))
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
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Workflow %s successfully completed.\n",
					printer.BoldBlue(workflow.Name),
				)
				return nil
			}

			return ch.Printer.PrintResource(toWorkflow(workflow))
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force thecutover without prompting for confirmation.")

	return cmd
}
