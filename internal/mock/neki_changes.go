package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type NekiChangesService struct {
	ListFn        func(context.Context, *ps.ListNekiChangesRequest) ([]*ps.NekiChange, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetNekiChangeRequest) (*ps.NekiChange, error)
	GetFnInvoked bool

	CancelFn        func(context.Context, *ps.CancelNekiChangeRequest) error
	CancelFnInvoked bool
}

func (s *NekiChangesService) List(ctx context.Context, req *ps.ListNekiChangesRequest) ([]*ps.NekiChange, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
}

func (s *NekiChangesService) Get(ctx context.Context, req *ps.GetNekiChangeRequest) (*ps.NekiChange, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *NekiChangesService) Cancel(ctx context.Context, req *ps.CancelNekiChangeRequest) error {
	s.CancelFnInvoked = true
	return s.CancelFn(ctx, req)
}
