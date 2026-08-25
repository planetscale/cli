package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type PostgresSwitchoversService struct {
	ListFn          func(context.Context, *ps.ListPostgresSwitchoversRequest, ...ps.ListOption) ([]*ps.PostgresSwitchover, error)
	ListFnInvoked   bool
	GetFn           func(context.Context, *ps.GetPostgresSwitchoverRequest) (*ps.PostgresSwitchover, error)
	GetFnInvoked    bool
	CreateFn        func(context.Context, *ps.CreatePostgresSwitchoverRequest) (*ps.PostgresSwitchover, error)
	CreateFnInvoked bool
}

func (s *PostgresSwitchoversService) List(ctx context.Context, req *ps.ListPostgresSwitchoversRequest, opts ...ps.ListOption) ([]*ps.PostgresSwitchover, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req, opts...)
}

func (s *PostgresSwitchoversService) Get(ctx context.Context, req *ps.GetPostgresSwitchoverRequest) (*ps.PostgresSwitchover, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *PostgresSwitchoversService) Create(ctx context.Context, req *ps.CreatePostgresSwitchoverRequest) (*ps.PostgresSwitchover, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}
