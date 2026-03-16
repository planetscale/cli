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
		keyspace    string
		workflow    string
		includeLogs bool
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

			req := &ps.VtctldListWorkflowsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     flags.keyspace,
				Workflow:     flags.workflow,
			}
			if cmd.Flags().Changed("include-logs") {
				req.IncludeLogs = &flags.includeLogs
			}

			data, err := client.Vtctld.ListWorkflows(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace to list workflows for")
	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Filter by workflow name")
	cmd.Flags().BoolVar(&flags.includeLogs, "include-logs", true, "Include workflow logs in the response")
	cmd.MarkFlagRequired("keyspace") // nolint:errcheck

	return cmd
}
