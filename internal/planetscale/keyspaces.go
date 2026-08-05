package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

type Keyspace struct {
	ID                               string                            `json:"id"`
	Name                             string                            `json:"name"`
	Shards                           int                               `json:"shards"`
	Sharded                          bool                              `json:"sharded"`
	Replicas                         uint64                            `json:"replicas"`
	ExtraReplicas                    uint64                            `json:"extra_replicas"`
	ResizePending                    bool                              `json:"resize_pending"`
	Resizing                         bool                              `json:"resizing"`
	Ready                            bool                              `json:"ready"`
	ClusterSize                      string                            `json:"cluster_name"`
	CreatedAt                        time.Time                         `json:"created_at"`
	UpdatedAt                        time.Time                         `json:"updated_at"`
	VReplicationFlags                *VReplicationFlags                `json:"vreplication_flags"`
	ReplicationDurabilityConstraints *ReplicationDurabilityConstraints `json:"replication_durability_constraints"`
	ReadOnlyRegions                  []*ReadOnlyRegionKeyspace         `json:"read_only_regions"`
}

type ReadOnlyRegionKeyspace struct {
	Region             string `json:"region"`
	ClusterName        string `json:"cluster_name"`
	ClusterDisplayName string `json:"cluster_display_name"`
	Replicas           int    `json:"replicas"`
}

// VSchema represnts the VSchema for a branch keyspace
type VSchema struct {
	Raw  string `json:"raw"`
	HTML string `json:"html"`
}

type ListKeyspacesRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

type CreateKeyspaceRequest struct {
	Organization  string `json:"-"`
	Database      string `json:"-"`
	Branch        string `json:"-"`
	Name          string `json:"name"`
	ClusterSize   string `json:"cluster_size"`
	ExtraReplicas int    `json:"extra_replicas"`
	Shards        int    `json:"shards"`
}

type GetKeyspaceRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Keyspace     string `json:"-"`
	Full         bool   `json:"-"`
}

type GetKeyspaceVSchemaRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Keyspace     string `json:"-"`
}

type UpdateKeyspaceVSchemaRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Keyspace     string `json:"-"`
	VSchema      string `json:"vschema"`
}

type keyspacesResponse struct {
	Keyspaces []*Keyspace `json:"data"`
}

type ResizeKeyspaceRequest struct {
	Organization  string  `json:"-"`
	Database      string  `json:"-"`
	Branch        string  `json:"-"`
	Keyspace      string  `json:"-"`
	ExtraReplicas *uint   `json:"extra_replicas,omitempty"`
	ClusterSize   *string `json:"cluster_size,omitempty"`
}

