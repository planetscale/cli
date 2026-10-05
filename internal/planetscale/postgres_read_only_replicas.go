package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// PostgresDedicatedReadReplica represents a dedicated read replica for a Postgres branch.
type PostgresDedicatedReadReplica struct {
	ID                           string               `json:"id"`
	Name                         string               `json:"name"`
	State                        string               `json:"state"`
	Replicas                     int                  `json:"replicas"`
	ClusterName                  string               `json:"cluster_name"`
	ClusterDisplayName           string               `json:"cluster_display_name"`
	AccessHostURL                string               `json:"access_host_url"`
	PrivateAccessHostURL         string               `json:"private_access_host_url"`
	PrivateConnectionServiceName *string              `json:"private_connection_service_name"`
	CreatedAt                    time.Time            `json:"created_at"`
	UpdatedAt                    time.Time            `json:"updated_at"`
	ReadyAt                      *time.Time           `json:"ready_at"`
	Ready                        bool                 `json:"ready"`
	Actor                        Actor                `json:"actor"`
	Region                       Region               `json:"region"`
	Parameters                   []*PostgresParameter `json:"parameters"`
}

// ListPostgresDedicatedReadReplicasRequest encapsulates listing dedicated read replicas.
type ListPostgresDedicatedReadReplicasRequest struct {
	Organization string
	Database     string
	Branch       string
}

// GetPostgresDedicatedReadReplicaRequest encapsulates getting a dedicated read replica by name.
type GetPostgresDedicatedReadReplicaRequest struct {
	Organization string
	Database     string
	Branch       string
	Replica      string
}

// CreatePostgresDedicatedReadReplicaRequest encapsulates creating a dedicated read replica.
type CreatePostgresDedicatedReadReplicaRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Name         string `json:"name"`
	Region       string `json:"region"`
	Replicas     *int   `json:"replicas,omitempty"`
	ClusterSize  string `json:"cluster_size,omitempty"`
}

// UpdatePostgresDedicatedReadReplicaRequest encapsulates updating a dedicated read replica.
type UpdatePostgresDedicatedReadReplicaRequest struct {
	Organization string                       `json:"-"`
	Database     string                       `json:"-"`
	Branch       string                       `json:"-"`
	Replica      string                       `json:"-"`
	Replicas     *int                         `json:"replicas,omitempty"`
	ClusterSize  string                       `json:"cluster_size,omitempty"`
	Parameters   map[string]map[string]string `json:"parameters,omitempty"`
}

// DeletePostgresDedicatedReadReplicaRequest encapsulates deleting a dedicated read replica.
type DeletePostgresDedicatedReadReplicaRequest struct {
	Organization string
	Database     string
	Branch       string
	Replica      string
}

// PostgresDedicatedReadReplicasService is an interface for the Postgres dedicated
// read replicas API.
type PostgresDedicatedReadReplicasService interface {
	List(context.Context, *ListPostgresDedicatedReadReplicasRequest) ([]*PostgresDedicatedReadReplica, error)
	Get(context.Context, *GetPostgresDedicatedReadReplicaRequest) (*PostgresDedicatedReadReplica, error)
	Create(context.Context, *CreatePostgresDedicatedReadReplicaRequest) (*PostgresDedicatedReadReplica, error)
	Update(context.Context, *UpdatePostgresDedicatedReadReplicaRequest) (*PostgresDedicatedReadReplica, error)
	Delete(context.Context, *DeletePostgresDedicatedReadReplicaRequest) error
}

// PostgresReadOnlyReplica is the former name for PostgresDedicatedReadReplica.
// Deprecated: use PostgresDedicatedReadReplica.
type PostgresReadOnlyReplica = PostgresDedicatedReadReplica

// ListPostgresReadOnlyReplicasRequest is the former name for ListPostgresDedicatedReadReplicasRequest.
// Deprecated: use ListPostgresDedicatedReadReplicasRequest.
type ListPostgresReadOnlyReplicasRequest = ListPostgresDedicatedReadReplicasRequest

// GetPostgresReadOnlyReplicaRequest is the former name for GetPostgresDedicatedReadReplicaRequest.
// Deprecated: use GetPostgresDedicatedReadReplicaRequest.
type GetPostgresReadOnlyReplicaRequest = GetPostgresDedicatedReadReplicaRequest

