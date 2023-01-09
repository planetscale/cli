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

	DiffFn        func(context.Context, *ps.DiffBranchRequest) ([]*ps.Diff, error)
	DiffFnInvoked bool

	SchemaFn        func(context.Context, *ps.BranchSchemaRequest) ([]*ps.Diff, error)
	SchemaFnInvoked bool

	VSchemaFn        func(context.Context, *ps.BranchVSchemaRequest) (*ps.VSchemaDiff, error)
	VSchemaFnInvoked bool

	KeyspacesFn        func(context.Context, *ps.BranchKeyspacesRequest) ([]*ps.Keyspace, error)
	KeyspacesFnInvoked bool

	RefreshSchemaFn        func(context.Context, *ps.RefreshSchemaRequest) error
	RefreshSchemaFnInvoked bool

	PromoteFn        func(context.Context, *ps.RequestPromotionRequest) (*ps.BranchPromotionRequest, error)
	PromoteFnInvoked bool

	GetPromotionRequestFn        func(context.Context, *ps.GetPromotionRequestRequest) (*ps.BranchPromotionRequest, error)
	GetPromotionRequestFnInvoked bool
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

func (d *DatabaseBranchesService) Diff(ctx context.Context, req *ps.DiffBranchRequest) ([]*ps.Diff, error) {
	d.DiffFnInvoked = true
	return d.DiffFn(ctx, req)
}

func (d *DatabaseBranchesService) Schema(ctx context.Context, req *ps.BranchSchemaRequest) ([]*ps.Diff, error) {
	d.SchemaFnInvoked = true
	return d.SchemaFn(ctx, req)
}

func (d *DatabaseBranchesService) VSchema(ctx context.Context, req *ps.BranchVSchemaRequest) (*ps.VSchemaDiff, error) {
	d.VSchemaFnInvoked = true
	return d.VSchemaFn(ctx, req)
}

func (d *DatabaseBranchesService) Keyspaces(ctx context.Context, req *ps.BranchKeyspacesRequest) ([]*ps.Keyspace, error) {
	d.KeyspacesFnInvoked = true
	return d.KeyspacesFn(ctx, req)
}

func (d *DatabaseBranchesService) RefreshSchema(ctx context.Context, req *ps.RefreshSchemaRequest) error {
	d.RefreshSchemaFnInvoked = true
	return d.RefreshSchemaFn(ctx, req)
}

func (d *DatabaseBranchesService) RequestPromotion(ctx context.Context, req *ps.RequestPromotionRequest) (*ps.BranchPromotionRequest, error) {
	d.PromoteFnInvoked = true
	return d.PromoteFn(ctx, req)
}

func (d *DatabaseBranchesService) GetPromotionRequest(ctx context.Context, req *ps.GetPromotionRequestRequest) (*ps.BranchPromotionRequest, error) {
	d.GetPromotionRequestFnInvoked = true
	return d.GetPromotionRequestFn(ctx, req)
}
