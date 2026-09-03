package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

// PostgresBranch represents a Postgres branch in the PlanetScale API.
type PostgresBranch struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	ClusterName         string    `json:"cluster_name"`
	ClusterDisplayName  string    `json:"cluster_display_name"`
	ClusterArchitecture string    `json:"cluster_architecture"`
	ClusterIOPS         int       `json:"cluster_iops"`
	State               string    `json:"state"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	Actor               Actor     `json:"actor"`
	Production          bool      `json:"production"`
	Ready               bool      `json:"ready"`
	ParentBranch        string    `json:"parent_branch"`
	Region              Region    `json:"region"`
	Kind                string    `json:"kind"`
	Replicas            int       `json:"replicas"`
}

type postgresBranchesResponse struct {
	Branches []*PostgresBranch `json:"data"`
}

// CreatePostgresBranchRequest encapsulates the request to create a Postgres branch.
type CreatePostgresBranchRequest struct {
	Organization string         `json:"-"`
	Database     string         `json:"-"`
	Region       string         `json:"region,omitempty"`
	Name         string         `json:"name"`
	ParentBranch string         `json:"parent_branch"`
	BackupID     string         `json:"backup_id,omitempty"`
	RestorePoint string         `json:"restore_point,omitempty"`
	ClusterName  string         `json:"cluster_name,omitempty"`
	MajorVersion string         `json:"major_version,omitempty"`
	Replicas     *int           `json:"replicas,omitempty"`
	Storage      *StorageConfig `json:"storage,omitempty"`
}

// ListPostgresBranchesRequest encapsulates the request to list Postgres branches for a database.
type ListPostgresBranchesRequest struct {
	Organization string
	Database     string
}

// GetPostgresBranchRequest encapsulates the request to get a specific Postgres branch.
type GetPostgresBranchRequest struct {
	Organization string
	Database     string
	Branch       string
}

// UpdatePostgresBranchRequest encapsulates the request for updating a Postgres
// branch's settings.
type UpdatePostgresBranchRequest struct {
	Organization      string `json:"-"`
	Database          string `json:"-"`
	Branch            string `json:"-"`
	NewName           string `json:"new_name,omitempty"`
	DeletionProtected *bool  `json:"deletion_protected,omitempty"`
}

// DeletePostgresBranchRequest encapsulates the request to delete a Postgres branch.
type DeletePostgresBranchRequest struct {
	Organization      string
	Database          string
	Branch            string
	DeleteDescendants bool
}

// ResizePostgresBranchRequest encapsulates the request to change a Postgres
// branch's cluster: its size, replica count, and/or configuration parameters.
// All change kinds are upserted into a single asynchronous change request via
// the branch changes API.
type ResizePostgresBranchRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`

	// ClusterSize is the fully-qualified cluster size SKU name to resize to,
	// e.g. "PS_10_GCP_X86". Use the values returned by ListClusterSKUs.
	ClusterSize string `json:"cluster_size,omitempty"`
	// Replicas is the desired number of replicas. Nil leaves it unchanged.
	Replicas *int `json:"replicas,omitempty"`
	// Parameters holds configuration parameter values nested by namespace,
	// e.g. {"pgconf": {"max_connections": "200"}}. Use the values returned by
	// ListParameters. Nil leaves parameters unchanged.
	Parameters map[string]map[string]string `json:"parameters,omitempty"`
}

