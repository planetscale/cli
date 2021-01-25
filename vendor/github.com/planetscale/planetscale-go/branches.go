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
	Branch *DatabaseBranch `json:"branch"`
}

// DatabaseBranchesService is an interface for communicating with the PlanetScale
// Database Branch API endpoint.
type DatabaseBranchesService interface {
	Create(context.Context, string, string, *CreateDatabaseBranchRequest) (*DatabaseBranch, error)
	List(context.Context, string, string) ([]*DatabaseBranch, error)
	Get(context.Context, string, string, string) (*DatabaseBranch, error)
	Delete(context.Context, string, string, string) error
	Status(context.Context, string, string, string) (*DatabaseBranchStatus, error)
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
func (ds *databaseBranchesService) Create(ctx context.Context, org, db string, createReq *CreateDatabaseBranchRequest) (*DatabaseBranch, error) {
	path := databaseBranchesAPIPath(org, db)

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
func (ds *databaseBranchesService) Get(ctx context.Context, org, db, branch string) (*DatabaseBranch, error) {
	path := fmt.Sprintf("%s/%s", databaseBranchesAPIPath(org, db), branch)
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
func (ds *databaseBranchesService) List(ctx context.Context, org, db string) ([]*DatabaseBranch, error) {
	req, err := ds.client.newRequest(http.MethodGet, databaseBranchesAPIPath(org, db), nil)
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
func (ds *databaseBranchesService) Delete(ctx context.Context, org, db, branch string) error {
	path := fmt.Sprintf("%s/%s", databaseBranchesAPIPath(org, db), branch)
	req, err := ds.client.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return errors.Wrap(err, "error creating request for delete branch")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return errors.New("branch not found")
	}

	return nil
}

// Status returns the status of a specific database branch
func (ds *databaseBranchesService) Status(ctx context.Context, org, db, branch string) (*DatabaseBranchStatus, error) {
	path := fmt.Sprintf("%s/%s/status", databaseBranchesAPIPath(org, db), branch)
	req, err := ds.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for branch status")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	status := &DatabaseBranchStatus{}
	err = json.NewDecoder(res.Body).Decode(&status)

	if err != nil {
		return nil, err
	}

	return status, nil
}

func databaseBranchesAPIPath(org, db string) string {
	return fmt.Sprintf("%s/%s/branches", databasesAPIPath(org), db)
}
