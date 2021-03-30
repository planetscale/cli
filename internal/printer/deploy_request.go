package printer

import "github.com/planetscale/planetscale-go/planetscale"

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
	DeployabilityErrors string `header:"deployability_errors" json:"deployability_errors"`

	Notes string `header:"notes" json:"notes"`

	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
	ClosedAt  *int64 `header:"closed_at,timestamp(ms|utc|human),-" json:"closed_at"`
}

func NewDeployRequestPrinter(dr *planetscale.DeployRequest) *DeployRequest {
	return &DeployRequest{
		ID:                  dr.ID,
		Branch:              dr.Branch,
		IntoBranch:          dr.IntoBranch,
		Notes:               dr.Notes,
		Number:              dr.Number,
		Approved:            dr.Approved,
		State:               dr.State,
		DeploymentState:     dr.DeploymentState,
		DeployabilityErrors: dr.DeployabilityErrors,
		CreatedAt:           getMilliseconds(dr.CreatedAt),
		UpdatedAt:           getMilliseconds(dr.UpdatedAt),
		ClosedAt:            getMillisecondsIfExists(dr.ClosedAt),
	}
}

func NewDeployRequestSlicePrinter(deployRequests []*planetscale.DeployRequest) []*DeployRequest {
	requests := make([]*DeployRequest, 0, len(deployRequests))

	for _, dr := range deployRequests {
		requests = append(requests, NewDeployRequestPrinter(dr))
	}

	return requests
}
