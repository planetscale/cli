package planetscale

import (
	"context"
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

// SchemaSnapshotDiffRequest is a request for getting the diff for a schema
// snapshot.
type DiffSchemaSnapshotRequest struct {
	SchemaSchemaSnapshotID string `json:"-"`
}

// SchemaSnapshotsService is an interface for	communicating with the PlanetScale
// Schema Snapshots API.
type SchemaSnapshotsService interface {
	Create(context.Context, *CreateSchemaSnapshotRequest) (*SchemaSnapshot, error)
	List(context.Context, *ListSchemaSnapshotsRequest) ([]*SchemaSnapshot, error)
	Get(context.Context, *GetSchemaSnapshotRequest) (*SchemaSnapshot, error)
	Diff(context.Context, *DiffSchemaSnapshotRequest) ([]*Diff, error)
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

	ss := &SchemaSnapshot{}
	if err := s.client.do(ctx, req, &ss); err != nil {
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

	ssr := schemaSnapshotsResponse{}
	if err := s.client.do(ctx, req, &ssr); err != nil {
		return nil, err
	}

	return ssr.SchemaSnapshots, nil
}

// Get returns a single schema snapshot.
func (s *schemaSnapshotsService) Get(ctx context.Context, getReq *GetSchemaSnapshotRequest) (*SchemaSnapshot, error) {
	req, err := s.client.newRequest(http.MethodGet, schemaSnapshotAPIPath(getReq.ID), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	ss := &SchemaSnapshot{}
	if err := s.client.do(ctx, req, &ss); err != nil {
		return nil, err
	}

	return ss, nil
}

func (s *schemaSnapshotsService) Diff(ctx context.Context, diffReq *DiffSchemaSnapshotRequest) ([]*Diff, error) {
	path := fmt.Sprintf("%s/diff", schemaSnapshotAPIPath(diffReq.SchemaSchemaSnapshotID))
	req, err := s.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	diffs := &diffResponse{}
	if err := s.client.do(ctx, req, &diffs); err != nil {
		return nil, err
	}

	return diffs.Diffs, nil
}

func schemaSnapshotsAPIPath(org, database, branch string) string {
	return fmt.Sprintf("%s/%s/schema-snapshots", databaseBranchesAPIPath(org, database), branch)
}

func schemaSnapshotAPIPath(id string) string {
	return fmt.Sprintf("/v1/schema-snapshots/%s", id)
}
