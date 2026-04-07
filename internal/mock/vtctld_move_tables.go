package mock

import (
	"context"
	"encoding/json"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type MoveTablesService struct {
	CreateFn        func(context.Context, *ps.MoveTablesCreateRequest) (*ps.VtctldOperationReference, error)
	CreateFnInvoked bool

	ShowFn        func(context.Context, *ps.MoveTablesShowRequest) (json.RawMessage, error)
	ShowFnInvoked bool

	StatusFn        func(context.Context, *ps.MoveTablesStatusRequest) (json.RawMessage, error)
	StatusFnInvoked bool

	SwitchTrafficFn        func(context.Context, *ps.MoveTablesSwitchTrafficRequest) (*ps.VtctldOperationReference, error)
	SwitchTrafficFnInvoked bool

	ReverseTrafficFn        func(context.Context, *ps.MoveTablesReverseTrafficRequest) (*ps.VtctldOperationReference, error)
	ReverseTrafficFnInvoked bool

	CancelFn        func(context.Context, *ps.MoveTablesCancelRequest) (json.RawMessage, error)
	CancelFnInvoked bool

	CompleteFn        func(context.Context, *ps.MoveTablesCompleteRequest) (*ps.VtctldOperationReference, error)
	CompleteFnInvoked bool
}

func (s *MoveTablesService) Create(ctx context.Context, req *ps.MoveTablesCreateRequest) (*ps.VtctldOperationReference, error) {
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

func (s *MoveTablesService) SwitchTraffic(ctx context.Context, req *ps.MoveTablesSwitchTrafficRequest) (*ps.VtctldOperationReference, error) {
	s.SwitchTrafficFnInvoked = true
	return s.SwitchTrafficFn(ctx, req)
}

func (s *MoveTablesService) ReverseTraffic(ctx context.Context, req *ps.MoveTablesReverseTrafficRequest) (*ps.VtctldOperationReference, error) {
	s.ReverseTrafficFnInvoked = true
	return s.ReverseTrafficFn(ctx, req)
}

func (s *MoveTablesService) Cancel(ctx context.Context, req *ps.MoveTablesCancelRequest) (json.RawMessage, error) {
	s.CancelFnInvoked = true
	return s.CancelFn(ctx, req)
}

func (s *MoveTablesService) Complete(ctx context.Context, req *ps.MoveTablesCompleteRequest) (*ps.VtctldOperationReference, error) {
	s.CompleteFnInvoked = true
	return s.CompleteFn(ctx, req)
}
