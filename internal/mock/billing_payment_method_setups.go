package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type BillingPaymentMethodSetupsService struct {
	CreateFn        func(context.Context, *ps.CreateBillingPaymentMethodSetupRequest) (*ps.BillingPaymentMethodSetup, error)
	CreateFnInvoked bool
	GetFn           func(context.Context, *ps.GetBillingPaymentMethodSetupRequest) (*ps.BillingPaymentMethodSetup, error)
	GetFnInvoked    bool
}

func (s *BillingPaymentMethodSetupsService) Create(ctx context.Context, req *ps.CreateBillingPaymentMethodSetupRequest) (*ps.BillingPaymentMethodSetup, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *BillingPaymentMethodSetupsService) Get(ctx context.Context, req *ps.GetBillingPaymentMethodSetupRequest) (*ps.BillingPaymentMethodSetup, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}
