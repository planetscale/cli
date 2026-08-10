package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type PostgresBouncersService struct {
	ListFn        func(context.Context, *ps.ListPostgresBouncersRequest, ...ps.ListOption) ([]*ps.PostgresBouncer, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetPostgresBouncerRequest) (*ps.PostgresBouncer, error)
	GetFnInvoked bool

	CreateFn        func(context.Context, *ps.CreatePostgresBouncerRequest) (*ps.PostgresBouncer, error)
	CreateFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeletePostgresBouncerRequest) error
	DeleteFnInvoked bool
}

func (s *PostgresBouncersService) List(ctx context.Context, req *ps.ListPostgresBouncersRequest, opts ...ps.ListOption) ([]*ps.PostgresBouncer, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req, opts...)
}

func (s *PostgresBouncersService) Get(ctx context.Context, req *ps.GetPostgresBouncerRequest) (*ps.PostgresBouncer, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *PostgresBouncersService) Create(ctx context.Context, req *ps.CreatePostgresBouncerRequest) (*ps.PostgresBouncer, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *PostgresBouncersService) Delete(ctx context.Context, req *ps.DeletePostgresBouncerRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}