// CreatePostgresReadOnlyReplicaRequest is the former name for CreatePostgresDedicatedReadReplicaRequest.
// Deprecated: use CreatePostgresDedicatedReadReplicaRequest.
type CreatePostgresReadOnlyReplicaRequest = CreatePostgresDedicatedReadReplicaRequest

// UpdatePostgresReadOnlyReplicaRequest is the former name for UpdatePostgresDedicatedReadReplicaRequest.
// Deprecated: use UpdatePostgresDedicatedReadReplicaRequest.
type UpdatePostgresReadOnlyReplicaRequest = UpdatePostgresDedicatedReadReplicaRequest

// DeletePostgresReadOnlyReplicaRequest is the former name for DeletePostgresDedicatedReadReplicaRequest.
// Deprecated: use DeletePostgresDedicatedReadReplicaRequest.
type DeletePostgresReadOnlyReplicaRequest = DeletePostgresDedicatedReadReplicaRequest

// PostgresReadOnlyReplicasService is the former name for PostgresDedicatedReadReplicasService.
// Deprecated: use PostgresDedicatedReadReplicasService.
type PostgresReadOnlyReplicasService = PostgresDedicatedReadReplicasService

type postgresDedicatedReadReplicasService struct {
	client *Client
}

var _ PostgresDedicatedReadReplicasService = &postgresDedicatedReadReplicasService{}

func (s *postgresDedicatedReadReplicasService) List(ctx context.Context, listReq *ListPostgresDedicatedReadReplicasRequest) ([]*PostgresDedicatedReadReplica, error) {
	req, err := s.client.newRequest(http.MethodGet, postgresDedicatedReadReplicasAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for list postgres dedicated read replicas: %w", err)
	}

	replicas := []*PostgresDedicatedReadReplica{}
	if err := s.client.do(ctx, req, &replicas); err != nil {
		return nil, err
	}
	return replicas, nil
}

func (s *postgresDedicatedReadReplicasService) Get(ctx context.Context, getReq *GetPostgresDedicatedReadReplicaRequest) (*PostgresDedicatedReadReplica, error) {
	req, err := s.client.newRequest(http.MethodGet, postgresDedicatedReadReplicaAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.Replica), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get postgres dedicated read replica: %w", err)
	}

	replica := &PostgresDedicatedReadReplica{}
	if err := s.client.do(ctx, req, replica); err != nil {
		return nil, err
	}
	return replica, nil
}

func (s *postgresDedicatedReadReplicasService) Create(ctx context.Context, createReq *CreatePostgresDedicatedReadReplicaRequest) (*PostgresDedicatedReadReplica, error) {
	req, err := s.client.newRequest(http.MethodPost, postgresDedicatedReadReplicasAPIPath(createReq.Organization, createReq.Database, createReq.Branch), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for create postgres dedicated read replica: %w", err)
	}

	replica := &PostgresDedicatedReadReplica{}
	if err := s.client.do(ctx, req, replica); err != nil {
		return nil, err
	}
	return replica, nil
}

func (s *postgresDedicatedReadReplicasService) Update(ctx context.Context, updateReq *UpdatePostgresDedicatedReadReplicaRequest) (*PostgresDedicatedReadReplica, error) {
	req, err := s.client.newRequest(http.MethodPatch, postgresDedicatedReadReplicaAPIPath(updateReq.Organization, updateReq.Database, updateReq.Branch, updateReq.Replica), updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for update postgres dedicated read replica: %w", err)
	}

	replica := &PostgresDedicatedReadReplica{}
	if err := s.client.do(ctx, req, replica); err != nil {
		return nil, err
	}
	return replica, nil
}

func (s *postgresDedicatedReadReplicasService) Delete(ctx context.Context, deleteReq *DeletePostgresDedicatedReadReplicaRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, postgresDedicatedReadReplicaAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.Branch, deleteReq.Replica), nil)
	if err != nil {
		return fmt.Errorf("error creating request for delete postgres dedicated read replica: %w", err)
	}
	return s.client.do(ctx, req, nil)
}

func postgresDedicatedReadReplicasAPIPath(org, db, branch string) string {
	return path.Join(postgresBranchAPIPath(org, db, branch), "read-only-replicas")
}

func postgresDedicatedReadReplicaAPIPath(org, db, branch, replica string) string {
	return path.Join(postgresDedicatedReadReplicasAPIPath(org, db, branch), replica)
}
