package deployrequest

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

// QueueCmd lists deployments currently in the database deploy queue.
func QueueCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queue <database>",
		Short: "Show the deploy queue for a database",
		Args:  cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching deploy queue for %s", printer.BoldBlue(database)))
			defer end()

			deployments, err := client.DeployRequests.GetDeployQueue(ctx, &planetscale.GetDeployQueueRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(deployments) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Deploy queue for %s is empty.\n", printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(toDeployments(deployments))
		},
	}

	return cmd
}

type DeploymentRow struct {
	ID                 string `header:"id" json:"id"`
	Number             uint64 `header:"number" json:"deploy_request_number"`
	State              string `header:"state" json:"state"`
	IntoBranch         string `header:"into_branch" json:"into_branch"`
	Deployable         bool   `header:"deployable" json:"deployable"`
	AutoCutover        bool   `header:"auto_cutover" json:"auto_cutover"`
	AutoDeleteBranch   bool   `header:"auto_delete_branch" json:"auto_delete_branch"`
	InstantDDLEligible bool   `header:"instant_ddl_eligible" json:"instant_ddl_eligible"`
	QueuePaused        bool   `header:"queue_paused" json:"queue_paused"`
	CreatedAt          string `header:"created_at" json:"created_at"`
	QueuedAt           string `header:"queued_at" json:"queued_at"`
	StartedAt          string `header:"started_at" json:"started_at"`
	FinishedAt         string `header:"finished_at" json:"finished_at"`

	orig *planetscale.Deployment
}

func (d *DeploymentRow) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func (d *DeploymentRow) MarshalCSVValue() interface{} {
	return []*DeploymentRow{d}
}

func toDeployment(d *planetscale.Deployment) *DeploymentRow {
	if d == nil {
		return &DeploymentRow{}
	}
	return &DeploymentRow{
		ID:                 d.ID,
		Number:             d.DeployRequestNumber,
		State:              d.State,
		IntoBranch:         d.IntoBranch,
		Deployable:         d.Deployable,
		AutoCutover:        d.AutoCutover,
		AutoDeleteBranch:   d.AutoDeleteBranch,
		InstantDDLEligible: d.InstantDDLEligible,
		QueuePaused:        d.QueuePaused,
		CreatedAt:          formatTimestampRequired(d.CreatedAt),
		QueuedAt:           formatTimestamp(d.QueuedAt),
		StartedAt:          formatTimestamp(d.StartedAt),
		FinishedAt:         formatTimestamp(d.FinishedAt),
		orig:               d,
	}
}

func toDeployments(deployments []*planetscale.Deployment) []*DeploymentRow {
	rows := make([]*DeploymentRow, 0, len(deployments))
	for _, d := range deployments {
		rows = append(rows, toDeployment(d))
	}
	return rows
}
