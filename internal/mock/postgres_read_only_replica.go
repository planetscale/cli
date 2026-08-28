package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type PostgresReadOnlyReplicasService struct {
	ListFn        func(context.Context, *ps.ListPostgresReadOnlyReplicasRequest) ([]*ps.PostgresReadOnlyReplica, error)
	ListFnInvoked bool

	CreateFn        func(context.Context, *ps.CreatePostgresReadOnlyReplicaRequest) (*ps.PostgresReadOnlyReplica, error)
	CreateFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdatePostgresReadOnlyReplicaRequest) (*ps.PostgresReadOnlyReplica, error)
	UpdateFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeletePostgresReadOnlyReplicaRequest) error
	DeleteFnInvoked bool
}

func (s *PostgresReadOnlyReplicasService) List(ctx context.Context, req *ps.ListPostgresReadOnlyReplicasRequest) ([]*ps.PostgresReadOnlyReplica, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
}

func (s *PostgresReadOnlyReplicasService) Create(ctx context.Context, req *ps.CreatePostgresReadOnlyReplicaRequest) (*ps.PostgresReadOnlyReplica, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *PostgresReadOnlyReplicasService) Update(ctx context.Context, req *ps.UpdatePostgresReadOnlyReplicaRequest) (*ps.PostgresReadOnlyReplica, error) {
	s.UpdateFnInvoked = true
	return s.UpdateFn(ctx, req)
}

func (s *PostgresReadOnlyReplicasService) Delete(ctx context.Context, req *ps.DeletePostgresReadOnlyReplicaRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}
