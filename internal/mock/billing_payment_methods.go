package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type BillingPaymentMethodsService struct {
	GetFn           func(context.Context, *ps.GetBillingPaymentMethodRequest) (*ps.BillingPaymentMethod, error)
	GetFnInvoked    bool
	DeleteFn        func(context.Context, *ps.DeleteBillingPaymentMethodRequest) error
	DeleteFnInvoked bool
}

func (s *BillingPaymentMethodsService) Get(ctx context.Context, req *ps.GetBillingPaymentMethodRequest) (*ps.BillingPaymentMethod, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *BillingPaymentMethodsService) Delete(ctx context.Context, req *ps.DeleteBillingPaymentMethodRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}
