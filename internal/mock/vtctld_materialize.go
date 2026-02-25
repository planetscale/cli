package mock

import (
	"context"
	"encoding/json"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type MaterializeService struct {
	CreateFn        func(context.Context, *ps.MaterializeCreateRequest) (json.RawMessage, error)
	CreateFnInvoked bool

	ShowFn        func(context.Context, *ps.MaterializeShowRequest) (json.RawMessage, error)
	ShowFnInvoked bool

	StartFn        func(context.Context, *ps.MaterializeStartRequest) (json.RawMessage, error)
	StartFnInvoked bool

	StopFn        func(context.Context, *ps.MaterializeStopRequest) (json.RawMessage, error)
	StopFnInvoked bool

	CancelFn        func(context.Context, *ps.MaterializeCancelRequest) (json.RawMessage, error)
	CancelFnInvoked bool
}

func (s *MaterializeService) Create(ctx context.Context, req *ps.MaterializeCreateRequest) (json.RawMessage, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *MaterializeService) Show(ctx context.Context, req *ps.MaterializeShowRequest) (json.RawMessage, error) {
	s.ShowFnInvoked = true
	return s.ShowFn(ctx, req)
}

func (s *MaterializeService) Start(ctx context.Context, req *ps.MaterializeStartRequest) (json.RawMessage, error) {
	s.StartFnInvoked = true
	return s.StartFn(ctx, req)
}

func (s *MaterializeService) Stop(ctx context.Context, req *ps.MaterializeStopRequest) (json.RawMessage, error) {
	s.StopFnInvoked = true
	return s.StopFn(ctx, req)
}

func (s *MaterializeService) Cancel(ctx context.Context, req *ps.MaterializeCancelRequest) (json.RawMessage, error) {
	s.CancelFnInvoked = true
	return s.CancelFn(ctx, req)
}
