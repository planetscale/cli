package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type KeyspacesService struct {
	ListFn        func(context.Context, *ps.ListKeyspacesRequest) ([]*ps.Keyspace, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetKeyspaceRequest) (*ps.Keyspace, error)
	GetFnInvoked bool

	VSchemaFn        func(context.Context, *ps.GetKeyspaceVSchemaRequest) (*ps.VSchema, error)
	VSchemaFnInvoked bool

	UpdateVSchemaFn        func(context.Context, *ps.UpdateKeyspaceVSchemaRequest) (*ps.VSchema, error)
	UpdateVSchemaFnInvoked bool

	CreateFn        func(context.Context, *ps.CreateKeyspaceRequest) (*ps.Keyspace, error)
	CreateFnInvoked bool

	ResizeFn        func(context.Context, *ps.ResizeKeyspaceRequest) (*ps.KeyspaceResizeRequest, error)
	ResizeFnInvoked bool

	CancelResizeFn        func(context.Context, *ps.CancelKeyspaceResizeRequest) error
	CancelResizeFnInvoked bool

	ResizeStatusFn        func(context.Context, *ps.KeyspaceResizeStatusRequest) (*ps.KeyspaceResizeRequest, error)
	ResizeStatusFnInvoked bool

	RolloutStatusFn        func(context.Context, *ps.KeyspaceRolloutStatusRequest) (*ps.KeyspaceRollout, error)
	RolloutStatusFnInvoked bool
}

func (s *KeyspacesService) List(ctx context.Context, req *ps.ListKeyspacesRequest) ([]*ps.Keyspace, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
}

func (s *KeyspacesService) Get(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
	s.GetFnInvoked = true
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *KeyspacesService) VSchema(ctx context.Context, req *ps.GetKeyspaceVSchemaRequest) (*ps.VSchema, error) {
	s.VSchemaFnInvoked = true
	return s.VSchemaFn(ctx, req)
}

func (s *KeyspacesService) UpdateVSchema(ctx context.Context, req *ps.UpdateKeyspaceVSchemaRequest) (*ps.VSchema, error) {
	s.UpdateVSchemaFnInvoked = true
	return s.UpdateVSchemaFn(ctx, req)
}

func (s *KeyspacesService) Create(ctx context.Context, req *ps.CreateKeyspaceRequest) (*ps.Keyspace, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *KeyspacesService) Resize(ctx context.Context, req *ps.ResizeKeyspaceRequest) (*ps.KeyspaceResizeRequest, error) {
	s.ResizeFnInvoked = true
	return s.ResizeFn(ctx, req)
}

func (s *KeyspacesService) CancelResize(ctx context.Context, req *ps.CancelKeyspaceResizeRequest) error {
	s.CancelResizeFnInvoked = true
	return s.CancelResizeFn(ctx, req)
}

func (s *KeyspacesService) ResizeStatus(ctx context.Context, req *ps.KeyspaceResizeStatusRequest) (*ps.KeyspaceResizeRequest, error) {
	s.ResizeStatusFnInvoked = true
	return s.ResizeStatusFn(ctx, req)
}

func (s *KeyspacesService) RolloutStatus(ctx context.Context, req *ps.KeyspaceRolloutStatusRequest) (*ps.KeyspaceRollout, error) {
	s.RolloutStatusFnInvoked = true
	return s.RolloutStatusFn(ctx, req)
}
