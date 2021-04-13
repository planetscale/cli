package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type DeployRequestsService struct {
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
