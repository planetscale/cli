package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type ReadOnlyRegionsService struct {
	ListFn        func(context.Context, *ps.ListReadOnlyRegionsRequest, ...ps.ListOption) ([]*ps.ReadOnlyRegion, error)
	ListFnInvoked bool
}

func (s *ReadOnlyRegionsService) List(ctx context.Context, req *ps.ListReadOnlyRegionsRequest, opts ...ps.ListOption) ([]*ps.ReadOnlyRegion, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req, opts...)
}
