package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

// Actor represents a user or service token
type Actor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"display_name"`
}

// DatabaseBranch represents a database branch.
type DatabaseBranch struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	ParentBranch   string    `json:"parent_branch"`
	Actor          Actor     `json:"actor"`
	Region         Region    `json:"region"`
	Ready          bool      `json:"ready"`
	Production     bool      `json:"production"`
	HtmlURL        string    `json:"html_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	SafeMigrations bool      `json:"safe_migrations"`
	VTGateSize     string    `json:"vtgate_size"`
	VTGateCount    int       `json:"vtgate_count"`
}

type databaseBranchesResponse struct {
	Branches []*DatabaseBranch `json:"data"`
}

// CreateDatabaseBranchRequest encapsulates the request for creating a new
// database branch
type CreateDatabaseBranchRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Region       string `json:"region,omitempty"`
	Name         string `json:"name"`
	ParentBranch string `json:"parent_branch"`
	BackupID     string `json:"backup_id,omitempty"`
	SeedData     string `json:"seed_data,omitempty"`
	ClusterSize  string `json:"cluster_size,omitempty"`
}

// UpdateDatabaseBranchRequest encapsulates the request for updating a database
// branch's settings.
type UpdateDatabaseBranchRequest struct {
	Organization      string `json:"-"`
	Database          string `json:"-"`
	Branch            string `json:"-"`
	NewName           string `json:"new_name,omitempty"`
	DeletionProtected *bool  `json:"deletion_protected,omitempty"`
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
	Organization      string
	Database          string
	Branch            string
	DeleteDescendants bool
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
	Keyspace     string `json:"-"`
}

type BranchRoutingRulesRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

type UpdateBranchRoutingRulesRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	RoutingRules string `json:"routing_rules"`
}

// RefreshSchemaRequest reflects the request needed to refresh a schema
// snapshot on a database branch.
type RefreshSchemaRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

// DemoteRequest encapsulates the request for demoting a branch to
// development.
type DemoteRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

// PromoteRequest encapsulates the request for promoting a request to
// production.
type PromoteRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

// EnableSafeMigrationsRequest encapsulates the request for enabling safe
// migrations on a branch.
type EnableSafeMigrationsRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

// DisableSafeMigrationsRequest encapsulates the request for disabling safe
// migrations on a branch.
type DisableSafeMigrationsRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

// LintSchemaRequest encapsulates the request for linting a branch's schema.
type LintSchemaRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

// SchemaLintError represents an error with the branch's schema
type SchemaLintError struct {
	LintError        string `json:"lint_error"`
	Keyspace         string `json:"keyspace_name"`
	Table            string `json:"table_name"`
	SubjectType      string `json:"subject_type"`
	ErrorDescription string `json:"error_description"`
	DocsURL          string `json:"docs_url"`
}

type RoutingRules struct {
	Raw  string `json:"raw"`
	HTML string `json:"html"`
}

// DatabaseBranchesService is an interface for communicating with the PlanetScale
// Database Branch API endpoint.
type DatabaseBranchesService interface {
	Create(context.Context, *CreateDatabaseBranchRequest) (*DatabaseBranch, error)
	List(context.Context, *ListDatabaseBranchesRequest, ...ListOption) ([]*DatabaseBranch, error)
	Get(context.Context, *GetDatabaseBranchRequest) (*DatabaseBranch, error)
	Update(context.Context, *UpdateDatabaseBranchRequest) (*DatabaseBranch, error)
	Delete(context.Context, *DeleteDatabaseBranchRequest) error
	Diff(context.Context, *DiffBranchRequest) ([]*Diff, error)
	Schema(context.Context, *BranchSchemaRequest) ([]*Diff, error)
	RoutingRules(context.Context, *BranchRoutingRulesRequest) (*RoutingRules, error)
	UpdateRoutingRules(context.Context, *UpdateBranchRoutingRulesRequest) (*RoutingRules, error)
	RefreshSchema(context.Context, *RefreshSchemaRequest) error
	Demote(context.Context, *DemoteRequest) (*DatabaseBranch, error)
	Promote(context.Context, *PromoteRequest) (*DatabaseBranch, error)
	EnableSafeMigrations(context.Context, *EnableSafeMigrationsRequest) (*DatabaseBranch, error)
	DisableSafeMigrations(context.Context, *DisableSafeMigrationsRequest) (*DatabaseBranch, error)
	LintSchema(context.Context, *LintSchemaRequest) ([]*SchemaLintError, error)
	ListClusterSKUs(context.Context, *ListBranchClusterSKUsRequest, ...ListOption) ([]*ClusterSKU, error)
	Resize(context.Context, *ResizeBranchRequest) (*BranchResizeRequest, error)
	ListResizes(context.Context, *ListBranchResizesRequest) ([]*BranchResizeRequest, error)
	CancelResize(context.Context, *CancelBranchResizeRequest) error
	ResizeStatus(context.Context, *BranchResizeStatusRequest) (*BranchResizeRequest, error)
}

