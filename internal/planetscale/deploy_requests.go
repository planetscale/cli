package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

type deployRequestsService struct {
	client *Client
}

var _ DeployRequestsService = (*deployRequestsService)(nil)

// DeployRequestsService is an interface for communicating with the PlanetScale
// deploy requests API.
type DeployRequestsService interface {
	ApplyDeploy(context.Context, *ApplyDeployRequestRequest) (*DeployRequest, error)
	AutoApplyDeploy(context.Context, *AutoApplyDeployRequestRequest) (*DeployRequest, error)
	CancelDeploy(context.Context, *CancelDeployRequestRequest) (*DeployRequest, error)
	CloseDeploy(context.Context, *CloseDeployRequestRequest) (*DeployRequest, error)
	Create(context.Context, *CreateDeployRequestRequest) (*DeployRequest, error)
	CreateReview(context.Context, *ReviewDeployRequestRequest) (*DeployRequestReview, error)
	Deploy(context.Context, *PerformDeployRequest) (*DeployRequest, error)
	Diff(ctx context.Context, diffReq *DiffRequest) ([]*Diff, error)
	ForceCutover(context.Context, *ForceCutoverDeployRequestRequest) (*DeployRequest, error)
	Get(context.Context, *GetDeployRequestRequest) (*DeployRequest, error)
	List(context.Context, *ListDeployRequestsRequest) ([]*DeployRequest, error)
	GetDeployOperations(context.Context, *GetDeployOperationsRequest) ([]*DeployOperation, error)
	SkipRevertDeploy(context.Context, *SkipRevertDeployRequestRequest) (*DeployRequest, error)
	RevertDeploy(context.Context, *RevertDeployRequestRequest) (*DeployRequest, error)
}

// DeployRequestReview posts a review to a deploy request.
type DeployRequestReview struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	Actor     Actor     `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PerformDeployRequest is a request for approving and deploying a deploy request.
// NOTE: We deviate from naming convention here because we have a data model
// named DeployRequest already.
type PerformDeployRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
	InstantDDL   bool   `json:"instant_ddl"`
	Strategy     string `json:"strategy,omitempty"`
}

// GetDeployRequest encapsulates the request for getting a single deploy
// request.
type GetDeployRequestRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

// ListDeployRequestsRequest gets the deploy requests for a specific database
// branch.
type ListDeployRequestsRequest struct {
	Organization string
	Database     string
	State        string
	Branch       string
	IntoBranch   string
}

// GetDeployOperationsRequest encapsulates the request for getting a deploy
// operation for a deploy request.
type GetDeployOperationsRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

// DeployOperation encapsulates a deploy operation within a deployment from the
// PlanetScale API.
type DeployOperation struct {
	ID                 string    `json:"id"`
	State              string    `json:"state"`
	Table              string    `json:"table_name"`
	Keyspace           string    `json:"keyspace_name"`
	Operation          string    `json:"operation_name"`
	ETASeconds         int64     `json:"eta_seconds"`
	ProgressPercentage uint64    `json:"progress_percentage"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// QueuedDeployment encapsulates a deployment that is in the queue.
