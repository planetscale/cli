package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type OIDCTrustsService struct {
	ListProvidersFn        func(context.Context, *ps.ListOIDCTrustProvidersRequest) ([]*ps.OIDCTrustProvider, error)
	ListProvidersFnInvoked bool
}

func (s *OIDCTrustsService) ListProviders(ctx context.Context, req *ps.ListOIDCTrustProvidersRequest) ([]*ps.OIDCTrustProvider, error) {
	s.ListProvidersFnInvoked = true
	return s.ListProvidersFn(ctx, req)
}
