package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type ServiceTokenService struct {
	CreateFn        func(context.Context, *ps.CreateServiceTokenRequest) (*ps.ServiceToken, error)
	CreateFnInvoked bool

	ListFn        func(context.Context, *ps.ListServiceTokensRequest) ([]*ps.ServiceToken, error)
	ListFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeleteServiceTokenRequest) error
	DeleteFnInvoked bool

	GetAccessFn        func(context.Context, *ps.GetServiceTokenAccessRequest) ([]*ps.ServiceTokenAccess, error)
	GetAccessFnInvoked bool

	AddAccessFn        func(context.Context, *ps.AddServiceTokenAccessRequest) ([]*ps.ServiceTokenAccess, error)
	AddAccessFnInvoked bool

	DeleteAccessFn        func(context.Context, *ps.DeleteServiceTokenAccessRequest) error
	DeleteAccessFnInvoked bool
}

func (s *ServiceTokenService) Create(ctx context.Context, req *ps.CreateServiceTokenRequest) (*ps.ServiceToken, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *ServiceTokenService) List(ctx context.Context, req *ps.ListServiceTokensRequest) ([]*ps.ServiceToken, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
}

func (s *ServiceTokenService) Delete(ctx context.Context, req *ps.DeleteServiceTokenRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}

func (s *ServiceTokenService) GetAccess(ctx context.Context, req *ps.GetServiceTokenAccessRequest) ([]*ps.ServiceTokenAccess, error) {
	s.GetAccessFnInvoked = true
	return s.GetAccessFn(ctx, req)
}

func (s *ServiceTokenService) AddAccess(ctx context.Context, req *ps.AddServiceTokenAccessRequest) ([]*ps.ServiceTokenAccess, error) {
	s.AddAccessFnInvoked = true
	return s.AddAccessFn(ctx, req)
}

func (s *ServiceTokenService) DeleteAccess(ctx context.Context, req *ps.DeleteServiceTokenAccessRequest) error {
	s.DeleteAccessFnInvoked = true
	return s.DeleteAccessFn(ctx, req)

}
