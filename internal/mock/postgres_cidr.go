package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type PostgresCIDRsService struct {
	ListFn        func(context.Context, *ps.ListPostgresCIDRsRequest, ...ps.ListOption) ([]*ps.PostgresCIDR, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetPostgresCIDRRequest) (*ps.PostgresCIDR, error)
	GetFnInvoked bool

	CreateFn        func(context.Context, *ps.CreatePostgresCIDRRequest) (*ps.PostgresCIDR, error)
	CreateFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdatePostgresCIDRRequest) (*ps.PostgresCIDR, error)
	UpdateFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeletePostgresCIDRRequest) error
	DeleteFnInvoked bool
}

func (s *PostgresCIDRsService) List(ctx context.Context, req *ps.ListPostgresCIDRsRequest, opts ...ps.ListOption) ([]*ps.PostgresCIDR, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req, opts...)
}

func (s *PostgresCIDRsService) Get(ctx context.Context, req *ps.GetPostgresCIDRRequest) (*ps.PostgresCIDR, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *PostgresCIDRsService) Create(ctx context.Context, req *ps.CreatePostgresCIDRRequest) (*ps.PostgresCIDR, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *PostgresCIDRsService) Update(ctx context.Context, req *ps.UpdatePostgresCIDRRequest) (*ps.PostgresCIDR, error) {
	s.UpdateFnInvoked = true
	return s.UpdateFn(ctx, req)
}

func (s *PostgresCIDRsService) Delete(ctx context.Context, req *ps.DeletePostgresCIDRRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}
