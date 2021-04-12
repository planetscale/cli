package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type DatabaseBranchesService struct {
	CreateFn        func(context.Context, *ps.CreateDatabaseBranchRequest) (*ps.DatabaseBranch, error)
	CreateFnInvoked bool

	ListFn        func(context.Context, *ps.ListDatabaseBranchesRequest) ([]*ps.DatabaseBranch, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error)
	GetFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeleteDatabaseBranchRequest) error
	DeleteFnInvoked bool

	GetStatusFn      func(context.Context, *ps.GetDatabaseBranchStatusRequest) (*ps.DatabaseBranchStatus, error)
	GetStatusInvoked bool
}

func (d *DatabaseBranchesService) Create(ctx context.Context, req *ps.CreateDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
	d.CreateFnInvoked = true
	return d.CreateFn(ctx, req)
}

func (d *DatabaseBranchesService) List(ctx context.Context, req *ps.ListDatabaseBranchesRequest) ([]*ps.DatabaseBranch, error) {
	d.ListFnInvoked = true
	return d.ListFn(ctx, req)
}

func (d *DatabaseBranchesService) Get(ctx context.Context, req *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
	d.GetFnInvoked = true
	return d.GetFn(ctx, req)
}

func (d *DatabaseBranchesService) Delete(ctx context.Context, req *ps.DeleteDatabaseBranchRequest) error {
	d.DeleteFnInvoked = true
	return d.DeleteFn(ctx, req)
}

func (d *DatabaseBranchesService) GetStatus(ctx context.Context, req *ps.GetDatabaseBranchStatusRequest) (*ps.DatabaseBranchStatus, error) {
	d.GetStatusInvoked = true
	return d.GetStatusFn(ctx, req)
}
