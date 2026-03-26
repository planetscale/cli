package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type TrafficRulesService struct {
	CreateFn        func(context.Context, *ps.CreateTrafficRuleRequest) (*ps.TrafficRule, error)
	CreateFnInvoked bool
	DeleteFn        func(context.Context, *ps.DeleteTrafficRuleRequest) error
	DeleteFnInvoked bool
}

func (s *TrafficRulesService) Create(ctx context.Context, req *ps.CreateTrafficRuleRequest) (*ps.TrafficRule, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *TrafficRulesService) Delete(ctx context.Context, req *ps.DeleteTrafficRuleRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}
