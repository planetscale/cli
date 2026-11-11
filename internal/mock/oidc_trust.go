package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type OIDCTrustsService struct {
	ListProvidersFn        func(context.Context, *ps.ListOIDCTrustProvidersRequest) ([]*ps.OIDCTrustProvider, error)
	ListProvidersFnInvoked bool

	ListFn        func(context.Context, *ps.ListOIDCTrustsRequest, ...ps.ListOption) ([]*ps.OIDCTrust, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetOIDCTrustRequest) (*ps.OIDCTrust, error)
	GetFnInvoked bool

	CreateFn        func(context.Context, *ps.CreateOIDCTrustRequest) (*ps.OIDCTrust, error)
	CreateFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdateOIDCTrustRequest) (*ps.OIDCTrust, error)
	UpdateFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeleteOIDCTrustRequest) error
	DeleteFnInvoked bool
}

func (s *OIDCTrustsService) ListProviders(ctx context.Context, req *ps.ListOIDCTrustProvidersRequest) ([]*ps.OIDCTrustProvider, error) {
	s.ListProvidersFnInvoked = true
	return s.ListProvidersFn(ctx, req)
}

func (s *OIDCTrustsService) List(ctx context.Context, req *ps.ListOIDCTrustsRequest, opts ...ps.ListOption) ([]*ps.OIDCTrust, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req, opts...)
}

func (s *OIDCTrustsService) Get(ctx context.Context, req *ps.GetOIDCTrustRequest) (*ps.OIDCTrust, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *OIDCTrustsService) Create(ctx context.Context, req *ps.CreateOIDCTrustRequest) (*ps.OIDCTrust, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *OIDCTrustsService) Update(ctx context.Context, req *ps.UpdateOIDCTrustRequest) (*ps.OIDCTrust, error) {
	s.UpdateFnInvoked = true
	return s.UpdateFn(ctx, req)
}

func (s *OIDCTrustsService) Delete(ctx context.Context, req *ps.DeleteOIDCTrustRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}
