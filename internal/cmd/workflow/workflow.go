package workflow

import (
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func WorkflowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "workflow <command>",
		Short:             "Manage the workflows for PlanetScale databases",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ListCmd(ch))

	return cmd
}

type MinimalWorkflow struct {
	Number      uint64 `header:"Number"`
	Name        string `header:"name"`
	State       string `header:"state"`
	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	CompletedAt *int64 `header:"completed_at,timestamp(ms|utc|human)" json:"completed_at"`
	Duration    int64  `header:"duration,unixduration" json:"duration"`

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
	duration := durationIfExists(w.CreatedAt, w.CompletedAt)

	return &MinimalWorkflow{
		Number:      w.Number,
		Name:        w.Name,
		State:       w.State,
		CreatedAt:   printer.GetMilliseconds(w.CreatedAt),
		CompletedAt: printer.GetMillisecondsIfExists(w.CompletedAt),
		Duration:    duration.Milliseconds(),
		orig:        w,
	}
}

func durationIfExists(start time.Time, end *time.Time) time.Duration {
	var duration time.Duration

	if end != nil {
		duration = end.Sub(start)
	}

	return duration
}
