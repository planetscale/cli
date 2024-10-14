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

	CreateFn        func(context.Context, *ps.CreateBranchKeyspaceRequest) (*ps.Keyspace, error)
	CreateFnInvoked bool

	ResizeFn        func(context.Context, *ps.ResizeKeyspaceRequest) (*ps.KeyspaceResizeRequest, error)
	ResizeFnInvoked bool

	CancelResizeFn        func(context.Context, *ps.CancelKeyspaceResizeRequest) error
	CancelResizeFnInvoked bool

	ResizeStatusFn        func(context.Context, *ps.KeyspaceResizeStatusRequest) (*ps.KeyspaceResizeRequest, error)
	ResizeStatusFnInvoked bool
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

func (s *BranchKeyspacesService) Create(ctx context.Context, req *ps.CreateBranchKeyspaceRequest) (*ps.Keyspace, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *BranchKeyspacesService) Resize(ctx context.Context, req *ps.ResizeKeyspaceRequest) (*ps.KeyspaceResizeRequest, error) {
	s.ResizeFnInvoked = true
	return s.ResizeFn(ctx, req)
}

func (s *BranchKeyspacesService) CancelResize(ctx context.Context, req *ps.CancelKeyspaceResizeRequest) error {
	s.CancelResizeFnInvoked = true
	return s.CancelResizeFn(ctx, req)
}

func (s *BranchKeyspacesService) ResizeStatus(ctx context.Context, req *ps.KeyspaceResizeStatusRequest) (*ps.KeyspaceResizeRequest, error) {
	s.ResizeStatusFnInvoked = true
	return s.ResizeStatusFn(ctx, req)
}
