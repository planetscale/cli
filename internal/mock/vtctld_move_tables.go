package mock

import (
	"context"
	"encoding/json"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type MoveTablesService struct {
	ListFn        func(context.Context, *ps.MoveTablesListRequest) (json.RawMessage, error)
	ListFnInvoked bool

	CreateFn        func(context.Context, *ps.MoveTablesCreateRequest) (*ps.VtctldOperationReference, error)
	CreateFnInvoked bool

	ShowFn        func(context.Context, *ps.MoveTablesShowRequest) (json.RawMessage, error)
	ShowFnInvoked bool

	StatusFn        func(context.Context, *ps.MoveTablesStatusRequest) (json.RawMessage, error)
	StatusFnInvoked bool

	StartFn        func(context.Context, *ps.MoveTablesStartRequest) (json.RawMessage, error)
	StartFnInvoked bool

	SwitchTrafficFn        func(context.Context, *ps.MoveTablesSwitchTrafficRequest) (*ps.VtctldOperationReference, error)
	SwitchTrafficFnInvoked bool

	ReverseTrafficFn        func(context.Context, *ps.MoveTablesReverseTrafficRequest) (*ps.VtctldOperationReference, error)
	ReverseTrafficFnInvoked bool

	CancelFn        func(context.Context, *ps.MoveTablesCancelRequest) (*ps.VtctldOperationReference, error)
	CancelFnInvoked bool

	CompleteFn        func(context.Context, *ps.MoveTablesCompleteRequest) (*ps.VtctldOperationReference, error)
	CompleteFnInvoked bool
}

func (s *MoveTablesService) List(ctx context.Context, req *ps.MoveTablesListRequest) (json.RawMessage, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
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

func (s *MoveTablesService) Start(ctx context.Context, req *ps.MoveTablesStartRequest) (json.RawMessage, error) {
	s.StartFnInvoked = true
	return s.StartFn(ctx, req)
}

func (s *MoveTablesService) SwitchTraffic(ctx context.Context, req *ps.MoveTablesSwitchTrafficRequest) (*ps.VtctldOperationReference, error) {
	s.SwitchTrafficFnInvoked = true
	return s.SwitchTrafficFn(ctx, req)
}

func (s *MoveTablesService) ReverseTraffic(ctx context.Context, req *ps.MoveTablesReverseTrafficRequest) (*ps.VtctldOperationReference, error) {
	s.ReverseTrafficFnInvoked = true
	return s.ReverseTrafficFn(ctx, req)
}

func (s *MoveTablesService) Cancel(ctx context.Context, req *ps.MoveTablesCancelRequest) (*ps.VtctldOperationReference, error) {
	s.CancelFnInvoked = true
	return s.CancelFn(ctx, req)
}

func (s *MoveTablesService) Complete(ctx context.Context, req *ps.MoveTablesCompleteRequest) (*ps.VtctldOperationReference, error) {
	s.CompleteFnInvoked = true
	return s.CompleteFn(ctx, req)
}
