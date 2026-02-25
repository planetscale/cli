package mock

import (
	"context"
	"encoding/json"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type MoveTablesService struct {
	CreateFn        func(context.Context, *ps.MoveTablesCreateRequest) (json.RawMessage, error)
	CreateFnInvoked bool

	ShowFn        func(context.Context, *ps.MoveTablesShowRequest) (json.RawMessage, error)
	ShowFnInvoked bool

	StatusFn        func(context.Context, *ps.MoveTablesStatusRequest) (json.RawMessage, error)
	StatusFnInvoked bool

	SwitchTrafficFn        func(context.Context, *ps.MoveTablesSwitchTrafficRequest) (json.RawMessage, error)
	SwitchTrafficFnInvoked bool

	ReverseTrafficFn        func(context.Context, *ps.MoveTablesReverseTrafficRequest) (json.RawMessage, error)
	ReverseTrafficFnInvoked bool

	CancelFn        func(context.Context, *ps.MoveTablesCancelRequest) (json.RawMessage, error)
	CancelFnInvoked bool

	CompleteFn        func(context.Context, *ps.MoveTablesCompleteRequest) (json.RawMessage, error)
	CompleteFnInvoked bool
}

func (s *MoveTablesService) Create(ctx context.Context, req *ps.MoveTablesCreateRequest) (json.RawMessage, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *MoveTablesService) Show(ctx context.Context, req *ps.MoveTablesShowRequest) (json.RawMessage, error) {
	s.ShowFnInvoked = true
	return s.ShowFn(ctx, req)
}

func (s *MoveTablesService) Status(ctx context.Context, req *ps.MoveTablesStatusRequest) (json.RawMessage, error) {
	s.StatusFnInvoked = true
	return s.StatusFn(ctx, req)
}

func (s *MoveTablesService) SwitchTraffic(ctx context.Context, req *ps.MoveTablesSwitchTrafficRequest) (json.RawMessage, error) {
	s.SwitchTrafficFnInvoked = true
	return s.SwitchTrafficFn(ctx, req)
}

func (s *MoveTablesService) ReverseTraffic(ctx context.Context, req *ps.MoveTablesReverseTrafficRequest) (json.RawMessage, error) {
	s.ReverseTrafficFnInvoked = true
	return s.ReverseTrafficFn(ctx, req)
}

func (s *MoveTablesService) Cancel(ctx context.Context, req *ps.MoveTablesCancelRequest) (json.RawMessage, error) {
	s.CancelFnInvoked = true
	return s.CancelFn(ctx, req)
}

func (s *MoveTablesService) Complete(ctx context.Context, req *ps.MoveTablesCompleteRequest) (json.RawMessage, error) {
	s.CompleteFnInvoked = true
	return s.CompleteFn(ctx, req)
}
