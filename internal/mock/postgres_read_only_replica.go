package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type PostgresDedicatedReadReplicasService struct {
	ListFn        func(context.Context, *ps.ListPostgresDedicatedReadReplicasRequest) ([]*ps.PostgresDedicatedReadReplica, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetPostgresDedicatedReadReplicaRequest) (*ps.PostgresDedicatedReadReplica, error)
	GetFnInvoked bool

	CreateFn        func(context.Context, *ps.CreatePostgresDedicatedReadReplicaRequest) (*ps.PostgresDedicatedReadReplica, error)
	CreateFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdatePostgresDedicatedReadReplicaRequest) (*ps.PostgresDedicatedReadReplica, error)
	UpdateFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeletePostgresDedicatedReadReplicaRequest) error
	DeleteFnInvoked bool
}

// PostgresReadOnlyReplicasService is the former name for PostgresDedicatedReadReplicasService.
// Deprecated: use PostgresDedicatedReadReplicasService.
type PostgresReadOnlyReplicasService = PostgresDedicatedReadReplicasService

func (s *PostgresDedicatedReadReplicasService) List(ctx context.Context, req *ps.ListPostgresDedicatedReadReplicasRequest) ([]*ps.PostgresDedicatedReadReplica, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
}

func (s *PostgresDedicatedReadReplicasService) Get(ctx context.Context, req *ps.GetPostgresDedicatedReadReplicaRequest) (*ps.PostgresDedicatedReadReplica, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *PostgresDedicatedReadReplicasService) Create(ctx context.Context, req *ps.CreatePostgresDedicatedReadReplicaRequest) (*ps.PostgresDedicatedReadReplica, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *PostgresDedicatedReadReplicasService) Update(ctx context.Context, req *ps.UpdatePostgresDedicatedReadReplicaRequest) (*ps.PostgresDedicatedReadReplica, error) {
	s.UpdateFnInvoked = true
	return s.UpdateFn(ctx, req)
}

func (s *PostgresDedicatedReadReplicasService) Delete(ctx context.Context, req *ps.DeletePostgresDedicatedReadReplicaRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}
