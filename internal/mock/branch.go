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

	RoutingRulesFn        func(context.Context, *ps.BranchRoutingRulesRequest) (*ps.RoutingRules, error)
	RoutingRulesFnInvoked bool

	UpdateRoutingRulesFn        func(context.Context, *ps.UpdateBranchRoutingRulesRequest) (*ps.RoutingRules, error)
	UpdateRoutingRulesFnInvoked bool

	RefreshSchemaFn        func(context.Context, *ps.RefreshSchemaRequest) error
	RefreshSchemaFnInvoked bool

	DemoteFn        func(context.Context, *ps.DemoteRequest) (*ps.DatabaseBranch, error)
	DemoteFnInvoked bool

	EnableSafeMigrationsFn        func(context.Context, *ps.EnableSafeMigrationsRequest) (*ps.DatabaseBranch, error)
	EnableSafeMigrationsFnInvoked bool

	DisableSafeMigrationsFn        func(context.Context, *ps.DisableSafeMigrationsRequest) (*ps.DatabaseBranch, error)
	DisableSafeMigrationsFnInvoked bool

	PromoteFn        func(context.Context, *ps.PromoteRequest) (*ps.DatabaseBranch, error)
	PromoteFnInvoked bool

	LintSchemaFn        func(context.Context, *ps.LintSchemaRequest) ([]*ps.SchemaLintError, error)
	LintSchemaFnInvoked bool
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

func (d *DatabaseBranchesService) RoutingRules(ctx context.Context, req *ps.BranchRoutingRulesRequest) (*ps.RoutingRules, error) {
	d.RoutingRulesFnInvoked = true
	return d.RoutingRulesFn(ctx, req)
}

func (d *DatabaseBranchesService) UpdateRoutingRules(ctx context.Context, req *ps.UpdateBranchRoutingRulesRequest) (*ps.RoutingRules, error) {
	d.UpdateRoutingRulesFnInvoked = true
	return d.UpdateRoutingRulesFn(ctx, req)
}

func (d *DatabaseBranchesService) RefreshSchema(ctx context.Context, req *ps.RefreshSchemaRequest) error {
	d.RefreshSchemaFnInvoked = true
	return d.RefreshSchemaFn(ctx, req)
}

func (d *DatabaseBranchesService) Demote(ctx context.Context, req *ps.DemoteRequest) (*ps.DatabaseBranch, error) {
	d.DemoteFnInvoked = true
	return d.DemoteFn(ctx, req)
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
	return d.PromoteFn(ctx, req)
}

func (d *DatabaseBranchesService) LintSchema(ctx context.Context, req *ps.LintSchemaRequest) ([]*ps.SchemaLintError, error) {
	d.LintSchemaFnInvoked = true
	return d.LintSchemaFn(ctx, req)
}
