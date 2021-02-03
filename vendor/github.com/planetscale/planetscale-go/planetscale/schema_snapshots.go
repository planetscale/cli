package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// CreateSchemaSnapshotRequest reflects the request needed to make a schema
// snapshot on a database branch.
type CreateSchemaSnapshotRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

// ListSchemaSnapshotsRequest reflects the request for listing schema snapshots.
type ListSchemaSnapshotsRequest struct {
	Organization string
	Database     string
	Branch       string
}

// GetSchemaSnapshotRequest reflects the request for getting a single schema
// snapshot.
type GetSchemaSnapshotRequest struct {
	ID string `json:"-"`
}

// SchemaSnapshot reflects the data model for a schema snapshot of a database
// branch.
type SchemaSnapshot struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SchemaSnapshotRequestDeployRequest is a request for requesting a deploy of a schema
// snapshot.
type SchemaSnapshotRequestDeployRequest struct {
	SchemaSnapshotID string `json:"-"`
	IntoBranch       string `json:"into_branch,omitempty"`
	Notes            string `json:"notes"`
}

// SchemaSnapshotsService is an interface for	communicating with the PlanetScale
// Schema Snapshots API.
type SchemaSnapshotsService interface {
	Create(context.Context, *CreateSchemaSnapshotRequest) (*SchemaSnapshot, error)
	List(context.Context, *ListSchemaSnapshotsRequest) ([]*SchemaSnapshot, error)
	Get(context.Context, *GetSchemaSnapshotRequest) (*SchemaSnapshot, error)
	RequestDeploy(context.Context, *SchemaSnapshotRequestDeployRequest) (*DeployRequest, error)
}

type schemaSnapshotsService struct {
	client *Client
}

type schemaSnapshotsResponse struct {
	SchemaSnapshots []*SchemaSnapshot `json:"data"`
}

var _ SchemaSnapshotsService = &schemaSnapshotsService{}

// NewSchemaSnapshotsService creates an instance of the schema snapshot service.
func NewSchemaSnapshotsService(client *Client) *schemaSnapshotsService {
	return &schemaSnapshotsService{
		client: client,
	}
}

// Create creates a new schema snapshot for a database branch.
func (s *schemaSnapshotsService) Create(ctx context.Context, createReq *CreateSchemaSnapshotRequest) (*SchemaSnapshot, error) {
	req, err := s.client.newRequest(http.MethodPost, schemaSnapshotsAPIPath(createReq.Organization, createReq.Database, createReq.Branch), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	res, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	ss := &SchemaSnapshot{}
	err = json.NewDecoder(res.Body).Decode(ss)
	if err != nil {
		return nil, err
	}
	return ss, nil
}

// List returns all the schema snapshots for a database branch.
func (s *schemaSnapshotsService) List(ctx context.Context, listReq *ListSchemaSnapshotsRequest) ([]*SchemaSnapshot, error) {
	req, err := s.client.newRequest(http.MethodGet, schemaSnapshotsAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	res, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	schemaSnapshotsResponse := schemaSnapshotsResponse{}
	err = json.NewDecoder(res.Body).Decode(&schemaSnapshotsResponse)
	if err != nil {
		return nil, err
	}

	return schemaSnapshotsResponse.SchemaSnapshots, nil
}

// Get returns a single schema snapshot.
func (s *schemaSnapshotsService) Get(ctx context.Context, getReq *GetSchemaSnapshotRequest) (*SchemaSnapshot, error) {
	req, err := s.client.newRequest(http.MethodGet, schemaSnapshotAPIPath(getReq.ID), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	res, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	ss := &SchemaSnapshot{}
	err = json.NewDecoder(res.Body).Decode(ss)
	if err != nil {
		return nil, err
	}

	return ss, nil
}

// RequestDeploy requests a deploy of a schema snapshot.
func (s *schemaSnapshotsService) RequestDeploy(ctx context.Context, deployReq *SchemaSnapshotRequestDeployRequest) (*DeployRequest, error) {
	req, err := s.client.newRequest(http.MethodPost, fmt.Sprintf("%s/deploy-requests", schemaSnapshotAPIPath(deployReq.SchemaSnapshotID)), deployReq)
	if err != nil {
		return nil, err
	}

	res, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	dr := &DeployRequest{}
	err = json.NewDecoder(res.Body).Decode(dr)
	if err != nil {
		return nil, err
	}

	return dr, nil
}

func schemaSnapshotsAPIPath(org, database, branch string) string {
	return fmt.Sprintf("%s/%s/schema-snapshots", databaseBranchesAPIPath(org, database), branch)
}

func schemaSnapshotAPIPath(id string) string {
	return fmt.Sprintf("/v1/schema-snapshots/%s", id)
}
