package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type DatabaseBranchesService struct {
	CreateFn        func(context.Context, *ps.CreateDatabaseBranchRequest) (*ps.DatabaseBranch, error)
	CreateFnInvoked bool

	ListFn        func(context.Context, *ps.ListDatabaseBranchesRequest, ...ps.ListOption) ([]*ps.DatabaseBranch, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error)
	GetFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdateDatabaseBranchRequest) (*ps.DatabaseBranch, error)
	UpdateFnInvoked bool

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

	ListClusterSKUsFn        func(context.Context, *ps.ListBranchClusterSKUsRequest, ...ps.ListOption) ([]*ps.ClusterSKU, error)
	ListClusterSKUsFnInvoked bool

	ResizeFn        func(context.Context, *ps.ResizeBranchRequest) (*ps.BranchResizeRequest, error)
	ResizeFnInvoked bool

	ListResizesFn        func(context.Context, *ps.ListBranchResizesRequest) ([]*ps.BranchResizeRequest, error)
	ListResizesFnInvoked bool

	CancelResizeFn        func(context.Context, *ps.CancelBranchResizeRequest) error
	CancelResizeFnInvoked bool

	ResizeStatusFn        func(context.Context, *ps.BranchResizeStatusRequest) (*ps.BranchResizeRequest, error)
	ResizeStatusFnInvoked bool
}

func (d *DatabaseBranchesService) Create(ctx context.Context, req *ps.CreateDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
	d.CreateFnInvoked = true
	return d.CreateFn(ctx, req)
}

func (d *DatabaseBranchesService) List(ctx context.Context, req *ps.ListDatabaseBranchesRequest, opts ...ps.ListOption) ([]*ps.DatabaseBranch, error) {
	d.ListFnInvoked = true
	return d.ListFn(ctx, req, opts...)
}

func (d *DatabaseBranchesService) Get(ctx context.Context, req *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
	d.GetFnInvoked = true
	return d.GetFn(ctx, req)
}

func (d *DatabaseBranchesService) Update(ctx context.Context, req *ps.UpdateDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
	d.UpdateFnInvoked = true
	return d.UpdateFn(ctx, req)
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

func (d *DatabaseBranchesService) ListClusterSKUs(ctx context.Context, req *ps.ListBranchClusterSKUsRequest, opts ...ps.ListOption) ([]*ps.ClusterSKU, error) {
	d.ListClusterSKUsFnInvoked = true
	return d.ListClusterSKUsFn(ctx, req, opts...)
}

func (d *DatabaseBranchesService) Resize(ctx context.Context, req *ps.ResizeBranchRequest) (*ps.BranchResizeRequest, error) {
	d.ResizeFnInvoked = true
	return d.ResizeFn(ctx, req)
}

func (d *DatabaseBranchesService) ListResizes(ctx context.Context, req *ps.ListBranchResizesRequest) ([]*ps.BranchResizeRequest, error) {
	d.ListResizesFnInvoked = true
	return d.ListResizesFn(ctx, req)
}

func (d *DatabaseBranchesService) CancelResize(ctx context.Context, req *ps.CancelBranchResizeRequest) error {
	d.CancelResizeFnInvoked = true
	return d.CancelResizeFn(ctx, req)
}

func (d *DatabaseBranchesService) ResizeStatus(ctx context.Context, req *ps.BranchResizeStatusRequest) (*ps.BranchResizeRequest, error) {
	d.ResizeStatusFnInvoked = true
	return d.ResizeStatusFn(ctx, req)
}

type PostgresBranchesService struct {
	CreateFn        func(context.Context, *ps.CreatePostgresBranchRequest) (*ps.PostgresBranch, error)
	CreateFnInvoked bool

	ListFn        func(context.Context, *ps.ListPostgresBranchesRequest, ...ps.ListOption) ([]*ps.PostgresBranch, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetPostgresBranchRequest) (*ps.PostgresBranch, error)
	GetFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdatePostgresBranchRequest) (*ps.PostgresBranch, error)
	UpdateFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeletePostgresBranchRequest) error
	DeleteFnInvoked bool

	SchemaFn        func(context.Context, *ps.PostgresBranchSchemaRequest) ([]*ps.PostgresBranchSchema, error)
	SchemaFnInvoked bool

	ListClusterSKUsFn        func(context.Context, *ps.ListBranchClusterSKUsRequest, ...ps.ListOption) ([]*ps.ClusterSKU, error)
	ListClusterSKUsFnInvoked bool

	ResizeFn        func(context.Context, *ps.ResizePostgresBranchRequest) (*ps.PostgresBranchClusterResizeRequest, error)
	ResizeFnInvoked bool

	ListChangesFn        func(context.Context, *ps.ListPostgresBranchChangesRequest) ([]*ps.PostgresBranchClusterResizeRequest, error)
	ListChangesFnInvoked bool

	GetChangeFn        func(context.Context, *ps.GetPostgresBranchChangeRequest) (*ps.PostgresBranchClusterResizeRequest, error)
	GetChangeFnInvoked bool

	CancelChangesFn        func(context.Context, *ps.CancelPostgresBranchChangesRequest) error
	CancelChangesFnInvoked bool

	ListParametersFn        func(context.Context, *ps.ListPostgresParametersRequest) ([]*ps.PostgresParameter, error)
	ListParametersFnInvoked bool
}

