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

	ListResizesFn        func(context.Context, *ps.ListPostgresBouncerResizesRequest, ...ps.ListOption) ([]*ps.PostgresBouncerResizeRequest, error)
	ListResizesFnInvoked bool

	ResizeFn        func(context.Context, *ps.ResizePostgresBouncerRequest) (*ps.PostgresBouncerResizeRequest, error)
	ResizeFnInvoked bool

	CancelResizesFn        func(context.Context, *ps.CancelPostgresBouncerResizesRequest) error
	CancelResizesFnInvoked bool

	CreateRestartFn        func(context.Context, *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error)
	CreateRestartFnInvoked bool

	ListRestartsFn        func(context.Context, *ps.ListPostgresBouncerRestartsRequest, ...ps.ListOption) ([]*ps.PostgresBouncerRestart, error)
	ListRestartsFnInvoked bool

	GetRestartFn        func(context.Context, *ps.GetPostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error)
	GetRestartFnInvoked bool
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

func (s *PostgresBouncersService) ListResizes(ctx context.Context, req *ps.ListPostgresBouncerResizesRequest, opts ...ps.ListOption) ([]*ps.PostgresBouncerResizeRequest, error) {
	s.ListResizesFnInvoked = true
	return s.ListResizesFn(ctx, req, opts...)
}

func (s *PostgresBouncersService) Resize(ctx context.Context, req *ps.ResizePostgresBouncerRequest) (*ps.PostgresBouncerResizeRequest, error) {
	s.ResizeFnInvoked = true
	return s.ResizeFn(ctx, req)
}

func (s *PostgresBouncersService) CancelResizes(ctx context.Context, req *ps.CancelPostgresBouncerResizesRequest) error {
	s.CancelResizesFnInvoked = true
	return s.CancelResizesFn(ctx, req)
}

func (s *PostgresBouncersService) CreateRestart(ctx context.Context, req *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
	s.CreateRestartFnInvoked = true
	return s.CreateRestartFn(ctx, req)
}

func (s *PostgresBouncersService) ListRestarts(ctx context.Context, req *ps.ListPostgresBouncerRestartsRequest, opts ...ps.ListOption) ([]*ps.PostgresBouncerRestart, error) {
	s.ListRestartsFnInvoked = true
	return s.ListRestartsFn(ctx, req, opts...)
}

func (s *PostgresBouncersService) GetRestart(ctx context.Context, req *ps.GetPostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
	s.GetRestartFnInvoked = true
	return s.GetRestartFn(ctx, req)
}
