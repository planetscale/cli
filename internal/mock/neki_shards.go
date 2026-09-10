package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type NekiShardsService struct {
	ListFn        func(context.Context, *ps.ListNekiShardsRequest) ([]*ps.NekiShard, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetNekiShardRequest) (*ps.NekiShard, error)
	GetFnInvoked bool

	CreateFn        func(context.Context, *ps.CreateNekiShardsRequest) (*ps.CreateNekiShardsResponse, error)
	CreateFnInvoked bool

	AssignFn        func(context.Context, *ps.AssignNekiShardsRequest) ([]*ps.NekiShardAssignment, error)
	AssignFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdateNekiShardRequest) (*ps.NekiShard, error)
	UpdateFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeleteNekiShardRequest) error
	DeleteFnInvoked bool
}

func (s *NekiShardsService) List(ctx context.Context, req *ps.ListNekiShardsRequest) ([]*ps.NekiShard, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
}

func (s *NekiShardsService) Get(ctx context.Context, req *ps.GetNekiShardRequest) (*ps.NekiShard, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *NekiShardsService) Create(ctx context.Context, req *ps.CreateNekiShardsRequest) (*ps.CreateNekiShardsResponse, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *NekiShardsService) Assign(ctx context.Context, req *ps.AssignNekiShardsRequest) ([]*ps.NekiShardAssignment, error) {
	s.AssignFnInvoked = true
	return s.AssignFn(ctx, req)
}

func (s *NekiShardsService) Update(ctx context.Context, req *ps.UpdateNekiShardRequest) (*ps.NekiShard, error) {
	s.UpdateFnInvoked = true
	return s.UpdateFn(ctx, req)
}

func (s *NekiShardsService) Delete(ctx context.Context, req *ps.DeleteNekiShardRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}
