package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ListWorkflowsCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
		workflow string
	}

	cmd := &cobra.Command{
		Use:   "list-workflows <database> <branch>",
		Short: "List vtctld workflows for a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Fetching workflows for %s/%s\u2026",
					printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.Vtctld.ListWorkflows(ctx, &ps.VtctldListWorkflowsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     flags.keyspace,
				Workflow:     flags.workflow,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace to list workflows for")
	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Filter by workflow name")
	cmd.MarkFlagRequired("keyspace") // nolint:errcheck

	return cmd
}
