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
	GetDeployQueue(context.Context, *GetDeployQueueRequest) ([]*Deployment, error)
	GetDeployment(context.Context, *GetDeploymentRequest) (*Deployment, error)
	ListReviews(context.Context, *ListDeployRequestReviewsRequest) ([]*DeployRequestReview, error)
	CheckStorage(context.Context, *CheckDeployRequestStorageRequest) (*DeployRequestStorageCheck, error)
	GetThrottler(context.Context, *GetDeployRequestThrottlerRequest) (*DeployRequestThrottler, error)
	UpdateThrottler(context.Context, *UpdateDeployRequestThrottlerRequest) (*DeployRequestThrottler, error)
	SkipRevertDeploy(context.Context, *SkipRevertDeployRequestRequest) (*DeployRequest, error)
	RevertDeploy(context.Context, *RevertDeployRequestRequest) (*DeployRequest, error)
	UnblockDeploy(context.Context, *UnblockDeployRequestRequest) (*DeployRequest, error)
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

	AutoCutover         bool   `json:"auto_cutover"`
	AutoDeleteBranch    bool   `json:"auto_delete_branch"`
	CutoverExpiring     bool   `json:"cutover_expiring"`
	TableLocked         bool   `json:"table_locked"`
	QueuePaused         bool   `json:"queue_paused"`
	ParallelLaneBlocked bool   `json:"parallel_lane_blocked"`
	DeployCheckErrors   string `json:"deploy_check_errors"`
	LockedTableName     string `json:"locked_table_name"`
	QueuePauseReason    string `json:"queue_pause_reason"`
	Strategy            string `json:"strategy"`

	Actor          *Actor `json:"actor"`
	CutoverActor   *Actor `json:"cutover_actor"`
	CancelledActor *Actor `json:"cancelled_actor"`

	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
	StartedAt               *time.Time `json:"started_at"`
	QueuedAt                *time.Time `json:"queued_at"`
	FinishedAt              *time.Time `json:"finished_at"`
	SubmittedAt             *time.Time `json:"submitted_at"`
	CutoverAt               *time.Time `json:"cutover_at"`
	ReadyToCutoverAt        *time.Time `json:"ready_to_cutover_at"`
	ForceCutoverRequestedAt *time.Time `json:"force_cutover_requested_at"`
	SchemaLastUpdatedAt     *time.Time `json:"schema_last_updated_at"`
}

// GetDeployQueueRequest gets the deploy queue for a database.
type GetDeployQueueRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
}

// GetDeploymentRequest gets the deployment for a deploy request.
type GetDeploymentRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

// ListDeployRequestReviewsRequest lists reviews for a deploy request.
type ListDeployRequestReviewsRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

// CheckDeployRequestStorageRequest checks storage readiness for a deploy request.
type CheckDeployRequestStorageRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

// DeployRequestStorageCheck is the storage check response for a deploy request.
type DeployRequestStorageCheck struct {
	EnoughStorage      bool           `json:"enough_storage"`
	Upgradeable        bool           `json:"upgradeable"`
	StorageBytesNeeded int64          `json:"storage_bytes_needed"`
	StorageReport      map[string]any `json:"storage_report"`
}

// GetDeployRequestThrottlerRequest gets throttler config for a deploy request.
type GetDeployRequestThrottlerRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

// UpdateDeployRequestThrottlerRequest updates throttler config for a deploy request.
type UpdateDeployRequestThrottlerRequest struct {
	Organization   string                          `json:"-"`
	Database       string                          `json:"-"`
	Number         uint64                          `json:"-"`
	Ratio          *int                            `json:"ratio,omitempty"`
	Configurations []*UpdateThrottlerConfiguration `json:"configurations,omitempty"`
}

// UpdateThrottlerConfiguration is a per-keyspace throttler ratio for updates.
type UpdateThrottlerConfiguration struct {
	KeyspaceName string `json:"keyspace_name"`
	Ratio        int    `json:"ratio"`
}

