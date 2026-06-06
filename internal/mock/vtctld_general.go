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

	GetRoutingRulesFn        func(context.Context, *ps.VtctldGetRoutingRulesRequest) (json.RawMessage, error)
	GetRoutingRulesFnInvoked bool

	GetShardFn        func(context.Context, *ps.VtctldGetShardRequest) (json.RawMessage, error)
	GetShardFnInvoked bool

	SetShardTabletControlFn        func(context.Context, *ps.VtctldSetShardTabletControlRequest) (json.RawMessage, error)
	SetShardTabletControlFnInvoked bool

	ListTabletsFn        func(context.Context, *ps.ListBranchTabletsRequest) ([]*ps.TabletGroup, error)
	ListTabletsFnInvoked bool

	StartWorkflowFn        func(context.Context, *ps.VtctldStartWorkflowRequest) (json.RawMessage, error)
	StartWorkflowFnInvoked bool

	StopWorkflowFn        func(context.Context, *ps.VtctldStopWorkflowRequest) (json.RawMessage, error)
	StopWorkflowFnInvoked bool

	GetThrottlerStatusFn        func(context.Context, *ps.VtctldGetThrottlerStatusRequest) (json.RawMessage, error)
	GetThrottlerStatusFnInvoked bool

	CheckThrottlerFn        func(context.Context, *ps.VtctldCheckThrottlerRequest) (json.RawMessage, error)
	CheckThrottlerFnInvoked bool

	UpdateThrottlerConfigFn        func(context.Context, *ps.VtctldUpdateThrottlerConfigRequest) (json.RawMessage, error)
	UpdateThrottlerConfigFnInvoked bool

	GetOperationFn        func(context.Context, *ps.GetVtctldOperationRequest) (*ps.VtctldOperation, error)
	GetOperationFnInvoked bool
}

func (s *VtctldService) ListWorkflows(ctx context.Context, req *ps.VtctldListWorkflowsRequest) (json.RawMessage, error) {
	s.ListWorkflowsFnInvoked = true
	return s.ListWorkflowsFn(ctx, req)
}

func (s *VtctldService) ListKeyspaces(ctx context.Context, req *ps.VtctldListKeyspacesRequest) (json.RawMessage, error) {
	s.ListKeyspacesFnInvoked = true
	return s.ListKeyspacesFn(ctx, req)
}

func (s *VtctldService) GetRoutingRules(ctx context.Context, req *ps.VtctldGetRoutingRulesRequest) (json.RawMessage, error) {
	s.GetRoutingRulesFnInvoked = true
	return s.GetRoutingRulesFn(ctx, req)
}

func (s *VtctldService) GetShard(ctx context.Context, req *ps.VtctldGetShardRequest) (json.RawMessage, error) {
	s.GetShardFnInvoked = true
	return s.GetShardFn(ctx, req)
}

func (s *VtctldService) SetShardTabletControl(ctx context.Context, req *ps.VtctldSetShardTabletControlRequest) (json.RawMessage, error) {
	s.SetShardTabletControlFnInvoked = true
	return s.SetShardTabletControlFn(ctx, req)
}

func (s *VtctldService) ListTablets(ctx context.Context, req *ps.ListBranchTabletsRequest) ([]*ps.TabletGroup, error) {
	s.ListTabletsFnInvoked = true
	return s.ListTabletsFn(ctx, req)
}

func (s *VtctldService) StartWorkflow(ctx context.Context, req *ps.VtctldStartWorkflowRequest) (json.RawMessage, error) {
	s.StartWorkflowFnInvoked = true
	return s.StartWorkflowFn(ctx, req)
}

func (s *VtctldService) StopWorkflow(ctx context.Context, req *ps.VtctldStopWorkflowRequest) (json.RawMessage, error) {
	s.StopWorkflowFnInvoked = true
	return s.StopWorkflowFn(ctx, req)
}

func (s *VtctldService) GetThrottlerStatus(ctx context.Context, req *ps.VtctldGetThrottlerStatusRequest) (json.RawMessage, error) {
	s.GetThrottlerStatusFnInvoked = true
	return s.GetThrottlerStatusFn(ctx, req)
}

func (s *VtctldService) CheckThrottler(ctx context.Context, req *ps.VtctldCheckThrottlerRequest) (json.RawMessage, error) {
	s.CheckThrottlerFnInvoked = true
	return s.CheckThrottlerFn(ctx, req)
}

func (s *VtctldService) UpdateThrottlerConfig(ctx context.Context, req *ps.VtctldUpdateThrottlerConfigRequest) (json.RawMessage, error) {
	s.UpdateThrottlerConfigFnInvoked = true
	return s.UpdateThrottlerConfigFn(ctx, req)
}

func (s *VtctldService) GetOperation(ctx context.Context, req *ps.GetVtctldOperationRequest) (*ps.VtctldOperation, error) {
	s.GetOperationFnInvoked = true
	return s.GetOperationFn(ctx, req)
}