// ListBranchClusterSKUsRequest encapsulates the request for getting a list of Cluster SKUs for a branch.
type ListBranchClusterSKUsRequest struct {
	Organization string
	Database     string
	Branch       string
}

type databaseBranchesService struct {
	client *Client
}

var _ DatabaseBranchesService = &databaseBranchesService{}

func NewDatabaseBranchesService(client *Client) *databaseBranchesService {
	return &databaseBranchesService{
		client: client,
	}
}

func (d *databaseBranchesService) Diff(ctx context.Context, diffReq *DiffBranchRequest) ([]*Diff, error) {
	path := path.Join(databaseBranchAPIPath(diffReq.Organization, diffReq.Database, diffReq.Branch), "diff")
	req, err := d.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
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
	path := path.Join(databaseBranchAPIPath(schemaReq.Organization, schemaReq.Database, schemaReq.Branch), "schema")
	v := url.Values{}
	if schemaReq.Keyspace != "" {
		v.Add("keyspace", schemaReq.Keyspace)
	}

	req, err := d.client.newRequest(http.MethodGet, path, nil, WithQueryParams(v))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	schemas := &schemaResponse{}
	if err := d.client.do(ctx, req, &schemas); err != nil {
		return nil, err
	}

	return schemas.Schemas, nil
}

func (d *databaseBranchesService) RoutingRules(ctx context.Context, routingRulesReq *BranchRoutingRulesRequest) (*RoutingRules, error) {
	path := path.Join(databaseBranchAPIPath(routingRulesReq.Organization, routingRulesReq.Database, routingRulesReq.Branch), "routing-rules")

	req, err := d.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	routingRules := &RoutingRules{}
	if err := d.client.do(ctx, req, &routingRules); err != nil {
		return nil, err
	}

	return routingRules, nil
}

func (d *databaseBranchesService) UpdateRoutingRules(ctx context.Context, updateRoutingRulesReq *UpdateBranchRoutingRulesRequest) (*RoutingRules, error) {
	path := path.Join(databaseBranchAPIPath(updateRoutingRulesReq.Organization, updateRoutingRulesReq.Database, updateRoutingRulesReq.Branch), "routing-rules")

	req, err := d.client.newRequest(http.MethodPatch, path, updateRoutingRulesReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	routingRules := &RoutingRules{}
	if err := d.client.do(ctx, req, &routingRules); err != nil {
		return nil, err
	}

	return routingRules, nil
}

// Create creates a new branch for an organization's database.
func (d *databaseBranchesService) Create(ctx context.Context, createReq *CreateDatabaseBranchRequest) (*DatabaseBranch, error) {
	path := databaseBranchesAPIPath(createReq.Organization, createReq.Database)

	req, err := d.client.newRequest(http.MethodPost, path, createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for branch database: %w", err)
	}

	dbBranch := &DatabaseBranch{}
	if err := d.client.do(ctx, req, &dbBranch); err != nil {
		return nil, err
	}

	return dbBranch, nil
}

// Get returns a database branch for an organization's database.
func (d *databaseBranchesService) Get(ctx context.Context, getReq *GetDatabaseBranchRequest) (*DatabaseBranch, error) {
	path := path.Join(databaseBranchesAPIPath(getReq.Organization, getReq.Database), getReq.Branch)
	req, err := d.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	dbBranch := &DatabaseBranch{}
	if err := d.client.do(ctx, req, &dbBranch); err != nil {
		return nil, err
	}

	return dbBranch, nil
}

// Update updates a database branch's settings.
func (d *databaseBranchesService) Update(ctx context.Context, updateReq *UpdateDatabaseBranchRequest) (*DatabaseBranch, error) {
	path := databaseBranchAPIPath(updateReq.Organization, updateReq.Database, updateReq.Branch)
	req, err := d.client.newRequest(http.MethodPatch, path, updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for update branch: %w", err)
	}

	dbBranch := &DatabaseBranch{}
	if err := d.client.do(ctx, req, &dbBranch); err != nil {
		return nil, err
	}

	return dbBranch, nil
}

// List returns all of the branches for an organization's
// database.
func (d *databaseBranchesService) List(ctx context.Context, listReq *ListDatabaseBranchesRequest, opts ...ListOption) ([]*DatabaseBranch, error) {
	path := databaseBranchesAPIPath(listReq.Organization, listReq.Database)

	defaultOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		err := opt(defaultOpts)
		if err != nil {
			return nil, err
		}
	}

	req, err := d.client.newRequest(http.MethodGet, path, nil, WithQueryParams(*defaultOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	dbBranches := &databaseBranchesResponse{}
	if err := d.client.do(ctx, req, &dbBranches); err != nil {
		return nil, err
	}

	return dbBranches.Branches, nil
}

// Delete deletes a database branch from an organization's database.
func (d *databaseBranchesService) Delete(ctx context.Context, deleteReq *DeleteDatabaseBranchRequest) error {
	path := path.Join(databaseBranchesAPIPath(deleteReq.Organization, deleteReq.Database), deleteReq.Branch)

	var opts []RequestOption
	if deleteReq.DeleteDescendants {
		v := url.Values{}
		v.Set("delete_descendants", "true")
		opts = append(opts, WithQueryParams(v))
	}

	req, err := d.client.newRequest(http.MethodDelete, path, nil, opts...)
	if err != nil {
		return fmt.Errorf("error creating request for delete branch: %w", err)
	}

	err = d.client.do(ctx, req, nil)
	return err
}

// RefreshSchema refreshes the schema for a
func (d *databaseBranchesService) RefreshSchema(ctx context.Context, refreshReq *RefreshSchemaRequest) error {
	path := path.Join(databaseBranchesAPIPath(refreshReq.Organization, refreshReq.Database), refreshReq.Branch, "refresh-schema")
	req, err := d.client.newRequest(http.MethodPost, path, nil)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	if err := d.client.do(ctx, req, nil); err != nil {
		return err
	}

	return nil
}

// Promote promotes a branch from development to production.
func (d *databaseBranchesService) Promote(ctx context.Context, promoteReq *PromoteRequest) (*DatabaseBranch, error) {
	path := path.Join(databaseBranchAPIPath(promoteReq.Organization, promoteReq.Database, promoteReq.Branch), "promote")
	req, err := d.client.newRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for branch promotion: %w", err)
	}

	branch := &DatabaseBranch{}
	err = d.client.do(ctx, req, &branch)
	if err != nil {
		return nil, err
	}

	return branch, nil
}

// EnableSafeMigrations enables safe migrations for a production branch. This
// will prevent DDL statements from being performed on the branch.
func (d *databaseBranchesService) EnableSafeMigrations(ctx context.Context, enableReq *EnableSafeMigrationsRequest) (*DatabaseBranch, error) {
	path := path.Join(databaseBranchAPIPath(enableReq.Organization, enableReq.Database, enableReq.Branch), "safe-migrations")
	req, err := d.client.newRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for enabling safe migrations: %w", err)
	}

	branch := &DatabaseBranch{}
	err = d.client.do(ctx, req, &branch)
	if err != nil {
		return nil, err
	}

	return branch, nil
}

