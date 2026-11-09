package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type OIDCTrustsService struct {
	ListFn        func(context.Context, *ps.ListOIDCTrustsRequest, ...ps.ListOption) ([]*ps.OIDCTrust, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetOIDCTrustRequest) (*ps.OIDCTrust, error)
	GetFnInvoked bool

	ListProvidersFn        func(context.Context, *ps.ListOIDCTrustProvidersRequest) ([]*ps.OIDCTrustProvider, error)
	ListProvidersFnInvoked bool
}

func (o *OIDCTrustsService) List(ctx context.Context, req *ps.ListOIDCTrustsRequest, opts ...ps.ListOption) ([]*ps.OIDCTrust, error) {
	o.ListFnInvoked = true
	return o.ListFn(ctx, req, opts...)
}

func (o *OIDCTrustsService) Get(ctx context.Context, req *ps.GetOIDCTrustRequest) (*ps.OIDCTrust, error) {
	o.GetFnInvoked = true
	return o.GetFn(ctx, req)
}

func (o *OIDCTrustsService) ListProviders(ctx context.Context, req *ps.ListOIDCTrustProvidersRequest) ([]*ps.OIDCTrustProvider, error) {
	o.ListProvidersFnInvoked = true
	return o.ListProvidersFn(ctx, req)
}
