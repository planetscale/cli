package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type ProcesslistService struct {
	ListFn        func(context.Context, *ps.ProcesslistRequest) (*ps.ProcesslistResult, error)
	ListFnInvoked bool

	KillFn        func(context.Context, *ps.KillProcessRequest) (*ps.KillProcessResult, error)
	KillFnInvoked bool
}

func (s *ProcesslistService) List(ctx context.Context, req *ps.ProcesslistRequest) (*ps.ProcesslistResult, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
}

func (s *ProcesslistService) Kill(ctx context.Context, req *ps.KillProcessRequest) (*ps.KillProcessResult, error) {
	s.KillFnInvoked = true
	return s.KillFn(ctx, req)
}
