package planetscale

import (
	"context"
	"net/http"
	"path"
	"time"
)

var _ ReadOnlyReplicasService = &readOnlyReplicasService{}

// ReadOnlyReplica represents a read-only replica of a database branch.
type ReadOnlyReplica struct {
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
	Actor                        *Actor               `json:"actor"`
	Region                       Region               `json:"region"`
	Parameters                   []*PostgresParameter `json:"parameters"`
}

// ReadOnlyReplicaChangeRequest represents a queued or completed change to a read-only replica.
type ReadOnlyReplicaChangeRequest struct {
	ID                         string                    `json:"id"`
	State                      string                    `json:"state"`
	ClusterName                string                    `json:"cluster_name"`
	ClusterDisplayName         string                    `json:"cluster_display_name"`
	ClusterRank                int                       `json:"cluster_rank"`
	Replicas                   int                       `json:"replicas"`
	Parameters                 map[string]map[string]any `json:"parameters"`
	PreviousClusterName        string                    `json:"previous_cluster_name"`
	PreviousClusterDisplayName string                    `json:"previous_cluster_display_name"`
	PreviousClusterRank        int                       `json:"previous_cluster_rank"`
	PreviousReplicas           int                       `json:"previous_replicas"`
	PreviousParameters         map[string]map[string]any `json:"previous_parameters"`
	StartedAt                  *time.Time                `json:"started_at"`
	CompletedAt                *time.Time                `json:"completed_at"`
	CreatedAt                  time.Time                 `json:"created_at"`
	UpdatedAt                  time.Time                 `json:"updated_at"`
	Actor                      *Actor                    `json:"actor"`
	Replica                    *ReadOnlyReplicaRef       `json:"replica"`
}

// ReadOnlyReplicaRef identifies the read-only replica a change request targets.
type ReadOnlyReplicaRef struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

// ReadOnlyReplicasService is an interface for communicating with the read-only
// replicas API.
type ReadOnlyReplicasService interface {
	List(context.Context, *ListReadOnlyReplicasRequest, ...ListOption) ([]*ReadOnlyReplica, error)
	Create(context.Context, *CreateReadOnlyReplicaRequest) (*ReadOnlyReplica, error)
	Get(context.Context, *GetReadOnlyReplicaRequest) (*ReadOnlyReplica, error)
	Delete(context.Context, *DeleteReadOnlyReplicaRequest) error
	ListChanges(context.Context, *ListReadOnlyReplicaChangesRequest, ...ListOption) ([]*ReadOnlyReplicaChangeRequest, error)
}

type ListReadOnlyReplicasRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

type CreateReadOnlyReplicaRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Name         string `json:"name"`
	Region       string `json:"region"`
	Replicas     *int   `json:"replicas,omitempty"`
	ClusterSize  string `json:"cluster_size,omitempty"`
}

type GetReadOnlyReplicaRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Name         string `json:"-"`
}

type DeleteReadOnlyReplicaRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Name         string `json:"-"`
}

type ListReadOnlyReplicaChangesRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Period       string `json:"-"`
}

type readOnlyReplicaChangesResponse struct {
	Data []*ReadOnlyReplicaChangeRequest `json:"data"`
}

type readOnlyReplicasService struct {
	client *Client
}

func (s *readOnlyReplicasService) List(ctx context.Context, listReq *ListReadOnlyReplicasRequest, opts ...ListOption) ([]*ReadOnlyReplica, error) {
	listOpts := defaultListOptions(opts...)
	req, err := s.client.newRequest(http.MethodGet, readOnlyReplicasAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	replicas := []*ReadOnlyReplica{}
	if err := s.client.do(ctx, req, &replicas); err != nil {
		return nil, err
	}
	return replicas, nil
}

func (s *readOnlyReplicasService) Create(ctx context.Context, createReq *CreateReadOnlyReplicaRequest) (*ReadOnlyReplica, error) {
	req, err := s.client.newRequest(http.MethodPost, readOnlyReplicasAPIPath(createReq.Organization, createReq.Database, createReq.Branch), createReq)
	if err != nil {
		return nil, err
	}

	replica := &ReadOnlyReplica{}
	if err := s.client.do(ctx, req, &replica); err != nil {
		return nil, err
	}
	return replica, nil
}

func (s *readOnlyReplicasService) Get(ctx context.Context, getReq *GetReadOnlyReplicaRequest) (*ReadOnlyReplica, error) {
	req, err := s.client.newRequest(http.MethodGet, readOnlyReplicaAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.Name), nil)
	if err != nil {
		return nil, err
	}

	replica := &ReadOnlyReplica{}
	if err := s.client.do(ctx, req, &replica); err != nil {
		return nil, err
	}
	return replica, nil
}

func (s *readOnlyReplicasService) Delete(ctx context.Context, deleteReq *DeleteReadOnlyReplicaRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, readOnlyReplicaAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.Branch, deleteReq.Name), nil)
	if err != nil {
		return err
	}
	return s.client.do(ctx, req, nil)
}

func (s *readOnlyReplicasService) ListChanges(ctx context.Context, listReq *ListReadOnlyReplicaChangesRequest, opts ...ListOption) ([]*ReadOnlyReplicaChangeRequest, error) {
	listOpts := defaultListOptions(opts...)
	if listReq.Period != "" {
		listOpts.URLValues.Set("period", listReq.Period)
	}

	req, err := s.client.newRequest(http.MethodGet, readOnlyReplicaChangesAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	resp := &readOnlyReplicaChangesResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func readOnlyReplicasAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "read-only-replicas")
}

func readOnlyReplicaAPIPath(org, db, branch, name string) string {
	return path.Join(readOnlyReplicasAPIPath(org, db, branch), name)
}

func readOnlyReplicaChangesAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "read-only-replica-changes")
}
