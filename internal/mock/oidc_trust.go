package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type OIDCTrustsService struct {
	ListProvidersFn        func(context.Context, *ps.ListOIDCTrustProvidersRequest) ([]*ps.OIDCTrustProvider, error)
	ListProvidersFnInvoked bool

	GetFn        func(context.Context, *ps.GetOIDCTrustRequest) (*ps.OIDCTrust, error)
	GetFnInvoked bool
}

func (s *OIDCTrustsService) ListProviders(ctx context.Context, req *ps.ListOIDCTrustProvidersRequest) ([]*ps.OIDCTrustProvider, error) {
	s.ListProvidersFnInvoked = true
	return s.ListProvidersFn(ctx, req)
}

func (s *OIDCTrustsService) Get(ctx context.Context, req *ps.GetOIDCTrustRequest) (*ps.OIDCTrust, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}
