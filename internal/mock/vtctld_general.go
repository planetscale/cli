package mock

import (
	"context"
	"encoding/json"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type VtctldService struct {
	ListWorkflowsFn        func(context.Context, *ps.VtctldListWorkflowsRequest) (json.RawMessage, error)
	ListWorkflowsFnInvoked bool

	ListKeyspacesFn        func(context.Context, *ps.VtctldListKeyspacesRequest) (json.RawMessage, error)
	ListKeyspacesFnInvoked bool
}

func (s *VtctldService) ListWorkflows(ctx context.Context, req *ps.VtctldListWorkflowsRequest) (json.RawMessage, error) {
	s.ListWorkflowsFnInvoked = true
	return s.ListWorkflowsFn(ctx, req)
}

func (s *VtctldService) ListKeyspaces(ctx context.Context, req *ps.VtctldListKeyspacesRequest) (json.RawMessage, error) {
	s.ListKeyspacesFnInvoked = true
	return s.ListKeyspacesFn(ctx, req)
}
