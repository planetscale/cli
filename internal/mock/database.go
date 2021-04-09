package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type DatabaseService struct {
	CreateFn        func(context.Context, *ps.CreateDatabaseRequest) (*ps.Database, error)
	CreateFnInvoked bool

	GetFn        func(context.Context, *ps.GetDatabaseRequest) (*ps.Database, error)
	GetFnInvoked bool

	ListFn        func(context.Context, *ps.ListDatabasesRequest) ([]*ps.Database, error)
	ListFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeleteDatabaseRequest) error
	DeleteFnInvoked bool
}

func (d *DatabaseService) Create(ctx context.Context, req *ps.CreateDatabaseRequest) (*ps.Database, error) {
	d.CreateFnInvoked = true
	return d.CreateFn(ctx, req)
}

func (d *DatabaseService) Get(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
	d.GetFnInvoked = true
	return d.GetFn(ctx, req)
}

func (d *DatabaseService) List(ctx context.Context, req *ps.ListDatabasesRequest) ([]*ps.Database, error) {
	d.ListFnInvoked = true
	return d.ListFn(ctx, req)
}

func (d *DatabaseService) Delete(ctx context.Context, req *ps.DeleteDatabaseRequest) error {
	d.DeleteFnInvoked = true
	return d.DeleteFn(ctx, req)
}
