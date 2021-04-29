package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

type deployRequestsService struct {
	client *Client
}

var _ DeployRequestsService = (*deployRequestsService)(nil)

// DeployRequestsService is an interface for communicating with the PlanetScale
// deploy requests API.
type DeployRequestsService interface {
	CancelDeploy(context.Context, *CancelDeployRequestRequest) (*DeployRequest, error)
	CloseDeploy(context.Context, *CloseDeployRequestRequest) (*DeployRequest, error)
	Create(context.Context, *CreateDeployRequestRequest) (*DeployRequest, error)
	CreateReview(context.Context, *ReviewDeployRequestRequest) (*DeployRequestReview, error)
	Deploy(context.Context, *PerformDeployRequest) (*DeployRequest, error)
	Diff(ctx context.Context, diffReq *DiffRequest) ([]*Diff, error)
	Get(context.Context, *GetDeployRequestRequest) (*DeployRequest, error)
	List(context.Context, *ListDeployRequestsRequest) ([]*DeployRequest, error)
}

// DeployRequestReview posts a review to a deploy request.
type DeployRequestReview struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
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

	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	StartedAt  *time.Time `json:"started_at"`
	QueuedAt   *time.Time `json:"queued_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

// Deployment encapsulates a deployment for a deploy request.
type Deployment struct {
	ID                   string              `json:"id"`
	State                string              `json:"state"`
	Deployable           bool                `json:"deployable"`
	DeployRequestNumber  uint64              `json:"deploy_request_number"`
	IntoBranch           string              `json:"into_branch"`
	PrecedingDeployments []*QueuedDeployment `json:"preceding_deployments"`

	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	StartedAt  *time.Time `json:"started_at"`
	QueuedAt   *time.Time `json:"queued_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

// DeployRequest encapsulates the request to deploy a database branch's schema
// to a production branch
type DeployRequest struct {
	ID string `json:"id"`

	Branch     string `json:"branch"`
	IntoBranch string `json:"into_branch"`

	Number uint64 `json:"number"`

	State string `json:"state"`

	Approved bool `json:"approved"`

	Notes string `json:"notes"`

	Deployment *Deployment `json:"deployment"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`
}

type CancelDeployRequestRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Number       uint64 `json:"-"`
}

type CreateDeployRequestRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"branch"`
	IntoBranch   string `json:"into_branch"`
	Notes        string `json:"notes"`
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
		return nil, errors.Wrap(err, "error creating http request")
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
		return nil, errors.Wrap(err, "error creating http request")
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
		return nil, errors.Wrap(err, "error creating http request")
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
		return nil, errors.Wrap(err, "error creating http request")
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
		return nil, errors.Wrap(err, "error creating http request")
	}

	diffs := &diffResponse{}
	if err := d.client.do(ctx, req, &diffs); err != nil {
		return nil, err
	}

	return diffs.Diffs, nil
}

func (d *deployRequestsService) List(ctx context.Context, listReq *ListDeployRequestsRequest) ([]*DeployRequest, error) {
	req, err := d.client.newRequest(http.MethodGet, deployRequestsAPIPath(listReq.Organization, listReq.Database), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	drReq := &deployRequestsResponse{}
	if err := d.client.do(ctx, req, &drReq); err != nil {
		return nil, err
	}

	return drReq.DeployRequests, nil
}

func (d *deployRequestsService) CreateReview(ctx context.Context, reviewReq *ReviewDeployRequestRequest) (*DeployRequestReview, error) {
	var reqBody = struct {
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
		return nil, errors.Wrap(err, "error creating http request")
	}

	drr := &DeployRequestReview{}
	if err := d.client.do(ctx, req, &drr); err != nil {
		return nil, err
	}

	return drr, nil
}

func deployRequestsAPIPath(org, db string) string {
	return fmt.Sprintf("%s/%s/deploy-requests", databasesAPIPath(org), db)
}

// deployRequestAPIPath gets the base path for accessing a single deploy request
func deployRequestAPIPath(org string, db string, number uint64) string {
	return fmt.Sprintf("%s/%s/deploy-requests/%d", databasesAPIPath(org), db, number)
}

func deployRequestActionAPIPath(org string, db string, number uint64, path string) string {
	return fmt.Sprintf("%s/%s", deployRequestAPIPath(org, db, number), path)
}
