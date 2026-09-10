package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type NekiAdminsService struct {
	GetFn        func(context.Context, *ps.GetNekiAdminRequest) (*ps.NekiAdmin, error)
	GetFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdateNekiAdminRequest) (*ps.NekiAdmin, error)
	UpdateFnInvoked bool

	ListParametersFn        func(context.Context, *ps.ListNekiAdminParametersRequest) ([]*ps.NekiParameter, error)
	ListParametersFnInvoked bool

	ListSizeSKUsFn        func(context.Context, *ps.ListNekiAdminSizeSKUsRequest) ([]*ps.NekiAdminSKU, error)
	ListSizeSKUsFnInvoked bool

	ListChangesFn        func(context.Context, *ps.ListNekiAdminChangesRequest) ([]*ps.NekiAdminChange, error)
	ListChangesFnInvoked bool

	GetChangeFn        func(context.Context, *ps.GetNekiAdminChangeRequest) (*ps.NekiAdminChange, error)
	GetChangeFnInvoked bool

	CancelChangeFn        func(context.Context, *ps.CancelNekiAdminChangeRequest) error
	CancelChangeFnInvoked bool
}

func (s *NekiAdminsService) Get(ctx context.Context, req *ps.GetNekiAdminRequest) (*ps.NekiAdmin, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *NekiAdminsService) Update(ctx context.Context, req *ps.UpdateNekiAdminRequest) (*ps.NekiAdmin, error) {
	s.UpdateFnInvoked = true
	return s.UpdateFn(ctx, req)
}

func (s *NekiAdminsService) ListParameters(ctx context.Context, req *ps.ListNekiAdminParametersRequest) ([]*ps.NekiParameter, error) {
	s.ListParametersFnInvoked = true
	return s.ListParametersFn(ctx, req)
}

func (s *NekiAdminsService) ListSizeSKUs(ctx context.Context, req *ps.ListNekiAdminSizeSKUsRequest) ([]*ps.NekiAdminSKU, error) {
	s.ListSizeSKUsFnInvoked = true
	return s.ListSizeSKUsFn(ctx, req)
}

func (s *NekiAdminsService) ListChanges(ctx context.Context, req *ps.ListNekiAdminChangesRequest) ([]*ps.NekiAdminChange, error) {
	s.ListChangesFnInvoked = true
	return s.ListChangesFn(ctx, req)
}

func (s *NekiAdminsService) GetChange(ctx context.Context, req *ps.GetNekiAdminChangeRequest) (*ps.NekiAdminChange, error) {
	s.GetChangeFnInvoked = true
	return s.GetChangeFn(ctx, req)
}

func (s *NekiAdminsService) CancelChange(ctx context.Context, req *ps.CancelNekiAdminChangeRequest) error {
	s.CancelChangeFnInvoked = true
	return s.CancelChangeFn(ctx, req)
}
