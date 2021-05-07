package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// DatabaseBranch represents a database branch.
type DatabaseBranch struct {
	Name         string    `json:"name"`
	Notes        string    `json:"notes"`
	ParentBranch string    `json:"parent_branch"`
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
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       *DatabaseBranch
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

// DiffBranchRequest encapsulates a request for getting the diff for a branch.
type DiffBranchRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

// BranchSchemaRequest encapsulates a request for getting a branch's schema.
type BranchSchemaRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

// RefreshSchemaRequest reflects the request needed to refresh a schema
// snapshot on a database branch.
type RefreshSchemaRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

// DatabaseBranchesService is an interface for communicating with the PlanetScale
// Database Branch API endpoint.
type DatabaseBranchesService interface {
	Create(context.Context, *CreateDatabaseBranchRequest) (*DatabaseBranch, error)
	List(context.Context, *ListDatabaseBranchesRequest) ([]*DatabaseBranch, error)
	Get(context.Context, *GetDatabaseBranchRequest) (*DatabaseBranch, error)
	Delete(context.Context, *DeleteDatabaseBranchRequest) error
	GetStatus(context.Context, *GetDatabaseBranchStatusRequest) (*DatabaseBranchStatus, error)
	Diff(context.Context, *DiffBranchRequest) ([]*Diff, error)
	Schema(context.Context, *BranchSchemaRequest) ([]*Diff, error)
	RefreshSchema(context.Context, *RefreshSchemaRequest) error
}

type databaseBranchesService struct {
	client *Client
}

type DatabaseBranchCredentials struct {
	GatewayHost string `json:"mysql_gateway_host"`
	GatewayPort int    `json:"mysql_gateway_port"`
	User        string `json:"mysql_gateway_user"`
	Password    string `json:"mysql_gateway_pass"`
}

// DatabaseBranchStatus represents the status of a PlanetScale database branch.
type DatabaseBranchStatus struct {
	Ready       bool                      `json:"ready"`
	Credentials DatabaseBranchCredentials `json:"credentials"`
}

var _ DatabaseBranchesService = &databaseBranchesService{}

func NewDatabaseBranchesService(client *Client) *databaseBranchesService {
	return &databaseBranchesService{
		client: client,
	}
}

func (d *databaseBranchesService) Diff(ctx context.Context, diffReq *DiffBranchRequest) ([]*Diff, error) {
	path := fmt.Sprintf("%s/diff", databaseBranchAPIPath(diffReq.Organization, diffReq.Database, diffReq.Branch))
	req, err := d.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	diffs := &diffResponse{}
	if err := d.client.do(ctx, req, &diffs); err != nil {
		return nil, err
	}

	return diffs.Diffs, nil
}

// schemaResponse returns the schemas
type schemaResponse struct {
	Schemas []*Diff `json:"data"`
}

func (d *databaseBranchesService) Schema(ctx context.Context, schemaReq *BranchSchemaRequest) ([]*Diff, error) {
	path := fmt.Sprintf("%s/schema", databaseBranchAPIPath(schemaReq.Organization, schemaReq.Database, schemaReq.Branch))
	req, err := d.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	schemas := &schemaResponse{}
	if err := d.client.do(ctx, req, &schemas); err != nil {
		return nil, err
	}

	return schemas.Schemas, nil
}

// Create creates a new branch for an organization's database.
func (d *databaseBranchesService) Create(ctx context.Context, createReq *CreateDatabaseBranchRequest) (*DatabaseBranch, error) {
	path := databaseBranchesAPIPath(createReq.Organization, createReq.Database)

	req, err := d.client.newRequest(http.MethodPost, path, createReq.Branch)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for branch database")
	}

	dbBranch := &DatabaseBranch{}
	if err := d.client.do(ctx, req, &dbBranch); err != nil {
		return nil, err
	}

	return dbBranch, nil
}

// Get returns a database branch for an organization's database.
func (d *databaseBranchesService) Get(ctx context.Context, getReq *GetDatabaseBranchRequest) (*DatabaseBranch, error) {
	path := fmt.Sprintf("%s/%s", databaseBranchesAPIPath(getReq.Organization, getReq.Database), getReq.Branch)
	req, err := d.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	dbBranch := &DatabaseBranch{}
	if err := d.client.do(ctx, req, &dbBranch); err != nil {
		return nil, err
	}

	return dbBranch, nil
}

// List returns all of the branches for an organization's
// database.
func (d *databaseBranchesService) List(ctx context.Context, listReq *ListDatabaseBranchesRequest) ([]*DatabaseBranch, error) {
	req, err := d.client.newRequest(http.MethodGet, databaseBranchesAPIPath(listReq.Organization, listReq.Database), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	dbBranches := &databaseBranchesResponse{}
	if err := d.client.do(ctx, req, &dbBranches); err != nil {
		return nil, err
	}

	return dbBranches.Branches, nil
}

// Delete deletes a database branch from an organization's database.
func (d *databaseBranchesService) Delete(ctx context.Context, deleteReq *DeleteDatabaseBranchRequest) error {
	path := fmt.Sprintf("%s/%s", databaseBranchesAPIPath(deleteReq.Organization, deleteReq.Database), deleteReq.Branch)
	req, err := d.client.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return errors.Wrap(err, "error creating request for delete branch")
	}

	err = d.client.do(ctx, req, nil)
	return err
}

// Status returns the status of a specific database branch
func (d *databaseBranchesService) GetStatus(ctx context.Context, statusReq *GetDatabaseBranchStatusRequest) (*DatabaseBranchStatus, error) {
	path := fmt.Sprintf("%s/%s/status", databaseBranchesAPIPath(statusReq.Organization, statusReq.Database), statusReq.Branch)
	req, err := d.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for branch status")
	}

	status := &DatabaseBranchStatus{}
	if err := d.client.do(ctx, req, &status); err != nil {
		return nil, err
	}

	return status, nil
}

// RefreshSchema refreshes the schema for a
func (d *databaseBranchesService) RefreshSchema(ctx context.Context, refreshReq *RefreshSchemaRequest) error {
	path := fmt.Sprintf("%s/%s/refresh-schema", databaseBranchesAPIPath(refreshReq.Organization, refreshReq.Database), refreshReq.Branch)
	req, err := d.client.newRequest(http.MethodPost, path, nil)
	if err != nil {
		return errors.Wrap(err, "error creating http request")
	}

	if err := d.client.do(ctx, req, nil); err != nil {
		return err
	}

	return nil
}

func databaseBranchesAPIPath(org, db string) string {
	return fmt.Sprintf("%s/%s/branches", databasesAPIPath(org), db)
}

func databaseBranchAPIPath(org, db, branch string) string {
	return fmt.Sprintf("%s/%s", databaseBranchesAPIPath(org, db), branch)
}
