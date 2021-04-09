package database

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type mockDatabaseService struct {
	createFn        func(context.Context, *ps.CreateDatabaseRequest) (*ps.Database, error)
	createFnInvoked bool

	getFn        func(context.Context, *ps.GetDatabaseRequest) (*ps.Database, error)
	getFnInvoked bool

	listFn        func(context.Context, *ps.ListDatabasesRequest) ([]*ps.Database, error)
	listFnInvoked bool

	deleteFn        func(context.Context, *ps.DeleteDatabaseRequest) error
	deleteFnInvoked bool
}

func (m *mockDatabaseService) Create(ctx context.Context, req *ps.CreateDatabaseRequest) (*ps.Database, error) {
	m.createFnInvoked = true
	return m.createFn(ctx, req)
}

func (m *mockDatabaseService) Get(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
	m.getFnInvoked = true
	return m.getFn(ctx, req)
}

func (m *mockDatabaseService) List(ctx context.Context, req *ps.ListDatabasesRequest) ([]*ps.Database, error) {
	m.listFnInvoked = true
	return m.listFn(ctx, req)
}

func (m *mockDatabaseService) Delete(ctx context.Context, req *ps.DeleteDatabaseRequest) error {
	m.deleteFnInvoked = true
	return m.deleteFn(ctx, req)
}
