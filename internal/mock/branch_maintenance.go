package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type BranchMaintenanceService struct {
	RunFn        func(context.Context, *ps.RunBranchMaintenanceRequest) error
	RunFnInvoked bool
}

func (s *BranchMaintenanceService) Run(ctx context.Context, req *ps.RunBranchMaintenanceRequest) error {
	s.RunFnInvoked = true
	return s.RunFn(ctx, req)
}
