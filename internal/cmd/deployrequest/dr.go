package deployrequest

import (
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// DeployRequestCmd encapsulates the commands for creatind and managing Deploy
// Requests.
func DeployRequestCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "deploy-request <command>",
		Short:             "Create, approve, diff, and manage deploy requests",
		Aliases:           []string{"dr"},
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CloseCmd(ch))
	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(DeployCmd(ch))
	cmd.AddCommand(DiffCmd(ch))
	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ReviewCmd(ch))
	cmd.AddCommand(ShowCmd(ch))

	return cmd
}

// DeployRequest returns a table-serializable deplo request model.
type DeployRequest struct {
	ID         string `header:"id" json:"id"`
	Number     uint64 `header:"number" json:"number"`
	Branch     string `header:"branch,timestamp(ms|utc|human)" json:"branch"`
	IntoBranch string `header:"into_branch,timestamp(ms|utc|human)" json:"into_branch"`

	Approved bool `header:"approved" json:"approved"`
	Ready    bool `header:"ready" json:"ready"`

	DeploymentState     string `header:"deployment_state,n/a" json:"deployment_state"`
	State               string `header:"state" json:"state"`
	DeployabilityErrors string `header:"errors" json:"deployability_errors"`

	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
	ClosedAt  *int64 `header:"closed_at,timestamp(ms|utc|human),-" json:"closed_at"`
}

func (d *DeployRequest) MarshalCSVValue() interface{} {
	return []*DeployRequest{d}
}

func toDeployRequest(dr *planetscale.DeployRequest) *DeployRequest {
	return &DeployRequest{
		ID:                  dr.ID,
		Branch:              dr.Branch,
		IntoBranch:          dr.IntoBranch,
		Number:              dr.Number,
		Approved:            dr.Approved,
		State:               dr.State,
		DeploymentState:     dr.DeploymentState,
		DeployabilityErrors: dr.DeployabilityErrors,
		CreatedAt:           printer.GetMilliseconds(dr.CreatedAt),
		UpdatedAt:           printer.GetMilliseconds(dr.UpdatedAt),
		ClosedAt:            printer.GetMillisecondsIfExists(dr.ClosedAt),
	}
}

func toDeployRequests(deployRequests []*planetscale.DeployRequest) []*DeployRequest {
	requests := make([]*DeployRequest, 0, len(deployRequests))

	for _, dr := range deployRequests {
		requests = append(requests, toDeployRequest(dr))
	}

	return requests
}