type QueuedDeployment struct {
	ID                  string `json:"id"`
	State               string `json:"state"`
	DeployRequestNumber uint64 `json:"deploy_request_number"`
	IntoBranch          string `json:"into_branch"`

	Actor *Actor `json:"actor"`

	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	StartedAt  *time.Time `json:"started_at"`
	QueuedAt   *time.Time `json:"queued_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

// DeploymentLintError represents an error that occurs during the deployment
// flow.
type DeploymentLintError struct {
	LintError        string `json:"lint_error"`
	Keyspace         string `json:"keyspace_name"`
	Table            string `json:"table_name"`
	SubjectType      string `json:"subject_type"`
	ErrorDescription string `json:"error_description"`
	DocsUrl          string `json:"docs_url"`
}

// Deployment encapsulates a deployment for a deploy request.
type Deployment struct {
	ID                   string                 `json:"id"`
	State                string                 `json:"state"`
	Deployable           bool                   `json:"deployable"`
	LintErrors           []*DeploymentLintError `json:"lint_errors"`
	DeployRequestNumber  uint64                 `json:"deploy_request_number"`
	IntoBranch           string                 `json:"into_branch"`
	PrecedingDeployments []*QueuedDeployment    `json:"preceding_deployments"`

	InstantDDLEligible bool `json:"instant_ddl_eligible"`
	InstantDDL         bool `json:"instant_ddl"`

	Actor          *Actor `json:"actor"`
	CutoverActor   *Actor `json:"cutover_actor"`
	CancelledActor *Actor `json:"cancelled_actor"`

	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
	StartedAt               *time.Time `json:"started_at"`
	QueuedAt                *time.Time `json:"queued_at"`
	FinishedAt              *time.Time `json:"finished_at"`
	ForceCutoverRequestedAt *time.Time `json:"force_cutover_requested_at"`
}

// DeployRequest encapsulates the request to deploy a database branch's schema
// to a production branch
type DeployRequest struct {
	ID string `json:"id"`

	Branch     string `json:"branch"`
	IntoBranch string `json:"into_branch"`

	Actor           Actor  `json:"actor"`
	ClosedBy        *Actor `json:"closed_by"`
	BranchDeletedBy *Actor `json:"branch_deleted_by"`
	Number          uint64 `json:"number"`

	State string `json:"state"`

	DeploymentState string `json:"deployment_state"`

	Approved bool `json:"approved"`

	Notes string `json:"notes"`

	Deployment *Deployment `json:"deployment"`

	HtmlURL string `json:"html_url"`

	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ClosedAt   *time.Time `json:"closed_at"`
	DeployedAt *time.Time `json:"deployed_at"`
}

type ApplyDeployRequestRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

type ForceCutoverDeployRequestRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

type AutoApplyDeployRequestRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
	Enable       bool   `json:"-"`
}

type CancelDeployRequestRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

type CreateDeployRequestRequest struct {
	Organization     string `json:"-"`
	Database         string `json:"-"`
	Branch           string `json:"branch"`
	IntoBranch       string `json:"into_branch,omitempty"`
	Notes            string `json:"notes"`
	AutoCutover      bool   `json:"auto_cutover,omitempty"`
	AutoDeleteBranch bool   `json:"auto_delete_branch,omitempty"`
}

type SkipRevertDeployRequestRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

type RevertDeployRequestRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

type ReviewDeployRequestRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`

	// CommentText represents the comment body to be posted
	CommentText string `json:"-"`

	// ReviewAction defines the action for an individual review.
	ReviewAction ReviewAction `json:"-"`
}

// ReviewAction defines the action for an individual review.
type ReviewAction int

const (
	// Comment is used to comment a Review with a custom text.
	ReviewComment ReviewAction = iota

	// Approve is used to approve a Review.
	ReviewApprove
)

func (r ReviewAction) String() string {
	switch r {
	case ReviewApprove:
		return "approved"
	case ReviewComment:
		fallthrough
	default:
		return "commented"
	}
}

type CloseDeployRequestRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

func NewDeployRequestsService(client *Client) *deployRequestsService {
	return &deployRequestsService{
		client: client,
	}
}

