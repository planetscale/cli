package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// DatabaseBranch represents a database branch.
type DatabaseBranch struct {
	Name         string    `json:"name"`
	Notes        string    `json:"notes"`
	ParentBranch string    `json:"parent_branch,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Status       string    `json:"status,omitempty"`
}

type databaseBranchesResponse struct {
	Branches []*DatabaseBranch `json:"data"`
}

// CreateDatabaseBranchRequest encapsulates the request for creating a new
// database branch
type CreateDatabaseBranchRequest struct {
	Organization string
	Database     string
	Branch       *DatabaseBranch `json:"branch"`
}

// ListDatabaseBranchesRequest encapsulates the request for listing the branches
// of a database.
type ListDatabaseBranchesRequest struct {
	Organization string
	Database     string
}

// GetDatabaseBranchRequest encapsulates the request for getting a single
// database branch for a database.
type GetDatabaseBranchRequest struct {
	Organization string
	Database     string
	Branch       string
}

// DeleteDatabaseRequest encapsulates the request for deleting a database branch
// from a database.
type DeleteDatabaseBranchRequest struct {
	Organization string
	Database     string
	Branch       string
}

// GetDatabaseBranchStatusRequest encapsulates the request for getting the status
// of a specific database branch.
type GetDatabaseBranchStatusRequest struct {
	Organization string
	Database     string
	Branch       string
}

// ListDeployRequestsRequest gets the deploy requests for a specific database
// branch.
type ListDeployRequestsRequest struct {
	Organization string
	Database     string
	Branch       string
}

type deployRequestsResponse struct {
	DeployRequests []*DeployRequest `json:"data"`
}

// DatabaseBranchesService is an interface for communicating with the PlanetScale
// Database Branch API endpoint.
type DatabaseBranchesService interface {
	Create(context.Context, *CreateDatabaseBranchRequest) (*DatabaseBranch, error)
	List(context.Context, *ListDatabaseBranchesRequest) ([]*DatabaseBranch, error)
	Get(context.Context, *GetDatabaseBranchRequest) (*DatabaseBranch, error)
	Delete(context.Context, *DeleteDatabaseBranchRequest) error
	GetStatus(context.Context, *GetDatabaseBranchStatusRequest) (*DatabaseBranchStatus, error)
	ListDeployRequests(context.Context, *ListDeployRequestsRequest) ([]*DeployRequest, error)
}

type databaseBranchesService struct {
	client *Client
}

// DatabaseBranchStatus represents the status of a PlanetScale database branch.
type DatabaseBranchStatus struct {
	DeployPhase string `json:"deploy_phase"`
	GatewayHost string `json:"mysql_gateway_host"`
	GatewayPort int    `json:"mysql_gateway_port"`
	User        string `json:"mysql_gateway_user"`
	Password    string `json:"mysql_gateway_pass"`
}

var _ DatabaseBranchesService = &databaseBranchesService{}

func NewDatabaseBranchesService(client *Client) *databaseBranchesService {
	return &databaseBranchesService{
		client: client,
	}
}

// Create creates a new branch for an organization's database.
func (ds *databaseBranchesService) Create(ctx context.Context, createReq *CreateDatabaseBranchRequest) (*DatabaseBranch, error) {
	path := databaseBranchesAPIPath(createReq.Organization, createReq.Database)

	req, err := ds.client.newRequest(http.MethodPost, path, createReq)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for branch database")
	}
	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	dbBranch := &DatabaseBranch{}
	err = json.NewDecoder(res.Body).Decode(&dbBranch)

	if err != nil {
		return nil, err
	}

	return dbBranch, nil
}

// Get returns a database branch for an organization's database.
func (ds *databaseBranchesService) Get(ctx context.Context, getReq *GetDatabaseBranchRequest) (*DatabaseBranch, error) {
	path := fmt.Sprintf("%s/%s", databaseBranchesAPIPath(getReq.Organization, getReq.Database), getReq.Branch)
	req, err := ds.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	dbBranch := &DatabaseBranch{}
	err = json.NewDecoder(res.Body).Decode(&dbBranch)

	if err != nil {
		return nil, err
	}

	return dbBranch, nil
}

// List returns all of the branches for an organization's
// database.
func (ds *databaseBranchesService) List(ctx context.Context, listReq *ListDatabaseBranchesRequest) ([]*DatabaseBranch, error) {
	req, err := ds.client.newRequest(http.MethodGet, databaseBranchesAPIPath(listReq.Organization, listReq.Database), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	dbBranches := &databaseBranchesResponse{}
	err = json.NewDecoder(res.Body).Decode(&dbBranches)

	if err != nil {
		return nil, err
	}

	return dbBranches.Branches, nil
}

// Delete deletes a database branch from an organization's database.
func (ds *databaseBranchesService) Delete(ctx context.Context, deleteReq *DeleteDatabaseBranchRequest) error {
	path := fmt.Sprintf("%s/%s", databaseBranchesAPIPath(deleteReq.Organization, deleteReq.Database), deleteReq.Branch)
	req, err := ds.client.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return errors.Wrap(err, "error creating request for delete branch")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

// Status returns the status of a specific database branch
func (ds *databaseBranchesService) GetStatus(ctx context.Context, statusReq *GetDatabaseBranchStatusRequest) (*DatabaseBranchStatus, error) {
	path := fmt.Sprintf("%s/%s/status", databaseBranchesAPIPath(statusReq.Organization, statusReq.Database), statusReq.Branch)
	req, err := ds.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for branch status")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	status := &DatabaseBranchStatus{}
	err = json.NewDecoder(res.Body).Decode(&status)

	if err != nil {
		return nil, err
	}

	return status, nil
}

func (ds *databaseBranchesService) ListDeployRequests(ctx context.Context, listReq *ListDeployRequestsRequest) ([]*DeployRequest, error) {
	path := fmt.Sprintf("%s/deploy-requests", databaseBranchAPIPath(listReq.Organization, listReq.Database, listReq.Branch))
	req, err := ds.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	deployRequestsResponse := &deployRequestsResponse{}
	err = json.NewDecoder(res.Body).Decode(deployRequestsResponse)
	if err != nil {
		return nil, err
	}

	return deployRequestsResponse.DeployRequests, nil
}

func databaseBranchesAPIPath(org, db string) string {
	return fmt.Sprintf("%s/%s/branches", databasesAPIPath(org), db)
}

func databaseBranchAPIPath(org, db, branch string) string {
	return fmt.Sprintf("%s/%s", databaseBranchesAPIPath(org, db), branch)
}
