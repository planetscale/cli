package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type BranchKeyspacesService struct {
	ListFn        func(context.Context, *ps.ListBranchKeyspacesRequest) ([]*ps.Keyspace, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetBranchKeyspaceRequest) (*ps.Keyspace, error)
	GetFnInvoked bool

	VSchemaFn        func(context.Context, *ps.GetKeyspaceVSchemaRequest) (*ps.VSchema, error)
	VSchemaFnInvoked bool

	UpdateVSchemaFn        func(context.Context, *ps.UpdateKeyspaceVSchemaRequest) (*ps.VSchema, error)
	UpdateVSchemaFnInvoked bool
}

func (s *BranchKeyspacesService) List(ctx context.Context, req *ps.ListBranchKeyspacesRequest) ([]*ps.Keyspace, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
}

func (s *BranchKeyspacesService) Get(ctx context.Context, req *ps.GetBranchKeyspaceRequest) (*ps.Keyspace, error) {
	s.GetFnInvoked = true
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *BranchKeyspacesService) VSchema(ctx context.Context, req *ps.GetKeyspaceVSchemaRequest) (*ps.VSchema, error) {
	s.VSchemaFnInvoked = true
	return s.VSchemaFn(ctx, req)
}

func (s *BranchKeyspacesService) UpdateVSchema(ctx context.Context, req *ps.UpdateKeyspaceVSchemaRequest) (*ps.VSchema, error) {
	s.UpdateVSchemaFnInvoked = true
	return s.UpdateVSchemaFn(ctx, req)
}
