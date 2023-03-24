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

	RequestPromotionFn        func(context.Context, *ps.RequestPromotionRequest) (*ps.BranchPromotionRequest, error)
	RequestPromotionFnInvoked bool

	GetPromotionRequestFn        func(context.Context, *ps.GetPromotionRequestRequest) (*ps.BranchPromotionRequest, error)
	GetPromotionRequestFnInvoked bool

	DemoteFn        func(context.Context, *ps.DemoteRequest) (*ps.BranchDemotionRequest, error)
	DemoteFnInvoked bool

	GetDemotionRequestFn        func(context.Context, *ps.GetDemotionRequestRequest) (*ps.BranchDemotionRequest, error)
	GetDemotionRequestFnInvoked bool

	DenyDemotionRequestFn        func(context.Context, *ps.DenyDemotionRequestRequest) error
	DenyDemotionRequestFnInvoked bool

	EnableSafeMigrationsFn        func(context.Context, *ps.EnableSafeMigrationsRequest) (*ps.DatabaseBranch, error)
	EnableSafeMigrationsFnInvoked bool

	DisableSafeMigrationsFn        func(context.Context, *ps.DisableSafeMigrationsRequest) (*ps.DatabaseBranch, error)
	DisableSafeMigrationsFnInvoked bool

	PromoteFn        func(context.Context, *ps.PromoteRequest) (*ps.DatabaseBranch, error)
	PromoteFnInvoked bool
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
	d.RequestPromotionFnInvoked = true
	return d.RequestPromotionFn(ctx, req)
}

func (d *DatabaseBranchesService) GetPromotionRequest(ctx context.Context, req *ps.GetPromotionRequestRequest) (*ps.BranchPromotionRequest, error) {
	d.GetPromotionRequestFnInvoked = true
	return d.GetPromotionRequestFn(ctx, req)
}

func (d *DatabaseBranchesService) Demote(ctx context.Context, req *ps.DemoteRequest) (*ps.BranchDemotionRequest, error) {
	d.DemoteFnInvoked = true
	return d.DemoteFn(ctx, req)
}

func (d *DatabaseBranchesService) GetDemotionRequest(ctx context.Context, req *ps.GetDemotionRequestRequest) (*ps.BranchDemotionRequest, error) {
	d.GetDemotionRequestFnInvoked = true
	return d.GetDemotionRequestFn(ctx, req)
}

func (d *DatabaseBranchesService) DenyDemotionRequest(ctx context.Context, req *ps.DenyDemotionRequestRequest) error {
	d.DenyDemotionRequestFnInvoked = true
	return d.DenyDemotionRequestFn(ctx, req)
}

func (d *DatabaseBranchesService) EnableSafeMigrations(ctx context.Context, req *ps.EnableSafeMigrationsRequest) (*ps.DatabaseBranch, error) {
	d.EnableSafeMigrationsFnInvoked = true
	return d.EnableSafeMigrationsFn(ctx, req)
}

func (d *DatabaseBranchesService) DisableSafeMigrations(ctx context.Context, req *ps.DisableSafeMigrationsRequest) (*ps.DatabaseBranch, error) {
	d.DisableSafeMigrationsFnInvoked = true
	return d.DisableSafeMigrationsFn(ctx, req)
}

func (d *DatabaseBranchesService) Promote(ctx context.Context, req *ps.PromoteRequest) (*ps.DatabaseBranch, error) {
	d.PromoteFnInvoked = true
	return d.Promote(ctx, req)
}
