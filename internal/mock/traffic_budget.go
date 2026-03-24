package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type TrafficBudgetsService struct {
	ListFn          func(context.Context, *ps.ListTrafficBudgetsRequest) ([]*ps.TrafficBudget, error)
	ListFnInvoked   bool
	GetFn           func(context.Context, *ps.GetTrafficBudgetRequest) (*ps.TrafficBudget, error)
	GetFnInvoked    bool
	CreateFn        func(context.Context, *ps.CreateTrafficBudgetRequest) (*ps.TrafficBudget, error)
	CreateFnInvoked bool
	UpdateFn        func(context.Context, *ps.UpdateTrafficBudgetRequest) (*ps.TrafficBudget, error)
	UpdateFnInvoked bool
	DeleteFn        func(context.Context, *ps.DeleteTrafficBudgetRequest) error
	DeleteFnInvoked bool
}

func (s *TrafficBudgetsService) List(ctx context.Context, req *ps.ListTrafficBudgetsRequest) ([]*ps.TrafficBudget, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
}

func (s *TrafficBudgetsService) Get(ctx context.Context, req *ps.GetTrafficBudgetRequest) (*ps.TrafficBudget, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *TrafficBudgetsService) Create(ctx context.Context, req *ps.CreateTrafficBudgetRequest) (*ps.TrafficBudget, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *TrafficBudgetsService) Update(ctx context.Context, req *ps.UpdateTrafficBudgetRequest) (*ps.TrafficBudget, error) {
	s.UpdateFnInvoked = true
	return s.UpdateFn(ctx, req)
}

func (s *TrafficBudgetsService) Delete(ctx context.Context, req *ps.DeleteTrafficBudgetRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}
