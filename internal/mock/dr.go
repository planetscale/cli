package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type DeployRequestsService struct {
	ApplyFn        func(context.Context, *ps.ApplyDeployRequestRequest) (*ps.DeployRequest, error)
	ApplyFnInvoked bool

	ForceCutoverFn        func(context.Context, *ps.ForceCutoverDeployRequestRequest) (*ps.DeployRequest, error)
	ForceCutoverFnInvoked bool

	UnblockFn        func(context.Context, *ps.UnblockDeployRequestRequest) (*ps.DeployRequest, error)
	UnblockFnInvoked bool

	AutoApplyFn        func(context.Context, *ps.AutoApplyDeployRequestRequest) (*ps.DeployRequest, error)
	AutoApplyFnInvoked bool

	AutoDeleteBranchFn        func(context.Context, *ps.AutoDeleteBranchRequest) (*ps.DeployRequest, error)
	AutoDeleteBranchFnInvoked bool

	CancelFn        func(context.Context, *ps.CancelDeployRequestRequest) (*ps.DeployRequest, error)
	CancelFnInvoked bool

	CloseFn        func(context.Context, *ps.CloseDeployRequestRequest) (*ps.DeployRequest, error)
	CloseFnInvoked bool

	CreateFn        func(context.Context, *ps.CreateDeployRequestRequest) (*ps.DeployRequest, error)
	CreateFnInvoked bool

	CreateReviewFn        func(context.Context, *ps.ReviewDeployRequestRequest) (*ps.DeployRequestReview, error)
	CreateReviewFnInvoked bool

	DeployFn        func(context.Context, *ps.PerformDeployRequest) (*ps.DeployRequest, error)
	DeployFnInvoked bool

	DiffFn        func(context.Context, *ps.DiffRequest) ([]*ps.Diff, error)
	DiffFnInvoked bool

	GetFn        func(context.Context, *ps.GetDeployRequestRequest) (*ps.DeployRequest, error)
	GetFnInvoked bool

	ListFn        func(context.Context, *ps.ListDeployRequestsRequest) ([]*ps.DeployRequest, error)
	ListFnInvoked bool

	RevertDeployFn        func(context.Context, *ps.RevertDeployRequestRequest) (*ps.DeployRequest, error)
	RevertDeployFnInvoked bool

	SkipRevertDeployFn        func(context.Context, *ps.SkipRevertDeployRequestRequest) (*ps.DeployRequest, error)
	SkipRevertDeployFnInvoked bool

	AutoApplyDeployFn        func(context.Context, *ps.AutoApplyDeployRequestRequest) (*ps.DeployRequest, error)
	AutoApplyDeployFnInvoked bool

	GetDeployOperationsFn        func(context.Context, *ps.GetDeployOperationsRequest) ([]*ps.DeployOperation, error)
	GetDeployOperationsFnInvoked bool

	GetDeployQueueFn        func(context.Context, *ps.GetDeployQueueRequest) ([]*ps.Deployment, error)
	GetDeployQueueFnInvoked bool

	GetDeploymentFn        func(context.Context, *ps.GetDeploymentRequest) (*ps.Deployment, error)
	GetDeploymentFnInvoked bool

	ListReviewsFn        func(context.Context, *ps.ListDeployRequestReviewsRequest) ([]*ps.DeployRequestReview, error)
	ListReviewsFnInvoked bool

	CheckStorageFn        func(context.Context, *ps.CheckDeployRequestStorageRequest) (*ps.DeployRequestStorageCheck, error)
	CheckStorageFnInvoked bool

	GetThrottlerFn        func(context.Context, *ps.GetDeployRequestThrottlerRequest) (*ps.DeployRequestThrottler, error)
	GetThrottlerFnInvoked bool

	UpdateThrottlerFn        func(context.Context, *ps.UpdateDeployRequestThrottlerRequest) (*ps.DeployRequestThrottler, error)
	UpdateThrottlerFnInvoked bool
}

func (d *DeployRequestsService) ApplyDeploy(ctx context.Context, req *ps.ApplyDeployRequestRequest) (*ps.DeployRequest, error) {
	d.ApplyFnInvoked = true
	return d.ApplyFn(ctx, req)
}

func (d *DeployRequestsService) ForceCutover(ctx context.Context, req *ps.ForceCutoverDeployRequestRequest) (*ps.DeployRequest, error) {
	d.ForceCutoverFnInvoked = true
	return d.ForceCutoverFn(ctx, req)
}

func (d *DeployRequestsService) UnblockDeploy(ctx context.Context, req *ps.UnblockDeployRequestRequest) (*ps.DeployRequest, error) {
	d.UnblockFnInvoked = true
	return d.UnblockFn(ctx, req)
}

