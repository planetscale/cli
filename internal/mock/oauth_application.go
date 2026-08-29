package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type OAuthApplicationsService struct {
	ListFn        func(context.Context, *ps.ListOAuthApplicationsRequest, ...ps.ListOption) ([]*ps.OAuthApplication, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetOAuthApplicationRequest) (*ps.OAuthApplication, error)
	GetFnInvoked bool

	ListTokensFn        func(context.Context, *ps.ListOAuthTokensRequest, ...ps.ListOption) ([]*ps.OAuthToken, error)
	ListTokensFnInvoked bool

	GetTokenFn        func(context.Context, *ps.GetOAuthTokenRequest) (*ps.OAuthToken, error)
	GetTokenFnInvoked bool

	DeleteTokenFn        func(context.Context, *ps.DeleteOAuthTokenRequest) error
	DeleteTokenFnInvoked bool

	CreateTokenFn        func(context.Context, *ps.CreateOAuthTokenRequest) (*ps.OAuthToken, error)
	CreateTokenFnInvoked bool
}

func (s *OAuthApplicationsService) List(ctx context.Context, req *ps.ListOAuthApplicationsRequest, opts ...ps.ListOption) ([]*ps.OAuthApplication, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req, opts...)
}

func (s *OAuthApplicationsService) Get(ctx context.Context, req *ps.GetOAuthApplicationRequest) (*ps.OAuthApplication, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *OAuthApplicationsService) ListTokens(ctx context.Context, req *ps.ListOAuthTokensRequest, opts ...ps.ListOption) ([]*ps.OAuthToken, error) {
	s.ListTokensFnInvoked = true
	return s.ListTokensFn(ctx, req, opts...)
}

func (s *OAuthApplicationsService) GetToken(ctx context.Context, req *ps.GetOAuthTokenRequest) (*ps.OAuthToken, error) {
	s.GetTokenFnInvoked = true
	return s.GetTokenFn(ctx, req)
}

func (s *OAuthApplicationsService) DeleteToken(ctx context.Context, req *ps.DeleteOAuthTokenRequest) error {
	s.DeleteTokenFnInvoked = true
	return s.DeleteTokenFn(ctx, req)
}

func (s *OAuthApplicationsService) CreateToken(ctx context.Context, req *ps.CreateOAuthTokenRequest) (*ps.OAuthToken, error) {
	s.CreateTokenFnInvoked = true
	return s.CreateTokenFn(ctx, req)
}
