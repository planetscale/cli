package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type NekiRoutersService struct {
	ListFn        func(context.Context, *ps.ListNekiRoutersRequest) ([]*ps.NekiRouter, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetNekiRouterRequest) (*ps.NekiRouter, error)
	GetFnInvoked bool

	CreateFn        func(context.Context, *ps.CreateNekiRouterRequest) (*ps.NekiRouter, error)
	CreateFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdateNekiRouterRequest) (*ps.NekiRouter, error)
	UpdateFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeleteNekiRouterRequest) error
	DeleteFnInvoked bool

	ListParametersFn        func(context.Context, *ps.ListNekiRouterParametersRequest) ([]*ps.NekiParameter, error)
	ListParametersFnInvoked bool

	ListSizeSKUsFn        func(context.Context, *ps.ListNekiRouterSizeSKUsRequest) ([]*ps.NekiRouterSKU, error)
	ListSizeSKUsFnInvoked bool

	ListChangesFn        func(context.Context, *ps.ListNekiRouterChangesRequest) ([]*ps.NekiRouterChange, error)
	ListChangesFnInvoked bool

	GetChangeFn        func(context.Context, *ps.GetNekiRouterChangeRequest) (*ps.NekiRouterChange, error)
	GetChangeFnInvoked bool

	CancelChangeFn        func(context.Context, *ps.CancelNekiRouterChangeRequest) error
	CancelChangeFnInvoked bool
}

func (s *NekiRoutersService) List(ctx context.Context, req *ps.ListNekiRoutersRequest) ([]*ps.NekiRouter, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
}

func (s *NekiRoutersService) Get(ctx context.Context, req *ps.GetNekiRouterRequest) (*ps.NekiRouter, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *NekiRoutersService) Create(ctx context.Context, req *ps.CreateNekiRouterRequest) (*ps.NekiRouter, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *NekiRoutersService) Update(ctx context.Context, req *ps.UpdateNekiRouterRequest) (*ps.NekiRouter, error) {
	s.UpdateFnInvoked = true
	return s.UpdateFn(ctx, req)
}

func (s *NekiRoutersService) Delete(ctx context.Context, req *ps.DeleteNekiRouterRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}

func (s *NekiRoutersService) ListParameters(ctx context.Context, req *ps.ListNekiRouterParametersRequest) ([]*ps.NekiParameter, error) {
	s.ListParametersFnInvoked = true
	return s.ListParametersFn(ctx, req)
}

func (s *NekiRoutersService) ListSizeSKUs(ctx context.Context, req *ps.ListNekiRouterSizeSKUsRequest) ([]*ps.NekiRouterSKU, error) {
	s.ListSizeSKUsFnInvoked = true
	return s.ListSizeSKUsFn(ctx, req)
}

func (s *NekiRoutersService) ListChanges(ctx context.Context, req *ps.ListNekiRouterChangesRequest) ([]*ps.NekiRouterChange, error) {
	s.ListChangesFnInvoked = true
	return s.ListChangesFn(ctx, req)
}

func (s *NekiRoutersService) GetChange(ctx context.Context, req *ps.GetNekiRouterChangeRequest) (*ps.NekiRouterChange, error) {
	s.GetChangeFnInvoked = true
	return s.GetChangeFn(ctx, req)
}

func (s *NekiRoutersService) CancelChange(ctx context.Context, req *ps.CancelNekiRouterChangeRequest) error {
	s.CancelChangeFnInvoked = true
	return s.CancelChangeFn(ctx, req)
}
