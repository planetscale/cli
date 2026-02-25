package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func StopWorkflowCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
	}

	cmd := &cobra.Command{
		Use:   "stop-workflow <database> <branch> <workflow>",
		Short: "Stop a workflow on a branch",
		Args:  cmdutil.RequiredArgs("database", "branch", "workflow"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, workflow := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Stopping workflow %s on %s/%s\u2026",
					printer.BoldBlue(workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.Vtctld.StopWorkflow(ctx, &ps.VtctldStopWorkflowRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Workflow:     workflow,
				Keyspace:     flags.keyspace,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace the workflow belongs to")
	cmd.MarkFlagRequired("keyspace") // nolint:errcheck

	return cmd
}
