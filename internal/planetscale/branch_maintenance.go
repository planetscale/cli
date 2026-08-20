package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
)

// RunBranchMaintenanceRequest encapsulates the request for running maintenance
// on a branch.
type RunBranchMaintenanceRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`

	// UpdatePostgresMinorVersion upgrades the branch to the latest PostgreSQL
	// minor version as part of the maintenance run.
	UpdatePostgresMinorVersion bool `json:"update_postgres_minor_version,omitempty"`
}

// BranchMaintenanceService is an interface for communicating with the
// PlanetScale branch maintenance API endpoint.
type BranchMaintenanceService interface {
	Run(context.Context, *RunBranchMaintenanceRequest) error
}

type branchMaintenanceService struct {
	client *Client
}

var _ BranchMaintenanceService = &branchMaintenanceService{}

// Run starts a maintenance run for the given branch.
func (s *branchMaintenanceService) Run(ctx context.Context, runReq *RunBranchMaintenanceRequest) error {
	req, err := s.client.newRequest(http.MethodPost,
		branchMaintenanceAPIPath(runReq.Organization, runReq.Database, runReq.Branch), runReq)
	if err != nil {
		return fmt.Errorf("error creating request for run branch maintenance: %w", err)
	}

	return s.client.do(ctx, req, nil)
}

func branchMaintenanceAPIPath(org, db, branch string) string {
	return path.Join(databasesAPIPath(org), db, "branches", branch, "maintenance")
}
