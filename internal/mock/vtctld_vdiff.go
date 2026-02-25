package mock

import (
	"context"
	"encoding/json"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type VDiffService struct {
	CreateFn        func(context.Context, *ps.VDiffCreateRequest) (json.RawMessage, error)
	CreateFnInvoked bool

	ShowFn        func(context.Context, *ps.VDiffShowRequest) (json.RawMessage, error)
	ShowFnInvoked bool

	StopFn        func(context.Context, *ps.VDiffStopRequest) (json.RawMessage, error)
	StopFnInvoked bool

	ResumeFn        func(context.Context, *ps.VDiffResumeRequest) (json.RawMessage, error)
	ResumeFnInvoked bool

	DeleteFn        func(context.Context, *ps.VDiffDeleteRequest) (json.RawMessage, error)
	DeleteFnInvoked bool
}

func (s *VDiffService) Create(ctx context.Context, req *ps.VDiffCreateRequest) (json.RawMessage, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *VDiffService) Show(ctx context.Context, req *ps.VDiffShowRequest) (json.RawMessage, error) {
	s.ShowFnInvoked = true
	return s.ShowFn(ctx, req)
}

func (s *VDiffService) Stop(ctx context.Context, req *ps.VDiffStopRequest) (json.RawMessage, error) {
	s.StopFnInvoked = true
	return s.StopFn(ctx, req)
}

func (s *VDiffService) Resume(ctx context.Context, req *ps.VDiffResumeRequest) (json.RawMessage, error) {
	s.ResumeFnInvoked = true
	return s.ResumeFn(ctx, req)
}

func (s *VDiffService) Delete(ctx context.Context, req *ps.VDiffDeleteRequest) (json.RawMessage, error) {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}
