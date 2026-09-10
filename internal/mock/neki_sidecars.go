package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type NekiSidecarsService struct {
	ListFn        func(context.Context, *ps.ListNekiSidecarsRequest) ([]*ps.NekiSidecar, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetNekiSidecarRequest) (*ps.NekiSidecar, error)
	GetFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdateNekiSidecarRequest) (*ps.NekiSidecar, error)
	UpdateFnInvoked bool

	ListParametersFn        func(context.Context, *ps.ListNekiSidecarParametersRequest) ([]*ps.NekiParameter, error)
	ListParametersFnInvoked bool

	ListChangesFn        func(context.Context, *ps.ListNekiSidecarChangesRequest) ([]*ps.NekiSidecarChange, error)
	ListChangesFnInvoked bool

	GetChangeFn        func(context.Context, *ps.GetNekiSidecarChangeRequest) (*ps.NekiSidecarChange, error)
	GetChangeFnInvoked bool

	CancelChangeFn        func(context.Context, *ps.CancelNekiSidecarChangeRequest) error
	CancelChangeFnInvoked bool
}

func (s *NekiSidecarsService) List(ctx context.Context, req *ps.ListNekiSidecarsRequest) ([]*ps.NekiSidecar, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
}

func (s *NekiSidecarsService) Get(ctx context.Context, req *ps.GetNekiSidecarRequest) (*ps.NekiSidecar, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *NekiSidecarsService) Update(ctx context.Context, req *ps.UpdateNekiSidecarRequest) (*ps.NekiSidecar, error) {
	s.UpdateFnInvoked = true
	return s.UpdateFn(ctx, req)
}

func (s *NekiSidecarsService) ListParameters(ctx context.Context, req *ps.ListNekiSidecarParametersRequest) ([]*ps.NekiParameter, error) {
	s.ListParametersFnInvoked = true
	return s.ListParametersFn(ctx, req)
}

func (s *NekiSidecarsService) ListChanges(ctx context.Context, req *ps.ListNekiSidecarChangesRequest) ([]*ps.NekiSidecarChange, error) {
	s.ListChangesFnInvoked = true
	return s.ListChangesFn(ctx, req)
}

func (s *NekiSidecarsService) GetChange(ctx context.Context, req *ps.GetNekiSidecarChangeRequest) (*ps.NekiSidecarChange, error) {
	s.GetChangeFnInvoked = true
	return s.GetChangeFn(ctx, req)
}

func (s *NekiSidecarsService) CancelChange(ctx context.Context, req *ps.CancelNekiSidecarChangeRequest) error {
	s.CancelChangeFnInvoked = true
	return s.CancelChangeFn(ctx, req)
}
