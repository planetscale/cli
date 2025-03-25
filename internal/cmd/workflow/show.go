package workflow

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <number>",
		Short: "Show a specific workflow for a PlanetScale database",
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching workflow %s for database %s…", printer.BoldBlue(number), printer.BoldBlue(db)))
			defer end()

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
			end()

			return ch.Printer.PrintResource(toWorkflow(workflow))
		},
	}

	return cmd
}

type Workflow struct {
	Number uint64 `header:"number"`
	Name   string `header:"name"`
	State  string `header:"state"`
	Actor  string `header:"actor"`
	Branch string `header:"branch"`

	Tables []string `header:"tables" json:"tables"`

	SourceKeyspace string `header:"source keyspace"`
	TargetKeyspace string `header:"target keyspace"`

	CreatedAt int64 `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`

	CopyDuration int64 `header:"copy duration,unixduration"`
	Duration     int64 `header:"total duration,unixduration" json:"duration"`

	TrafficServing string `header:"traffic serving"`

	FinishedAt *int64 `header:"finished at,timestamp(ms|utc|human)" json:"finished_at"`

	orig *ps.Workflow
}

func toWorkflow(w *ps.Workflow) *Workflow {
	var finishedAt *time.Time
	if w.CompletedAt != nil {
		finishedAt = w.CompletedAt
	} else if (w.CancelledAt) != nil {
		finishedAt = w.CancelledAt
	}

	duration := durationIfExists(&w.CreatedAt, finishedAt)
	copyDuration := durationIfExists(w.StartedAt, w.DataCopyCompletedAt)

	return &Workflow{
		Number:         w.Number,
		Name:           w.Name,
		State:          w.State,
		Actor:          w.Actor.Name,
		Branch:         w.Branch.Name,
		SourceKeyspace: w.SourceKeyspace.Name,
		TargetKeyspace: w.TargetKeyspace.Name,

		Tables: getTables(w),

		TrafficServing: getTrafficServingState(w),

		CreatedAt:    printer.GetMilliseconds(w.CreatedAt),
		CopyDuration: copyDuration.Milliseconds(),
		Duration:     duration.Milliseconds(),

		FinishedAt: printer.GetMillisecondsIfExists(finishedAt),

		orig: w,
	}
}

func getTables(w *ps.Workflow) []string {
	tables := make([]string, 0, len(w.Tables))

	for _, table := range w.Tables {
		tables = append(tables, table.Name)
	}

	return tables
}

func getTrafficServingState(w *ps.Workflow) string {
	if (w.PrimariesSwitched && w.ReplicasSwitched) || w.CompletedAt != nil {
		return "Primary & replicas"
	} else if w.PrimariesSwitched {
		return "Primary"
	} else if w.ReplicasSwitched {
		return "Replicas"
	} else {
		return ""
	}
}

func (w *Workflow) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(w.orig, "", " ")
}

func (w *Workflow) MarshalCSVValue() interface{} {
	return []*Workflow{w}
}
