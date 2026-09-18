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
	External                         bool                              `json:"external"`
	CreatedAt                        time.Time                         `json:"created_at"`
	UpdatedAt                        time.Time                         `json:"updated_at"`
	VReplicationFlags                *VReplicationFlags                `json:"vreplication_flags"`
	ReplicationDurabilityConstraints *ReplicationDurabilityConstraints `json:"replication_durability_constraints"`
	MaxRollout                       *int                              `json:"max_rollout"`
	Throttler                        *KeyspaceThrottler                `json:"throttler"`
	ReadOnlyRegions                  []*ReadOnlyRegionKeyspace         `json:"read_only_regions"`
}

type ReadOnlyRegionKeyspace struct {
	Region             string `json:"region"`
	ClusterName        string `json:"cluster_name"`
	ClusterDisplayName string `json:"cluster_display_name"`
	Replicas           int    `json:"replicas"`
}

type ReadOnlyRegionKeyspaceConfig struct {
	Region      string  `json:"region"`
	ClusterSize *string `json:"cluster_size,omitempty"`
	Replicas    *int    `json:"replicas,omitempty"`
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

type ExternalDatasource struct {
	DatabaseName  string `json:"database_name,omitempty"`
	Hostname      string `json:"hostname,omitempty"`
	Port          int    `json:"port,omitempty"`
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	SSLMode       string `json:"ssl_mode,omitempty"`
	SSLCA         string `json:"ssl_ca,omitempty"`
	SSLCert       string `json:"ssl_cert,omitempty"`
	SSLKey        string `json:"ssl_key,omitempty"`
	SSLServerName string `json:"ssl_server_name,omitempty"`
	MinTLSVersion string `json:"min_tls_version,omitempty"`
	TabletCell    string `json:"tablet_cell,omitempty"`
}

type CreateExternalKeyspaceRequest struct {
	Organization       string             `json:"-"`
	Database           string             `json:"-"`
	Branch             string             `json:"-"`
	Name               string             `json:"name"`
	ClusterSize        string             `json:"cluster_size,omitempty"`
	SkipLintErrors     bool               `json:"skip_lint_errors,omitempty"`
	ExternalDatasource ExternalDatasource `json:"external_datasource"`
}

type LintExternalKeyspaceRequest struct {
	Organization       string             `json:"-"`
	Database           string             `json:"-"`
	Branch             string             `json:"-"`
	ExternalDatasource ExternalDatasource `json:"external_datasource"`
}

type ExternalKeyspaceLintError struct {
	LintError        string `json:"lint_error"`
	TableName        string `json:"table_name"`
	ErrorDescription string `json:"error_description"`
}

type LintExternalKeyspaceResponse struct {
	CanConnect                    bool                         `json:"can_connect"`
	AllowSkipFailedTestConnection bool                         `json:"allow_skip_failed_test_connection"`
	Error                         string                       `json:"error,omitempty"`
	HasForeignKeys                bool                         `json:"has_foreign_keys"`
	LintErrors                    []*ExternalKeyspaceLintError `json:"lint_errors"`
	MaxPoolSize                   int                          `json:"max_pool_size"`
	ServerVersion                 string                       `json:"server_version"`
	TotalStorageBytes             int64                        `json:"total_storage_bytes"`
	DefaultKeyspaceStorageBytes   int64                        `json:"default_keyspace_storage_bytes"`
}

type GetKeyspaceRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Keyspace     string `json:"-"`
	Full         bool   `json:"-"`
}

type DeleteKeyspaceRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Keyspace     string `json:"-"`
}

type UpdateReadOnlyRegionsRequest struct {
	Organization    string                          `json:"-"`
	Database        string                          `json:"-"`
	Branch          string                          `json:"-"`
	Keyspace        string                          `json:"-"`
	ReadOnlyRegions []*ReadOnlyRegionKeyspaceConfig `json:"read_only_regions"`
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
	Throttler                        *KeyspaceThrottler                `json:"throttler,omitempty"`
	MaxRollout                       *int                              `json:"max_rollout,omitempty"`
}

type ReplicationDurabilityConstraints struct {
	Strategy string `json:"strategy"`
}

type VReplicationFlags struct {
	OptimizeInserts           bool `json:"optimize_inserts"`
	AllowNoBlobBinlogRowImage bool `json:"allow_no_blob_binlog_row_image"`
	VPlayerBatching           bool `json:"vplayer_batching"`
}

type KeyspaceThrottler struct {
	Enabled   *bool    `json:"enabled,omitempty"`
	Threshold *float64 `json:"threshold,omitempty"`
}

// KeyspacesService is an interface for interacting with the keyspace endpoints of the PlanetScale API
type KeyspacesService interface {
	Create(context.Context, *CreateKeyspaceRequest) (*Keyspace, error)
	CreateExternal(context.Context, *CreateExternalKeyspaceRequest) (*Keyspace, error)
	LintExternal(context.Context, *LintExternalKeyspaceRequest) (*LintExternalKeyspaceResponse, error)
	List(context.Context, *ListKeyspacesRequest) ([]*Keyspace, error)
	Get(context.Context, *GetKeyspaceRequest) (*Keyspace, error)
	Delete(context.Context, *DeleteKeyspaceRequest) error
	UpdateReadOnlyRegions(context.Context, *UpdateReadOnlyRegionsRequest) ([]*ReadOnlyRegionKeyspace, error)
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

// UpdateReadOnlyRegions configures a keyspace's read-only regions.
func (s *keyspacesService) UpdateReadOnlyRegions(ctx context.Context, updateReq *UpdateReadOnlyRegionsRequest) ([]*ReadOnlyRegionKeyspace, error) {
	pathStr := path.Join(keyspaceAPIPath(updateReq.Organization, updateReq.Database, updateReq.Branch, updateReq.Keyspace), "read-only-regions")
	req, err := s.client.newRequest(http.MethodPut, pathStr, updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	regions := []*ReadOnlyRegionKeyspace{}
	if err := s.client.do(ctx, req, &regions); err != nil {
		return nil, err
	}

	return regions, nil
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

func (s *keyspacesService) CreateExternal(ctx context.Context, createReq *CreateExternalKeyspaceRequest) (*Keyspace, error) {
	req, err := s.client.newRequest(http.MethodPost, keyspacesExternalAPIPath(createReq.Organization, createReq.Database, createReq.Branch), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	keyspace := &Keyspace{}
	if err := s.client.do(ctx, req, keyspace); err != nil {
		return nil, err
	}

	return keyspace, nil
}

func (s *keyspacesService) LintExternal(ctx context.Context, lintReq *LintExternalKeyspaceRequest) (*LintExternalKeyspaceResponse, error) {
	req, err := s.client.newRequest(http.MethodPost, keyspacesExternalLintAPIPath(lintReq.Organization, lintReq.Database, lintReq.Branch), lintReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp := &LintExternalKeyspaceResponse{}
	if err := s.client.do(ctx, req, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// Delete deletes a keyspace from a branch.
func (s *keyspacesService) Delete(ctx context.Context, deleteReq *DeleteKeyspaceRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, keyspaceAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.Branch, deleteReq.Keyspace), nil)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	return s.client.do(ctx, req, nil)
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

func keyspacesExternalAPIPath(org, db, branch string) string {
	return path.Join(keyspacesAPIPath(org, db, branch), "external")
}

func keyspacesExternalLintAPIPath(org, db, branch string) string {
	return path.Join(keyspacesExternalAPIPath(org, db, branch), "lint")
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