// DeployRequestThrottler is the throttler configuration for a deploy request.
type DeployRequestThrottler struct {
	Keyspaces      []string                  `json:"keyspaces"`
	Configurable   *ThrottlerConfigurable    `json:"configurable"`
	Configurations []*ThrottlerConfiguration `json:"configurations"`
}

// ThrottlerConfigurable identifies the resource owning throttler config.
type ThrottlerConfigurable struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

// ThrottlerConfiguration is a per-keyspace throttler ratio.
type ThrottlerConfiguration struct {
	KeyspaceName string  `json:"keyspace_name"`
	Ratio        float64 `json:"ratio"`
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

// UnblockDeployRequestRequest unblocks the deploy queue after a failed deploy
// or revert (complete_error / complete_revert_error).
type UnblockDeployRequestRequest struct {
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

// UnblockDeploy marks a failed deploy or revert complete so the queue can proceed.
func (d *deployRequestsService) UnblockDeploy(ctx context.Context, unblockReq *UnblockDeployRequestRequest) (*DeployRequest, error) {
	path := deployRequestActionAPIPath(unblockReq.Organization, unblockReq.Database, unblockReq.Number, "complete-deploy")
	req, err := d.client.newRequest(http.MethodPost, path, unblockReq)
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

type deployQueueResponse struct {
	Deployments []*Deployment `json:"data"`
}

func (d *deployRequestsService) GetDeployQueue(ctx context.Context, getReq *GetDeployQueueRequest) ([]*Deployment, error) {
	pathStr := path.Join(databasesAPIPath(getReq.Organization), getReq.Database, "deploy-queue")
	req, err := d.client.newRequest(http.MethodGet, pathStr, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp := &deployQueueResponse{}
	if err := d.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Deployments, nil
}

func (d *deployRequestsService) GetDeployment(ctx context.Context, getReq *GetDeploymentRequest) (*Deployment, error) {
	req, err := d.client.newRequest(http.MethodGet, deployRequestActionAPIPath(getReq.Organization, getReq.Database, getReq.Number, "deployment"), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	deployment := &Deployment{}
	if err := d.client.do(ctx, req, deployment); err != nil {
		return nil, err
	}

	return deployment, nil
}

type deployRequestReviewsResponse struct {
	Reviews []*DeployRequestReview `json:"data"`
}

func (d *deployRequestsService) ListReviews(ctx context.Context, listReq *ListDeployRequestReviewsRequest) ([]*DeployRequestReview, error) {
	req, err := d.client.newRequest(http.MethodGet, deployRequestActionAPIPath(listReq.Organization, listReq.Database, listReq.Number, "reviews"), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp := &deployRequestReviewsResponse{}
	if err := d.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Reviews, nil
}

func (d *deployRequestsService) CheckStorage(ctx context.Context, checkReq *CheckDeployRequestStorageRequest) (*DeployRequestStorageCheck, error) {
	req, err := d.client.newRequest(http.MethodGet, deployRequestActionAPIPath(checkReq.Organization, checkReq.Database, checkReq.Number, "storage-check"), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	check := &DeployRequestStorageCheck{}
	if err := d.client.do(ctx, req, check); err != nil {
		return nil, err
	}

	return check, nil
}

func (d *deployRequestsService) GetThrottler(ctx context.Context, getReq *GetDeployRequestThrottlerRequest) (*DeployRequestThrottler, error) {
	req, err := d.client.newRequest(http.MethodGet, deployRequestActionAPIPath(getReq.Organization, getReq.Database, getReq.Number, "throttler"), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	throttler := &DeployRequestThrottler{}
	if err := d.client.do(ctx, req, throttler); err != nil {
		return nil, err
	}

	return throttler, nil
}

func (d *deployRequestsService) UpdateThrottler(ctx context.Context, updateReq *UpdateDeployRequestThrottlerRequest) (*DeployRequestThrottler, error) {
	req, err := d.client.newRequest(http.MethodPatch, deployRequestActionAPIPath(updateReq.Organization, updateReq.Database, updateReq.Number, "throttler"), updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	throttler := &DeployRequestThrottler{}
	if err := d.client.do(ctx, req, throttler); err != nil {
		return nil, err
	}

	return throttler, nil
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