func (p *PostgresBranchesService) Create(ctx context.Context, req *ps.CreatePostgresBranchRequest) (*ps.PostgresBranch, error) {
	p.CreateFnInvoked = true
	return p.CreateFn(ctx, req)
}

func (p *PostgresBranchesService) List(ctx context.Context, req *ps.ListPostgresBranchesRequest, opts ...ps.ListOption) ([]*ps.PostgresBranch, error) {
	p.ListFnInvoked = true
	return p.ListFn(ctx, req, opts...)
}

func (p *PostgresBranchesService) Get(ctx context.Context, req *ps.GetPostgresBranchRequest) (*ps.PostgresBranch, error) {
	p.GetFnInvoked = true
	return p.GetFn(ctx, req)
}

func (p *PostgresBranchesService) Update(ctx context.Context, req *ps.UpdatePostgresBranchRequest) (*ps.PostgresBranch, error) {
	p.UpdateFnInvoked = true
	return p.UpdateFn(ctx, req)
}

func (p *PostgresBranchesService) Delete(ctx context.Context, req *ps.DeletePostgresBranchRequest) error {
	p.DeleteFnInvoked = true
	return p.DeleteFn(ctx, req)
}

func (p *PostgresBranchesService) Schema(ctx context.Context, req *ps.PostgresBranchSchemaRequest) ([]*ps.PostgresBranchSchema, error) {
	p.SchemaFnInvoked = true
	return p.SchemaFn(ctx, req)
}

func (p *PostgresBranchesService) ListClusterSKUs(ctx context.Context, req *ps.ListBranchClusterSKUsRequest, opts ...ps.ListOption) ([]*ps.ClusterSKU, error) {
	p.ListClusterSKUsFnInvoked = true
	return p.ListClusterSKUsFn(ctx, req, opts...)
}

func (p *PostgresBranchesService) Resize(ctx context.Context, req *ps.ResizePostgresBranchRequest) (*ps.PostgresBranchClusterResizeRequest, error) {
	p.ResizeFnInvoked = true
	return p.ResizeFn(ctx, req)
}

func (p *PostgresBranchesService) ListChanges(ctx context.Context, req *ps.ListPostgresBranchChangesRequest) ([]*ps.PostgresBranchClusterResizeRequest, error) {
	p.ListChangesFnInvoked = true
	return p.ListChangesFn(ctx, req)
}

func (p *PostgresBranchesService) GetChange(ctx context.Context, req *ps.GetPostgresBranchChangeRequest) (*ps.PostgresBranchClusterResizeRequest, error) {
	p.GetChangeFnInvoked = true
	return p.GetChangeFn(ctx, req)
}

func (p *PostgresBranchesService) CancelChanges(ctx context.Context, req *ps.CancelPostgresBranchChangesRequest) error {
	p.CancelChangesFnInvoked = true
	return p.CancelChangesFn(ctx, req)
}

func (p *PostgresBranchesService) ListParameters(ctx context.Context, req *ps.ListPostgresParametersRequest) ([]*ps.PostgresParameter, error) {
	p.ListParametersFnInvoked = true
	return p.ListParametersFn(ctx, req)
}