// PostgresBranchClusterResizeRequest represents an asynchronous Postgres branch
// cluster change (resize) request.
type PostgresBranchClusterResizeRequest struct {
	ID    string `json:"id"`
	State string `json:"state"`

	ClusterName        string `json:"cluster_name"`
	ClusterDisplayName string `json:"cluster_display_name"`
	Replicas           int    `json:"replicas"`

	PreviousClusterName        string `json:"previous_cluster_name"`
	PreviousClusterDisplayName string `json:"previous_cluster_display_name"`
	PreviousReplicas           int    `json:"previous_replicas"`

	// Parameters and PreviousParameters are configuration parameter values
	// nested by namespace. Values are left untyped since the API returns
	// strings, numbers, and booleans depending on the parameter type.
	Parameters         map[string]map[string]any `json:"parameters"`
	PreviousParameters map[string]map[string]any `json:"previous_parameters"`

	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Terminal states for a PostgresBranchClusterResizeRequest. Non-terminal
// states are "queued", "pending", and "resizing".
const (
	PostgresBranchChangeStateCompleted = "completed"
	PostgresBranchChangeStateCanceled  = "canceled"
)

// Finished reports whether the change request reached a terminal state.
func (r *PostgresBranchClusterResizeRequest) Finished() bool {
	return r.State == PostgresBranchChangeStateCompleted || r.State == PostgresBranchChangeStateCanceled
}

// ListPostgresBranchChangesRequest encapsulates the request to list change
// requests for a Postgres branch.
type ListPostgresBranchChangesRequest struct {
	Organization string
	Database     string
	Branch       string
}

// GetPostgresBranchChangeRequest encapsulates the request to get a single
// change request for a Postgres branch.
type GetPostgresBranchChangeRequest struct {
	Organization string
	Database     string
	Branch       string
	ID           string
}

// CancelPostgresBranchChangesRequest encapsulates the request to cancel the
// queued change requests for a Postgres branch.
type CancelPostgresBranchChangesRequest struct {
	Organization string
	Database     string
	Branch       string
}

type postgresBranchChangesResponse struct {
	Changes []*PostgresBranchClusterResizeRequest `json:"data"`
}

// ListPostgresParametersRequest encapsulates the request to list the
// configuration parameters of a Postgres branch.
type ListPostgresParametersRequest struct {
	Organization string
	Database     string
	Branch       string

	// Extension filters parameters by whether they configure an extension.
	// Nil returns both.
	Extension *bool
	// Internal filters parameters by whether they are internal (immutable).
	// Nil returns both.
	Internal *bool
}

// PostgresParameter represents a configuration parameter of a Postgres branch.
type PostgresParameter struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DisplayName   string `json:"display_name"`
	Namespace     string `json:"namespace"`
	Category      string `json:"category"`
	Description   string `json:"description"`
	Extension     bool   `json:"extension"`
	Immutable     bool   `json:"immutable"`
	ParameterType string `json:"parameter_type"`
	DefaultValue  any    `json:"default_value"`
	Value         any    `json:"value"`
	Required      bool   `json:"required"`
	Restart       bool   `json:"restart"`

	// Min, Max, and Step are numbers for plain numeric parameters but strings
	// for byte- and time-typed ones (e.g. "8kB", "10s").
	Max     any      `json:"max,omitempty"`
	Min     any      `json:"min,omitempty"`
	Step    any      `json:"step,omitempty"`
	Options []string `json:"options,omitempty"`
	Units   []string `json:"units,omitempty"`
	URL     string   `json:"url"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// ListPostgresExtensionsRequest lists extensions available on a Postgres
// branch's cluster image.
type ListPostgresExtensionsRequest struct {
	Organization string
	Database     string
	Branch       string
}

// PostgresExtension is an extension defined on the branch's cluster image.
// This is the catalog of what the image can load, not CREATE EXTENSION state.
type PostgresExtension struct {
	Type              string               `json:"type"`
	Name              string               `json:"name"`
	Description       string               `json:"description"`
	Internal          bool                 `json:"internal"`
	Loader            string               `json:"loader"`
	URL               string               `json:"url"`
	Available         bool                 `json:"available"`
	UnavailableReason string               `json:"unavailable_reason"`
	Parameters        []*PostgresParameter `json:"parameters"`
}

// PostgresBranchSchemaRequest encapsulates the request to get the schema of a Postgres branch.
type PostgresBranchSchemaRequest struct {
	Organization string
	Database     string
	Branch       string
	Namespace    string `json:"-"`
}

// PostgresBranchSchema encapsulates the schema of a Postgres branch.
type PostgresBranchSchema struct {
	Name string `json:"name"`
	Raw  string `json:"raw"`
	HTML string `json:"html"`
}

// postgresBranchSchemaResponse returns the schemas
type postgresBranchSchemaResponse struct {
	Schemas []*PostgresBranchSchema `json:"data"`
}

type PostgresBranchesService interface {
	Create(context.Context, *CreatePostgresBranchRequest) (*PostgresBranch, error)
	List(context.Context, *ListPostgresBranchesRequest, ...ListOption) ([]*PostgresBranch, error)
	Get(context.Context, *GetPostgresBranchRequest) (*PostgresBranch, error)
	Update(context.Context, *UpdatePostgresBranchRequest) (*PostgresBranch, error)
	Delete(context.Context, *DeletePostgresBranchRequest) error
	Schema(context.Context, *PostgresBranchSchemaRequest) ([]*PostgresBranchSchema, error)
	ListClusterSKUs(context.Context, *ListBranchClusterSKUsRequest, ...ListOption) ([]*ClusterSKU, error)
	Resize(context.Context, *ResizePostgresBranchRequest) (*PostgresBranchClusterResizeRequest, error)
	ListChanges(context.Context, *ListPostgresBranchChangesRequest) ([]*PostgresBranchClusterResizeRequest, error)
	GetChange(context.Context, *GetPostgresBranchChangeRequest) (*PostgresBranchClusterResizeRequest, error)
	CancelChanges(context.Context, *CancelPostgresBranchChangesRequest) error
	ListParameters(context.Context, *ListPostgresParametersRequest) ([]*PostgresParameter, error)
	ListExtensions(context.Context, *ListPostgresExtensionsRequest) ([]*PostgresExtension, error)
}

type postgresBranchesService struct {
	client *Client
}

var _ PostgresBranchesService = &postgresBranchesService{}

func NewPostgresBranchesService(client *Client) *postgresBranchesService {
	return &postgresBranchesService{
		client: client,
	}
}

// Create creates a new Postgres branch in the specified organization and database.
func (p *postgresBranchesService) Create(ctx context.Context, createReq *CreatePostgresBranchRequest) (*PostgresBranch, error) {
	path := postgresBranchesAPIPath(createReq.Organization, createReq.Database)
	req, err := p.client.newRequest(http.MethodPost, path, createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	b := &PostgresBranch{}
	if err := p.client.do(ctx, req, b); err != nil {
		return nil, err
	}

	return b, nil
}

// List returns a list of Postgres branches for the specified organization and database.
func (p *postgresBranchesService) List(ctx context.Context, listReq *ListPostgresBranchesRequest, opts ...ListOption) ([]*PostgresBranch, error) {
	path := postgresBranchesAPIPath(listReq.Organization, listReq.Database)

	defaultOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		err := opt(defaultOpts)
		if err != nil {
			return nil, err
		}
	}

	req, err := p.client.newRequest(http.MethodGet, path, nil, WithQueryParams(*defaultOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	pgBranches := &postgresBranchesResponse{}
	if err := p.client.do(ctx, req, &pgBranches); err != nil {
		return nil, err
	}

	return pgBranches.Branches, nil
}

// Get returns a single Postgres branch for the specified organization, database, and branch.
func (p *postgresBranchesService) Get(ctx context.Context, getReq *GetPostgresBranchRequest) (*PostgresBranch, error) {
	path := path.Join(postgresBranchesAPIPath(getReq.Organization, getReq.Database), getReq.Branch)
	req, err := p.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	pgBranch := &PostgresBranch{}
	if err := p.client.do(ctx, req, &pgBranch); err != nil {
		return nil, err
	}

	return pgBranch, nil
}

// Delete deletes a Postgres branch from the specified organization and database.
// Update updates a Postgres branch's settings.
func (p *postgresBranchesService) Update(ctx context.Context, updateReq *UpdatePostgresBranchRequest) (*PostgresBranch, error) {
	path := postgresBranchAPIPath(updateReq.Organization, updateReq.Database, updateReq.Branch)
	req, err := p.client.newRequest(http.MethodPatch, path, updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	b := &PostgresBranch{}
	if err := p.client.do(ctx, req, b); err != nil {
		return nil, err
	}

	return b, nil
}

func (p *postgresBranchesService) Delete(ctx context.Context, deleteReq *DeletePostgresBranchRequest) error {
	path := path.Join(postgresBranchesAPIPath(deleteReq.Organization, deleteReq.Database), deleteReq.Branch)

	var opts []RequestOption
	if deleteReq.DeleteDescendants {
		v := url.Values{}
		v.Set("delete_descendants", "true")
		opts = append(opts, WithQueryParams(v))
	}

	req, err := p.client.newRequest(http.MethodDelete, path, nil, opts...)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	err = p.client.do(ctx, req, nil)
	return err
}

// ListClusterSKUs returns a list of cluster SKUs for the specified Postgres branch.
func (p *postgresBranchesService) ListClusterSKUs(ctx context.Context, listReq *ListBranchClusterSKUsRequest, opts ...ListOption) ([]*ClusterSKU, error) {
	path := path.Join(postgresBranchAPIPath(listReq.Organization, listReq.Database, listReq.Branch), "cluster-size-skus")

	defaultOpts := defaultListOptions()
	for _, opt := range opts {
		err := opt(defaultOpts)
		if err != nil {
			return nil, err
		}
	}

	req, err := p.client.newRequest(http.MethodGet, path, nil, WithQueryParams(*defaultOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	clusterSKUs := []*ClusterSKU{}
	if err := p.client.do(ctx, req, &clusterSKUs); err != nil {
		return nil, err
	}

	return clusterSKUs, nil
}

// Resize starts an asynchronous resize of a Postgres branch's cluster (and
// optionally changes its replica count). It returns the resulting change
// request, whose State can be polled via the branch changes API. A nil change
// request is returned when the requested configuration matches the current one
// (the API responds 204 No Content).
func (p *postgresBranchesService) Resize(ctx context.Context, resizeReq *ResizePostgresBranchRequest) (*PostgresBranchClusterResizeRequest, error) {
	path := path.Join(postgresBranchAPIPath(resizeReq.Organization, resizeReq.Database, resizeReq.Branch), "changes")
	req, err := p.client.newRequest(http.MethodPatch, path, resizeReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	change := &PostgresBranchClusterResizeRequest{}
	if err := p.client.do(ctx, req, change); err != nil {
		return nil, err
	}

	// A 204 No Content response (requested configuration already matches the
	// current one) leaves the body, and therefore the ID, empty. Surface that
	// as a nil change request so callers can detect the no-op.
	if change.ID == "" {
		return nil, nil
	}

	return change, nil
}

// ListChanges returns the change requests for the specified Postgres branch,
// most recent first.
func (p *postgresBranchesService) ListChanges(ctx context.Context, listReq *ListPostgresBranchChangesRequest) ([]*PostgresBranchClusterResizeRequest, error) {
	path := path.Join(postgresBranchAPIPath(listReq.Organization, listReq.Database, listReq.Branch), "changes")
	req, err := p.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	changes := &postgresBranchChangesResponse{}
	if err := p.client.do(ctx, req, changes); err != nil {
		return nil, err
	}

	return changes.Changes, nil
}

// GetChange returns a single change request for the specified Postgres branch.
func (p *postgresBranchesService) GetChange(ctx context.Context, getReq *GetPostgresBranchChangeRequest) (*PostgresBranchClusterResizeRequest, error) {
	path := path.Join(postgresBranchAPIPath(getReq.Organization, getReq.Database, getReq.Branch), "changes", getReq.ID)
	req, err := p.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	change := &PostgresBranchClusterResizeRequest{}
	if err := p.client.do(ctx, req, change); err != nil {
		return nil, err
	}

	return change, nil
}

// CancelChanges cancels the queued change requests for the specified Postgres
// branch.
func (p *postgresBranchesService) CancelChanges(ctx context.Context, cancelReq *CancelPostgresBranchChangesRequest) error {
	path := path.Join(postgresBranchAPIPath(cancelReq.Organization, cancelReq.Database, cancelReq.Branch), "changes")
	req, err := p.client.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	return p.client.do(ctx, req, nil)
}

// ListParameters returns the configuration parameters for the specified
// Postgres branch. Pending values from a queued change request are reflected
// in the returned parameters.
func (p *postgresBranchesService) ListParameters(ctx context.Context, listReq *ListPostgresParametersRequest) ([]*PostgresParameter, error) {
	path := path.Join(postgresBranchAPIPath(listReq.Organization, listReq.Database, listReq.Branch), "parameters")

	v := url.Values{}
	if listReq.Extension != nil {
		v.Set("extension", fmt.Sprintf("%t", *listReq.Extension))
	}
	if listReq.Internal != nil {
		v.Set("internal", fmt.Sprintf("%t", *listReq.Internal))
	}

	req, err := p.client.newRequest(http.MethodGet, path, nil, WithQueryParams(v))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	// Unlike most list endpoints, the parameters endpoint returns a bare
	// JSON array rather than a {"data": [...]} envelope.
	parameters := []*PostgresParameter{}
	if err := p.client.do(ctx, req, &parameters); err != nil {
		return nil, err
	}

	return parameters, nil
}

// ListExtensions returns extensions available on the Postgres branch's cluster
// image. The API returns a bare JSON array.
func (p *postgresBranchesService) ListExtensions(ctx context.Context, listReq *ListPostgresExtensionsRequest) ([]*PostgresExtension, error) {
	path := path.Join(postgresBranchAPIPath(listReq.Organization, listReq.Database, listReq.Branch), "extensions")
	req, err := p.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	extensions := []*PostgresExtension{}
	if err := p.client.do(ctx, req, &extensions); err != nil {
		return nil, err
	}

	return extensions, nil
}

// Schema returns the schema for the specified Postgres branch.
func (p *postgresBranchesService) Schema(ctx context.Context, schemaReq *PostgresBranchSchemaRequest) ([]*PostgresBranchSchema, error) {
	path := path.Join(postgresBranchAPIPath(schemaReq.Organization, schemaReq.Database, schemaReq.Branch), "schema")
	v := url.Values{}
	if schemaReq.Namespace != "" {
		v.Set("namespace", schemaReq.Namespace)
	}

	req, err := p.client.newRequest(http.MethodGet, path, nil, WithQueryParams(v))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	schemas := &postgresBranchSchemaResponse{}
	if err := p.client.do(ctx, req, &schemas); err != nil {
		return nil, err
	}

	return schemas.Schemas, nil
}

func postgresBranchesAPIPath(org, db string) string {
	return path.Join(databasesAPIPath(org), db, "branches")
}

func postgresBranchAPIPath(org, db, branch string) string {
	return path.Join(postgresBranchesAPIPath(org, db), branch)
}
