package deployrequest

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// DeployRequestCmd encapsulates the commands for creatind and managing Deploy
// Requests.
func DeployRequestCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "deploy-request <command>",
		Short:             "Create, review, diff, revert, and manage deploy requests",
		Long:              "Create, review, diff, revert, and manage deploy requests.\n\nThis command is only supported for Vitess databases.",
		Aliases:           []string{"dr"},
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ApplyCmd(ch))
	cmd.AddCommand(CancelCmd(ch))
	cmd.AddCommand(CloseCmd(ch))
	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(DeployCmd(ch))
	cmd.AddCommand(DiffCmd(ch))
	cmd.AddCommand(EditCmd(ch))
	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ReviewCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(RevertCmd(ch))
	cmd.AddCommand(SkipRevertCmd(ch))

	return cmd
}

// DeployRequest returns a table-serializable deplo request model.
type DeployRequest struct {
	ID         string `header:"id" json:"id"`
	Number     uint64 `header:"number" json:"number"`
	Branch     string `header:"branch" json:"branch"`
	IntoBranch string `header:"into_branch" json:"into_branch"`

	Approved bool `header:"approved" json:"approved"`

	State string `header:"state" json:"state"`

	Deployment inlineDeployment `header:"inline" json:"deployment"`
	CreatedAt  string           `header:"created_at" json:"created_at"`
	UpdatedAt  string           `header:"updated_at" json:"updated_at"`
	ClosedAt   string           `header:"closed_at" json:"closed_at"`

	orig *planetscale.DeployRequest
}

type inlineDeployment struct {
	State              string `header:"deploy state" json:"state"`
	Deployable         bool   `header:"deployable" json:"deployable"`
	InstantDDLEligible bool   `header:"instant ddl eligible" json:"instant_ddl_eligible"`

	QueuedAt   string `header:"queued_at" json:"queued_at"`
	StartedAt  string `header:"started_at" json:"started_at"`
	FinishedAt string `header:"finished_at" json:"finished_at"`

	orig *planetscale.Deployment
}

func (d *DeployRequest) MarshalCSVValue() interface{} {
	return []*DeployRequest{d}
}

// formatTimestamp formats a timestamp to human readable "X ago" format
func formatTimestamp(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}

	duration := time.Since(*t)

	switch {
	case duration < time.Minute:
		return "less than a minute ago"
	case duration < time.Hour:
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// formatTimestampRequired formats a required timestamp (non-pointer)
func formatTimestampRequired(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return formatTimestamp(&t)
}

func toInlineDeployment(d *planetscale.Deployment) inlineDeployment {
	if d == nil {
		return inlineDeployment{}
	}

	return inlineDeployment{
		State:              d.State,
		Deployable:         d.Deployable,
		InstantDDLEligible: d.InstantDDLEligible,
		QueuedAt:           formatTimestamp(d.QueuedAt),
		StartedAt:          formatTimestamp(d.StartedAt),
		FinishedAt:         formatTimestamp(d.FinishedAt),
		orig:               d,
	}
}

func toDeployRequest(dr *planetscale.DeployRequest) *DeployRequest {
	return &DeployRequest{
		ID:         dr.ID,
		Branch:     dr.Branch,
		IntoBranch: dr.IntoBranch,
		Number:     dr.Number,
		Approved:   dr.Approved,
		State:      dr.State,
		Deployment: toInlineDeployment(dr.Deployment),
		CreatedAt:  formatTimestampRequired(dr.CreatedAt),
		UpdatedAt:  formatTimestampRequired(dr.UpdatedAt),
		ClosedAt:   formatTimestamp(dr.ClosedAt),
		orig:       dr,
	}
}

func toDeployRequests(deployRequests []*planetscale.DeployRequest) []*DeployRequest {
	requests := make([]*DeployRequest, 0, len(deployRequests))

	for _, dr := range deployRequests {
		requests = append(requests, toDeployRequest(dr))
	}

	return requests
}

func (d *DeployRequest) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func (d *inlineDeployment) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}