// Get fetches a single deploy request.
func (d *deployRequestsService) Get(ctx context.Context, getReq *GetDeployRequestRequest) (*DeployRequest, error) {
	req, err := d.client.newRequest(http.MethodGet, deployRequestAPIPath(getReq.Organization, getReq.Database, getReq.Number), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	dr := &DeployRequest{}
	if err := d.client.do(ctx, req, &dr); err != nil {
		return nil, err
	}

	return dr, nil
}

type CloseRequest struct {
	State string `json:"state"`
}

// CloseDeploy closes a deploy request
func (d *deployRequestsService) CloseDeploy(ctx context.Context, closeReq *CloseDeployRequestRequest) (*DeployRequest, error) {
	updateReq := &CloseRequest{
		State: "closed",
	}

	req, err := d.client.newRequest(http.MethodPatch, deployRequestAPIPath(closeReq.Organization, closeReq.Database, closeReq.Number), updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	dr := &DeployRequest{}
	if err := d.client.do(ctx, req, &dr); err != nil {
		return nil, err
	}

	return dr, nil
}

// Deploy approves and executes a specific deploy request.
func (d *deployRequestsService) Deploy(ctx context.Context, deployReq *PerformDeployRequest) (*DeployRequest, error) {
	path := deployRequestActionAPIPath(deployReq.Organization, deployReq.Database, deployReq.Number, "deploy")
	req, err := d.client.newRequest(http.MethodPost, path, deployReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	dr := &DeployRequest{}
	if err := d.client.do(ctx, req, &dr); err != nil {
		return nil, err
	}

	return dr, nil
}

type deployRequestsResponse struct {
	DeployRequests []*DeployRequest `json:"data"`
}

func (d *deployRequestsService) Create(ctx context.Context, createReq *CreateDeployRequestRequest) (*DeployRequest, error) {
	path := deployRequestsAPIPath(createReq.Organization, createReq.Database)
	req, err := d.client.newRequest(http.MethodPost, path, createReq)
	if err != nil {
		return nil, err
	}

	dr := &DeployRequest{}
	if err := d.client.do(ctx, req, &dr); err != nil {
		return nil, err
	}
	return dr, nil
}

// CancelDeploy cancels a queued deploy request.
func (d *deployRequestsService) CancelDeploy(ctx context.Context, deployReq *CancelDeployRequestRequest) (*DeployRequest, error) {
	path := deployRequestActionAPIPath(deployReq.Organization, deployReq.Database, deployReq.Number, "cancel")
	req, err := d.client.newRequest(http.MethodPost, path, deployReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	dr := &DeployRequest{}
	if err := d.client.do(ctx, req, &dr); err != nil {
		return nil, err
	}

	return dr, nil
}

func (d *deployRequestsService) ApplyDeploy(ctx context.Context, applyReq *ApplyDeployRequestRequest) (*DeployRequest, error) {
	path := deployRequestActionAPIPath(applyReq.Organization, applyReq.Database, applyReq.Number, "apply-deploy")
	req, err := d.client.newRequest(http.MethodPost, path, applyReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	drr := &DeployRequest{}
	if err := d.client.do(ctx, req, &drr); err != nil {
		return nil, err
	}

	return drr, nil
}

// ForceCutover requests a force cutover for a deploy request stuck in the cutover phase.
func (d *deployRequestsService) ForceCutover(ctx context.Context, forceReq *ForceCutoverDeployRequestRequest) (*DeployRequest, error) {
	path := deployRequestActionAPIPath(forceReq.Organization, forceReq.Database, forceReq.Number, "force-cutover")
	req, err := d.client.newRequest(http.MethodPost, path, forceReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	drr := &DeployRequest{}
	if err := d.client.do(ctx, req, &drr); err != nil {
		return nil, err
	}

	return drr, nil
}

func (d *deployRequestsService) AutoApplyDeploy(ctx context.Context, autoApplyReq *AutoApplyDeployRequestRequest) (*DeployRequest, error) {
	reqBody := struct {
		Enable bool `json:"enable"`
	}{
		Enable: autoApplyReq.Enable,
	}

	path := deployRequestActionAPIPath(autoApplyReq.Organization, autoApplyReq.Database, autoApplyReq.Number, "auto-apply")
	req, err := d.client.newRequest(http.MethodPut, path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	drr := &DeployRequest{}
	if err := d.client.do(ctx, req, &drr); err != nil {
		return nil, err
	}

	return drr, nil
}

// SkipRevert skips a pending revert of a completed deploy request
func (d *deployRequestsService) SkipRevertDeploy(ctx context.Context, deployReq *SkipRevertDeployRequestRequest) (*DeployRequest, error) {
	path := deployRequestActionAPIPath(deployReq.Organization, deployReq.Database, deployReq.Number, "skip-revert")
	req, err := d.client.newRequest(http.MethodPost, path, deployReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	dr := &DeployRequest{}
	if err := d.client.do(ctx, req, &dr); err != nil {
		return nil, err
	}

	return dr, nil
}

// RevertDeploy reverts a completed deploy request
func (d *deployRequestsService) RevertDeploy(ctx context.Context, deployReq *RevertDeployRequestRequest) (*DeployRequest, error) {
	path := deployRequestActionAPIPath(deployReq.Organization, deployReq.Database, deployReq.Number, "revert")
	req, err := d.client.newRequest(http.MethodPost, path, deployReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	dr := &DeployRequest{}
	if err := d.client.do(ctx, req, &dr); err != nil {
		return nil, err
	}

	return dr, nil
}

// Diff returns the diff for a database deploy request
type Diff struct {
	Name string `json:"name"`
	Raw  string `json:"raw"`
	HTML string `json:"html"`
}

type diffResponse struct {
	Diffs []*Diff `json:"data"`
}

type DiffRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

// Diff returns a diff
func (d *deployRequestsService) Diff(ctx context.Context, diffReq *DiffRequest) ([]*Diff, error) {
	req, err := d.client.newRequest(
		http.MethodGet,
		deployRequestActionAPIPath(diffReq.Organization, diffReq.Database, diffReq.Number, "diff"),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	diffs := &diffResponse{}
	if err := d.client.do(ctx, req, &diffs); err != nil {
		return nil, err
	}

	return diffs.Diffs, nil
}

func (d *deployRequestsService) List(ctx context.Context, listReq *ListDeployRequestsRequest) ([]*DeployRequest, error) {
	baseURL := deployRequestsAPIPath(listReq.Organization, listReq.Database)

	queryParams := url.Values{}
	if listReq.State != "" {
		queryParams.Set("state", listReq.State)
	}
	if listReq.Branch != "" {
		queryParams.Set("branch", listReq.Branch)
	}
	if listReq.IntoBranch != "" {
		queryParams.Set("into_branch", listReq.IntoBranch)
	}

	req, err := d.client.newRequest(http.MethodGet, baseURL, nil, WithQueryParams(queryParams))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	drReq := &deployRequestsResponse{}
	if err := d.client.do(ctx, req, &drReq); err != nil {
		return nil, err
	}

	return drReq.DeployRequests, nil
}

func (d *deployRequestsService) CreateReview(ctx context.Context, reviewReq *ReviewDeployRequestRequest) (*DeployRequestReview, error) {
	reqBody := struct {
		State string `json:"state"`
		Body  string `json:"body"`
	}{
		State: reviewReq.ReviewAction.String(),
		Body:  reviewReq.CommentText,
	}

	req, err := d.client.newRequest(http.MethodPost,
		deployRequestActionAPIPath(
			reviewReq.Organization,
			reviewReq.Database,
			reviewReq.Number,
			"reviews",
		), reqBody)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	drr := &DeployRequestReview{}
	if err := d.client.do(ctx, req, &drr); err != nil {
		return nil, err
	}

	return drr, nil
}

type deployOperationResponse struct {
	Ops []*DeployOperation `json:"data"`
}

func (d *deployRequestsService) GetDeployOperations(ctx context.Context, getReq *GetDeployOperationsRequest) ([]*DeployOperation, error) {
	req, err := d.client.newRequest(http.MethodGet, deployRequestActionAPIPath(getReq.Organization, getReq.Database, getReq.Number, "operations"), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp := &deployOperationResponse{}
	if err := d.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Ops, nil
}

func deployRequestsAPIPath(org, db string) string {
	return path.Join(databasesAPIPath(org), db, "deploy-requests")
}

// deployRequestAPIPath gets the base path for accessing a single deploy request
func deployRequestAPIPath(org string, db string, number uint64) string {
	return path.Join(databasesAPIPath(org), db, "deploy-requests", fmt.Sprintf("%d", number))
}

func deployRequestActionAPIPath(org string, db string, number uint64, actionPath string) string {
	return path.Join(deployRequestAPIPath(org, db, number), actionPath)
}