func (d *DeployRequestsService) AutoApplyDeploy(ctx context.Context, req *ps.AutoApplyDeployRequestRequest) (*ps.DeployRequest, error) {
	d.AutoApplyFnInvoked = true
	return d.AutoApplyFn(ctx, req)
}

func (d *DeployRequestsService) AutoDeleteBranch(ctx context.Context, req *ps.AutoDeleteBranchRequest) (*ps.DeployRequest, error) {
	d.AutoDeleteBranchFnInvoked = true
	return d.AutoDeleteBranchFn(ctx, req)
}

func (d *DeployRequestsService) CancelDeploy(ctx context.Context, req *ps.CancelDeployRequestRequest) (*ps.DeployRequest, error) {
	d.CancelFnInvoked = true
	return d.CancelFn(ctx, req)
}

func (d *DeployRequestsService) CloseDeploy(ctx context.Context, req *ps.CloseDeployRequestRequest) (*ps.DeployRequest, error) {
	d.CloseFnInvoked = true
	return d.CloseFn(ctx, req)
}

func (d *DeployRequestsService) Create(ctx context.Context, req *ps.CreateDeployRequestRequest) (*ps.DeployRequest, error) {
	d.CreateFnInvoked = true
	return d.CreateFn(ctx, req)
}

func (d *DeployRequestsService) CreateReview(ctx context.Context, req *ps.ReviewDeployRequestRequest) (*ps.DeployRequestReview, error) {
	d.CreateReviewFnInvoked = true
	return d.CreateReviewFn(ctx, req)
}

func (d *DeployRequestsService) Deploy(ctx context.Context, req *ps.PerformDeployRequest) (*ps.DeployRequest, error) {
	d.DeployFnInvoked = true
	return d.DeployFn(ctx, req)
}

func (d *DeployRequestsService) Diff(ctx context.Context, req *ps.DiffRequest) ([]*ps.Diff, error) {
	d.DiffFnInvoked = true
	return d.DiffFn(ctx, req)
}

func (d *DeployRequestsService) Get(ctx context.Context, req *ps.GetDeployRequestRequest) (*ps.DeployRequest, error) {
	d.GetFnInvoked = true
	return d.GetFn(ctx, req)
}

func (d *DeployRequestsService) List(ctx context.Context, req *ps.ListDeployRequestsRequest) ([]*ps.DeployRequest, error) {
	d.ListFnInvoked = true
	return d.ListFn(ctx, req)
}

func (d *DeployRequestsService) RevertDeploy(ctx context.Context, req *ps.RevertDeployRequestRequest) (*ps.DeployRequest, error) {
	d.RevertDeployFnInvoked = true
	return d.RevertDeployFn(ctx, req)
}

func (d *DeployRequestsService) SkipRevertDeploy(ctx context.Context, req *ps.SkipRevertDeployRequestRequest) (*ps.DeployRequest, error) {
	d.SkipRevertDeployFnInvoked = true
	return d.SkipRevertDeployFn(ctx, req)
}

func (d *DeployRequestsService) GetDeployOperations(ctx context.Context, req *ps.GetDeployOperationsRequest) ([]*ps.DeployOperation, error) {
	d.GetDeployOperationsFnInvoked = true
	return d.GetDeployOperationsFn(ctx, req)
}

func (d *DeployRequestsService) GetDeployQueue(ctx context.Context, req *ps.GetDeployQueueRequest) ([]*ps.Deployment, error) {
	d.GetDeployQueueFnInvoked = true
	return d.GetDeployQueueFn(ctx, req)
}

func (d *DeployRequestsService) GetDeployment(ctx context.Context, req *ps.GetDeploymentRequest) (*ps.Deployment, error) {
	d.GetDeploymentFnInvoked = true
	return d.GetDeploymentFn(ctx, req)
}

func (d *DeployRequestsService) ListReviews(ctx context.Context, req *ps.ListDeployRequestReviewsRequest) ([]*ps.DeployRequestReview, error) {
	d.ListReviewsFnInvoked = true
	return d.ListReviewsFn(ctx, req)
}

func (d *DeployRequestsService) CheckStorage(ctx context.Context, req *ps.CheckDeployRequestStorageRequest) (*ps.DeployRequestStorageCheck, error) {
	d.CheckStorageFnInvoked = true
	return d.CheckStorageFn(ctx, req)
}

func (d *DeployRequestsService) GetThrottler(ctx context.Context, req *ps.GetDeployRequestThrottlerRequest) (*ps.DeployRequestThrottler, error) {
	d.GetThrottlerFnInvoked = true
	return d.GetThrottlerFn(ctx, req)
}

func (d *DeployRequestsService) UpdateThrottler(ctx context.Context, req *ps.UpdateDeployRequestThrottlerRequest) (*ps.DeployRequestThrottler, error) {
	d.UpdateThrottlerFnInvoked = true
	return d.UpdateThrottlerFn(ctx, req)
}
