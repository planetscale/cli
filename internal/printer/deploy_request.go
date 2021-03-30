package printer

import "github.com/planetscale/planetscale-go/planetscale"

// DeployRequest returns a table-serializable deplo request model.
type DeployRequest struct {
	Number     uint64 `header:"number" json:"number"`
	Branch     string `header:"branch,timestamp(ms|utc|human)" json:"branch"`
	IntoBranch string `header:"into_branch,timestamp(ms|utc|human)" json:"into_branch"`

	Approved        bool `header:"approved" json:"approved"`
	Deployed        bool `header:"deployed" json:"deployed"`
	DeploymentState bool `header:"deployment_state" json:"deployment_state"`

	DeployabilityErrors string `header:"deployability_errors" json:"deployability_errors"`
	Ready               bool   `header:"ready" json:"ready"`
	State               string `header:"state" json:"state"`

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
		Deployed:            dr.Deployed,
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