// DisableSafeMigrations disables safe migrations for a production branch. This
// will allow DDL statements to be performed on the branch.
func (d *databaseBranchesService) DisableSafeMigrations(ctx context.Context, disableReq *DisableSafeMigrationsRequest) (*DatabaseBranch, error) {
	path := path.Join(databaseBranchAPIPath(disableReq.Organization, disableReq.Database, disableReq.Branch), "safe-migrations")
	req, err := d.client.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for disabling safe migrations: %w", err)
	}

	branch := &DatabaseBranch{}
	err = d.client.do(ctx, req, &branch)
	if err != nil {
		return nil, err
	}

	return branch, nil
}

// Demote demotes a branch from production to development. If the branch belongs
// to an Enterprise organization, it will return a demote request and require a
// second call by a different admin in order to complete demotion.
func (d *databaseBranchesService) Demote(ctx context.Context, demoteReq *DemoteRequest) (*DatabaseBranch, error) {
	path := path.Join(databaseBranchAPIPath(demoteReq.Organization, demoteReq.Database, demoteReq.Branch), "demote")
	req, err := d.client.newRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for branch demotion: %w", err)
	}

	branch := &DatabaseBranch{}
	err = d.client.do(ctx, req, &branch)
	if err != nil {
		return nil, err
	}

	return branch, nil
}

// lintSchemaResponse represents the response from the lint schema endpoint.
type lintSchemaResponse struct {
	Errors []*SchemaLintError `json:"data"`
}

// LintSchema lints the current schema of a branch and returns any errors that
// may be present.
func (d *databaseBranchesService) LintSchema(ctx context.Context, lintReq *LintSchemaRequest) ([]*SchemaLintError, error) {
	path := path.Join(databaseBranchAPIPath(lintReq.Organization, lintReq.Database, lintReq.Branch), "schema", "lint")
	req, err := d.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for linting branch schema: %w", err)
	}

	lintResp := &lintSchemaResponse{}
	err = d.client.do(ctx, req, &lintResp)
	if err != nil {
		return nil, err
	}

	return lintResp.Errors, nil
}

func (o *databaseBranchesService) ListClusterSKUs(ctx context.Context, listReq *ListBranchClusterSKUsRequest, opts ...ListOption) ([]*ClusterSKU, error) {
	path := path.Join(databaseBranchAPIPath(listReq.Organization, listReq.Database, listReq.Branch), "cluster-size-skus")

	defaultOpts := defaultListOptions()
	for _, opt := range opts {
		err := opt(defaultOpts)
		if err != nil {
			return nil, err
		}
	}

	req, err := o.client.newRequest(http.MethodGet, path, nil, WithQueryParams(*defaultOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	clusterSKUs := []*ClusterSKU{}
	if err := o.client.do(ctx, req, &clusterSKUs); err != nil {
		return nil, err
	}

	return clusterSKUs, nil
}

func databaseBranchesAPIPath(org, db string) string {
	return path.Join(databasesAPIPath(org), db, "branches")
}

func databaseBranchAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchesAPIPath(org, db), branch)
}