type KeyspaceResizeRequest struct {
	ID    string `json:"id"`
	State string `json:"state"`
	Actor *Actor `json:"actor"`

	ClusterSize         string `json:"cluster_name"`
	PreviousClusterSize string `json:"previous_cluster_name"`

	Replicas         uint `json:"replicas"`
	ExtraReplicas    uint `json:"extra_replicas"`
	PreviousReplicas uint `json:"previous_replicas"`

	UpdatedAt   time.Time  `json:"updated_at"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

type KeyspaceRollout struct {
	Name  string `json:"name"`
	State string `json:"state"`

	Shards []ShardRollout `json:"shards"`
}

type ShardRollout struct {
	Name  string `json:"name"`
	State string `json:"state"`

	LastRolloutStartedAt  time.Time `json:"last_rollout_started_at"`
	LastRolloutFinishedAt time.Time `json:"last_rollout_finished_at"`
}

type CancelKeyspaceResizeRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Keyspace     string `json:"-"`
}

type KeyspaceResizeStatusRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Keyspace     string `json:"-"`
}

type KeyspaceRolloutStatusRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Keyspace     string `json:"-"`
}

type UpdateKeyspaceSettingsRequest struct {
	Organization                     string                            `json:"-"`
	Database                         string                            `json:"-"`
	Branch                           string                            `json:"-"`
	Keyspace                         string                            `json:"-"`
	ReplicationDurabilityConstraints *ReplicationDurabilityConstraints `json:"replication_durability_constraints,omitempty"`
	VReplicationFlags                *VReplicationFlags                `json:"vreplication_flags,omitempty"`
}

type ReplicationDurabilityConstraints struct {
	Strategy string `json:"strategy"`
}

type VReplicationFlags struct {
	OptimizeInserts           bool `json:"optimize_inserts"`
	AllowNoBlobBinlogRowImage bool `json:"allow_no_blob_binlog_row_image"`
	VPlayerBatching           bool `json:"vplayer_batching"`
}

// KeyspacesService is an interface for interacting with the keyspace endpoints of the PlanetScale API
type KeyspacesService interface {
	Create(context.Context, *CreateKeyspaceRequest) (*Keyspace, error)
	List(context.Context, *ListKeyspacesRequest) ([]*Keyspace, error)
	Get(context.Context, *GetKeyspaceRequest) (*Keyspace, error)
	VSchema(context.Context, *GetKeyspaceVSchemaRequest) (*VSchema, error)
	UpdateVSchema(context.Context, *UpdateKeyspaceVSchemaRequest) (*VSchema, error)
	Resize(context.Context, *ResizeKeyspaceRequest) (*KeyspaceResizeRequest, error)
	CancelResize(context.Context, *CancelKeyspaceResizeRequest) error
	ResizeStatus(context.Context, *KeyspaceResizeStatusRequest) (*KeyspaceResizeRequest, error)
	RolloutStatus(context.Context, *KeyspaceRolloutStatusRequest) (*KeyspaceRollout, error)
	UpdateSettings(context.Context, *UpdateKeyspaceSettingsRequest) (*Keyspace, error)
}

type keyspacesService struct {
	client *Client
}

var _ KeyspacesService = &keyspacesService{}

func NewKeyspacesService(client *Client) *keyspacesService {
	return &keyspacesService{client}
}

// List returns a list of keyspaces for a branch
func (s *keyspacesService) List(ctx context.Context, listReq *ListKeyspacesRequest) ([]*Keyspace, error) {
	req, err := s.client.newRequest(http.MethodGet, keyspacesAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	keyspaces := &keyspacesResponse{}
	if err := s.client.do(ctx, req, keyspaces); err != nil {
		return nil, err
	}

	return keyspaces.Keyspaces, nil
}

// Get returns a keyspace for a branch
func (s *keyspacesService) Get(ctx context.Context, getReq *GetKeyspaceRequest) (*Keyspace, error) {
	query := url.Values{}
	if getReq.Full {
		query.Set("full", "true")
	}

	req, err := s.client.newRequest(http.MethodGet, keyspaceAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.Keyspace), nil, WithQueryParams(query))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	keyspace := &Keyspace{}
	if err := s.client.do(ctx, req, keyspace); err != nil {
		return nil, err
	}

	return keyspace, nil
}

// Create creates a keyspace for a branch
func (s *keyspacesService) Create(ctx context.Context, createReq *CreateKeyspaceRequest) (*Keyspace, error) {
	req, err := s.client.newRequest(http.MethodPost, keyspacesAPIPath(createReq.Organization, createReq.Database, createReq.Branch), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	keyspace := &Keyspace{}
	if err := s.client.do(ctx, req, keyspace); err != nil {
		return nil, err
	}

	return keyspace, nil
}

// VSchema returns the VSchema for a keyspace in a branch
func (s *keyspacesService) VSchema(ctx context.Context, getReq *GetKeyspaceVSchemaRequest) (*VSchema, error) {
	pathStr := path.Join(keyspaceAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.Keyspace), "vschema")
	req, err := s.client.newRequest(http.MethodGet, pathStr, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	vschema := &VSchema{}
	if err := s.client.do(ctx, req, vschema); err != nil {
		return nil, err
	}

	return vschema, nil
}

func (s *keyspacesService) UpdateVSchema(ctx context.Context, updateReq *UpdateKeyspaceVSchemaRequest) (*VSchema, error) {
	pathStr := path.Join(keyspaceAPIPath(updateReq.Organization, updateReq.Database, updateReq.Branch, updateReq.Keyspace), "vschema")
	req, err := s.client.newRequest(http.MethodPatch, pathStr, updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	vschema := &VSchema{}
	if err := s.client.do(ctx, req, vschema); err != nil {
		return nil, err
	}

	return vschema, nil
}

// Resize starts or queues a resize of a branch's keyspace.
func (s *keyspacesService) Resize(ctx context.Context, resizeReq *ResizeKeyspaceRequest) (*KeyspaceResizeRequest, error) {
	req, err := s.client.newRequest(http.MethodPut, keyspaceResizesAPIPath(resizeReq.Organization, resizeReq.Database, resizeReq.Branch, resizeReq.Keyspace), resizeReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	keyspaceResize := &KeyspaceResizeRequest{}
	if err := s.client.do(ctx, req, keyspaceResize); err != nil {
		return nil, err
	}

	return keyspaceResize, nil
}

// CancelResize cancels a queued resize of a branch's keyspace.
func (s *keyspacesService) CancelResize(ctx context.Context, cancelReq *CancelKeyspaceResizeRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, keyspaceResizesAPIPath(cancelReq.Organization, cancelReq.Database, cancelReq.Branch, cancelReq.Keyspace), nil)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	return s.client.do(ctx, req, nil)
}

func keyspacesAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "keyspaces")
}

func keyspaceAPIPath(org, db, branch, keyspace string) string {
	return path.Join(keyspacesAPIPath(org, db, branch), keyspace)
}

func keyspaceResizesAPIPath(org, db, branch, keyspace string) string {
	return path.Join(keyspaceAPIPath(org, db, branch, keyspace), "resizes")
}

type keyspaceResizesResponse struct {
	Resizes []*KeyspaceResizeRequest `json:"data"`
}

func (s *keyspacesService) ResizeStatus(ctx context.Context, resizeReq *KeyspaceResizeStatusRequest) (*KeyspaceResizeRequest, error) {
	req, err := s.client.newRequest(http.MethodGet, keyspaceResizesAPIPath(resizeReq.Organization, resizeReq.Database, resizeReq.Branch, resizeReq.Keyspace), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resizesResponse := &keyspaceResizesResponse{}
	if err := s.client.do(ctx, req, resizesResponse); err != nil {
		return nil, err
	}

	// If there are no resizes, treat the same as a not found error
	if len(resizesResponse.Resizes) == 0 {
		return nil, &Error{
			msg:  "Not Found",
			Code: ErrNotFound,
		}
	}

	return resizesResponse.Resizes[0], nil
}

func keyspaceRolloutStatusAPIPath(org, db, branch, keyspace string) string {
	return path.Join(keyspaceAPIPath(org, db, branch, keyspace), "rollout-status")
}

func (s *keyspacesService) RolloutStatus(ctx context.Context, rolloutReq *KeyspaceRolloutStatusRequest) (*KeyspaceRollout, error) {
	req, err := s.client.newRequest(http.MethodGet, keyspaceRolloutStatusAPIPath(rolloutReq.Organization, rolloutReq.Database, rolloutReq.Branch, rolloutReq.Keyspace), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	rolloutStatusResponse := &KeyspaceRollout{}
	if err := s.client.do(ctx, req, rolloutStatusResponse); err != nil {
		return nil, err
	}

	return rolloutStatusResponse, nil
}

func (s *keyspacesService) UpdateSettings(ctx context.Context, updateReq *UpdateKeyspaceSettingsRequest) (*Keyspace, error) {
	req, err := s.client.newRequest(http.MethodPatch, keyspaceAPIPath(updateReq.Organization, updateReq.Database, updateReq.Branch, updateReq.Keyspace), updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	keyspace := &Keyspace{}
	if err := s.client.do(ctx, req, keyspace); err != nil {
		return nil, err
	}

	return keyspace, nil
}
