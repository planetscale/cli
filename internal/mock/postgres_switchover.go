package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type PostgresSwitchoversService struct {
	CreateFn        func(context.Context, *ps.CreatePostgresSwitchoverRequest) (*ps.PostgresSwitchover, error)
	CreateFnInvoked bool
}

func (s *PostgresSwitchoversService) Create(ctx context.Context, req *ps.CreatePostgresSwitchoverRequest) (*ps.PostgresSwitchover, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}
