package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type ReadOnlyReplicasService struct {
	ListFn               func(context.Context, *ps.ListReadOnlyReplicasRequest) ([]*ps.ReadOnlyReplica, error)
	ListFnInvoked        bool
	CreateFn             func(context.Context, *ps.CreateReadOnlyReplicaRequest) (*ps.ReadOnlyReplica, error)
	CreateFnInvoked      bool
	GetFn                func(context.Context, *ps.GetReadOnlyReplicaRequest) (*ps.ReadOnlyReplica, error)
	GetFnInvoked         bool
	DeleteFn             func(context.Context, *ps.DeleteReadOnlyReplicaRequest) error
	DeleteFnInvoked      bool
	ListChangesFn        func(context.Context, *ps.ListReadOnlyReplicaChangesRequest, ...ps.ListOption) ([]*ps.ReadOnlyReplicaChangeRequest, error)
	ListChangesFnInvoked bool
}

func (s *ReadOnlyReplicasService) List(ctx context.Context, req *ps.ListReadOnlyReplicasRequest) ([]*ps.ReadOnlyReplica, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
}

func (s *ReadOnlyReplicasService) Create(ctx context.Context, req *ps.CreateReadOnlyReplicaRequest) (*ps.ReadOnlyReplica, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *ReadOnlyReplicasService) Get(ctx context.Context, req *ps.GetReadOnlyReplicaRequest) (*ps.ReadOnlyReplica, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *ReadOnlyReplicasService) Delete(ctx context.Context, req *ps.DeleteReadOnlyReplicaRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}

func (s *ReadOnlyReplicasService) ListChanges(ctx context.Context, req *ps.ListReadOnlyReplicaChangesRequest, opts ...ps.ListOption) ([]*ps.ReadOnlyReplicaChangeRequest, error) {
	s.ListChangesFnInvoked = true
	return s.ListChangesFn(ctx, req, opts...)
}
