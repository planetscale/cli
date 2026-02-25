package mock

import (
	"context"
	"encoding/json"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type LookupVindexService struct {
	CreateFn        func(context.Context, *ps.LookupVindexCreateRequest) (json.RawMessage, error)
	CreateFnInvoked bool

	ShowFn        func(context.Context, *ps.LookupVindexShowRequest) (json.RawMessage, error)
	ShowFnInvoked bool

	ExternalizeFn        func(context.Context, *ps.LookupVindexExternalizeRequest) (json.RawMessage, error)
	ExternalizeFnInvoked bool

	InternalizeFn        func(context.Context, *ps.LookupVindexInternalizeRequest) (json.RawMessage, error)
	InternalizeFnInvoked bool

	CancelFn        func(context.Context, *ps.LookupVindexCancelRequest) (json.RawMessage, error)
	CancelFnInvoked bool

	CompleteFn        func(context.Context, *ps.LookupVindexCompleteRequest) (json.RawMessage, error)
	CompleteFnInvoked bool
}

func (s *LookupVindexService) Create(ctx context.Context, req *ps.LookupVindexCreateRequest) (json.RawMessage, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *LookupVindexService) Show(ctx context.Context, req *ps.LookupVindexShowRequest) (json.RawMessage, error) {
	s.ShowFnInvoked = true
	return s.ShowFn(ctx, req)
}

func (s *LookupVindexService) Externalize(ctx context.Context, req *ps.LookupVindexExternalizeRequest) (json.RawMessage, error) {
	s.ExternalizeFnInvoked = true
	return s.ExternalizeFn(ctx, req)
}

func (s *LookupVindexService) Internalize(ctx context.Context, req *ps.LookupVindexInternalizeRequest) (json.RawMessage, error) {
	s.InternalizeFnInvoked = true
	return s.InternalizeFn(ctx, req)
}

func (s *LookupVindexService) Cancel(ctx context.Context, req *ps.LookupVindexCancelRequest) (json.RawMessage, error) {
	s.CancelFnInvoked = true
	return s.CancelFn(ctx, req)
}

func (s *LookupVindexService) Complete(ctx context.Context, req *ps.LookupVindexCompleteRequest) (json.RawMessage, error) {
	s.CompleteFnInvoked = true
	return s.CompleteFn(ctx, req)
}
