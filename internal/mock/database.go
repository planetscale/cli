package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type DatabaseService struct {
	CreateFn        func(context.Context, *ps.CreateDatabaseRequest) (*ps.Database, error)
	CreateFnInvoked bool

	GetFn        func(context.Context, *ps.GetDatabaseRequest) (*ps.Database, error)
	GetFnInvoked bool

	ListFn        func(context.Context, *ps.ListDatabasesRequest, ...ps.ListOption) ([]*ps.Database, error)
	ListFnInvoked bool

	ListRegionsFn        func(context.Context, *ps.ListDatabaseRegionsRequest, ...ps.ListOption) ([]*ps.Region, error)
	ListRegionsFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeleteDatabaseRequest) (*ps.DatabaseDeletionRequest, error)
	DeleteFnInvoked bool

	UpdateSettingsFn        func(context.Context, *ps.UpdateDatabaseSettingsRequest) (*ps.Database, error)
	UpdateSettingsFnInvoked bool

	GetThrottlerFn        func(context.Context, *ps.GetDatabaseThrottlerRequest) (*ps.DatabaseThrottler, error)
	GetThrottlerFnInvoked bool

	UpdateThrottlerFn        func(context.Context, *ps.UpdateDatabaseThrottlerRequest) (*ps.DatabaseThrottler, error)
	UpdateThrottlerFnInvoked bool

	GetAggressiveCutoverFn        func(context.Context, *ps.AggressiveCutoverRequest) (*ps.AggressiveCutover, error)
	GetAggressiveCutoverFnInvoked bool

	EnableAggressiveCutoverFn        func(context.Context, *ps.AggressiveCutoverRequest) (*ps.AggressiveCutover, error)
	EnableAggressiveCutoverFnInvoked bool

	DisableAggressiveCutoverFn        func(context.Context, *ps.AggressiveCutoverRequest) (*ps.AggressiveCutover, error)
	DisableAggressiveCutoverFnInvoked bool
}

func (d *DatabaseService) Create(ctx context.Context, req *ps.CreateDatabaseRequest) (*ps.Database, error) {
	d.CreateFnInvoked = true
	return d.CreateFn(ctx, req)
}

func (d *DatabaseService) Get(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
	d.GetFnInvoked = true
	return d.GetFn(ctx, req)
}

func (d *DatabaseService) List(ctx context.Context, req *ps.ListDatabasesRequest, opts ...ps.ListOption) ([]*ps.Database, error) {
	d.ListFnInvoked = true
	return d.ListFn(ctx, req, opts...)
}

func (d *DatabaseService) ListRegions(ctx context.Context, req *ps.ListDatabaseRegionsRequest, opts ...ps.ListOption) ([]*ps.Region, error) {
	d.ListRegionsFnInvoked = true
	return d.ListRegionsFn(ctx, req, opts...)
}

func (d *DatabaseService) Delete(ctx context.Context, req *ps.DeleteDatabaseRequest) (*ps.DatabaseDeletionRequest, error) {
	d.DeleteFnInvoked = true
	return d.DeleteFn(ctx, req)
}

func (d *DatabaseService) UpdateSettings(ctx context.Context, req *ps.UpdateDatabaseSettingsRequest) (*ps.Database, error) {
	d.UpdateSettingsFnInvoked = true
	return d.UpdateSettingsFn(ctx, req)
}

func (d *DatabaseService) GetThrottler(ctx context.Context, req *ps.GetDatabaseThrottlerRequest) (*ps.DatabaseThrottler, error) {
	d.GetThrottlerFnInvoked = true
	return d.GetThrottlerFn(ctx, req)
}

func (d *DatabaseService) UpdateThrottler(ctx context.Context, req *ps.UpdateDatabaseThrottlerRequest) (*ps.DatabaseThrottler, error) {
	d.UpdateThrottlerFnInvoked = true
	return d.UpdateThrottlerFn(ctx, req)
}

func (d *DatabaseService) GetAggressiveCutover(ctx context.Context, req *ps.AggressiveCutoverRequest) (*ps.AggressiveCutover, error) {
	d.GetAggressiveCutoverFnInvoked = true
	return d.GetAggressiveCutoverFn(ctx, req)
}

func (d *DatabaseService) EnableAggressiveCutover(ctx context.Context, req *ps.AggressiveCutoverRequest) (*ps.AggressiveCutover, error) {
	d.EnableAggressiveCutoverFnInvoked = true
	return d.EnableAggressiveCutoverFn(ctx, req)
}

func (d *DatabaseService) DisableAggressiveCutover(ctx context.Context, req *ps.AggressiveCutoverRequest) (*ps.AggressiveCutover, error) {
	d.DisableAggressiveCutoverFnInvoked = true
	return d.DisableAggressiveCutoverFn(ctx, req)
}
