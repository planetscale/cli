package workflow

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <database>",
		Short: "List all of the workflows for a PlanetScale database",
		Args:  cmdutil.RequiredArgs("database"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching workflows for database %s...", printer.BoldBlue(db)))
			defer end()

			workflows, err := client.Workflows.List(ctx, &ps.ListWorkflowsRequest{
				Organization: ch.Config.Organization,
				Database:     db,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(db), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toMinimalWorkflows(workflows))
		},
	}

	return cmd
}

type MinimalWorkflow struct {
	Number      int64  `header:"#"`
	Name        string `header:"name"`
	State       string `header:"state"`
	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	Duration    *int64 `header:"duration,timestamp(ms|utc|human)" json:"duration"`
	CompletedAt *int64 `header:"completed_at,timestamp(ms|utc|human)" json:"completed_at"`

	orig *ps.Workflow
}

func toMinimalWorkflows(workflows []*ps.Workflow) []*MinimalWorkflow {
	minimalWorkflows := make([]*MinimalWorkflow, 0, len(workflows))

	for _, w := range workflows {
		minimalWorkflows = append(minimalWorkflows, toMinimalWorkflow(w))
	}

	return minimalWorkflows
}

func toMinimalWorkflow(w *ps.Workflow) *MinimalWorkflow {
	return &MinimalWorkflow{
		orig: w,
	}
}
