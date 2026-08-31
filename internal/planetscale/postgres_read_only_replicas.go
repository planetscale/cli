package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// PostgresReadOnlyReplica represents a read-only replica for a Postgres branch.
type PostgresReadOnlyReplica struct {
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

// ListPostgresReadOnlyReplicasRequest encapsulates listing read-only replicas.
type ListPostgresReadOnlyReplicasRequest struct {
	Organization string
	Database     string
	Branch       string
}

// GetPostgresReadOnlyReplicaRequest encapsulates getting a read-only replica by name.
type GetPostgresReadOnlyReplicaRequest struct {
	Organization string
	Database     string
	Branch       string
	Replica      string
}

// CreatePostgresReadOnlyReplicaRequest encapsulates creating a read-only replica.
type CreatePostgresReadOnlyReplicaRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Name         string `json:"name"`
	Region       string `json:"region"`
	Replicas     *int   `json:"replicas,omitempty"`
	ClusterSize  string `json:"cluster_size,omitempty"`
}

// UpdatePostgresReadOnlyReplicaRequest encapsulates updating a read-only replica.
type UpdatePostgresReadOnlyReplicaRequest struct {
	Organization string                       `json:"-"`
	Database     string                       `json:"-"`
	Branch       string                       `json:"-"`
	Replica      string                       `json:"-"`
	Replicas     *int                         `json:"replicas,omitempty"`
	ClusterSize  string                       `json:"cluster_size,omitempty"`
	Parameters   map[string]map[string]string `json:"parameters,omitempty"`
}

// DeletePostgresReadOnlyReplicaRequest encapsulates deleting a read-only replica.
type DeletePostgresReadOnlyReplicaRequest struct {
	Organization string
	Database     string
	Branch       string
	Replica      string
}

// PostgresReadOnlyReplicasService is an interface for the Postgres read-only
// replicas API.
type PostgresReadOnlyReplicasService interface {
	List(context.Context, *ListPostgresReadOnlyReplicasRequest) ([]*PostgresReadOnlyReplica, error)
	Get(context.Context, *GetPostgresReadOnlyReplicaRequest) (*PostgresReadOnlyReplica, error)
	Create(context.Context, *CreatePostgresReadOnlyReplicaRequest) (*PostgresReadOnlyReplica, error)
	Update(context.Context, *UpdatePostgresReadOnlyReplicaRequest) (*PostgresReadOnlyReplica, error)
	Delete(context.Context, *DeletePostgresReadOnlyReplicaRequest) error
}

type postgresReadOnlyReplicasService struct {
	client *Client
}

var _ PostgresReadOnlyReplicasService = &postgresReadOnlyReplicasService{}

func (s *postgresReadOnlyReplicasService) List(ctx context.Context, listReq *ListPostgresReadOnlyReplicasRequest) ([]*PostgresReadOnlyReplica, error) {
	req, err := s.client.newRequest(http.MethodGet, postgresReadOnlyReplicasAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for list postgres read-only replicas: %w", err)
	}

	replicas := []*PostgresReadOnlyReplica{}
	if err := s.client.do(ctx, req, &replicas); err != nil {
		return nil, err
	}
	return replicas, nil
}

func (s *postgresReadOnlyReplicasService) Get(ctx context.Context, getReq *GetPostgresReadOnlyReplicaRequest) (*PostgresReadOnlyReplica, error) {
	req, err := s.client.newRequest(http.MethodGet, postgresReadOnlyReplicaAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.Replica), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get postgres read-only replica: %w", err)
	}

	replica := &PostgresReadOnlyReplica{}
	if err := s.client.do(ctx, req, replica); err != nil {
		return nil, err
	}
	return replica, nil
}

func (s *postgresReadOnlyReplicasService) Create(ctx context.Context, createReq *CreatePostgresReadOnlyReplicaRequest) (*PostgresReadOnlyReplica, error) {
	req, err := s.client.newRequest(http.MethodPost, postgresReadOnlyReplicasAPIPath(createReq.Organization, createReq.Database, createReq.Branch), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for create postgres read-only replica: %w", err)
	}

	replica := &PostgresReadOnlyReplica{}
	if err := s.client.do(ctx, req, replica); err != nil {
		return nil, err
	}
	return replica, nil
}

func (s *postgresReadOnlyReplicasService) Update(ctx context.Context, updateReq *UpdatePostgresReadOnlyReplicaRequest) (*PostgresReadOnlyReplica, error) {
	req, err := s.client.newRequest(http.MethodPatch, postgresReadOnlyReplicaAPIPath(updateReq.Organization, updateReq.Database, updateReq.Branch, updateReq.Replica), updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for update postgres read-only replica: %w", err)
	}

	replica := &PostgresReadOnlyReplica{}
	if err := s.client.do(ctx, req, replica); err != nil {
		return nil, err
	}
	return replica, nil
}

func (s *postgresReadOnlyReplicasService) Delete(ctx context.Context, deleteReq *DeletePostgresReadOnlyReplicaRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, postgresReadOnlyReplicaAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.Branch, deleteReq.Replica), nil)
	if err != nil {
		return fmt.Errorf("error creating request for delete postgres read-only replica: %w", err)
	}
	return s.client.do(ctx, req, nil)
}

func postgresReadOnlyReplicasAPIPath(org, db, branch string) string {
	return path.Join(postgresBranchAPIPath(org, db, branch), "read-only-replicas")
}

func postgresReadOnlyReplicaAPIPath(org, db, branch, replica string) string {
	return path.Join(postgresReadOnlyReplicasAPIPath(org, db, branch), replica)
}
