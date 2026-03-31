package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type PlannedReparentShardService struct {
	CreateFn        func(context.Context, *ps.PlannedReparentShardRequest) (*ps.VtctldOperation, error)
	CreateFnInvoked bool

	GetFn        func(context.Context, *ps.GetPlannedReparentShardRequest) (*ps.VtctldOperation, error)
	GetFnInvoked bool
}

func (s *PlannedReparentShardService) Create(ctx context.Context, req *ps.PlannedReparentShardRequest) (*ps.VtctldOperation, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *PlannedReparentShardService) Get(ctx context.Context, req *ps.GetPlannedReparentShardRequest) (*ps.VtctldOperation, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}
