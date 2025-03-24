package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type WorkflowsService struct {
	ListFn        func(context.Context, *ps.ListWorkflowsRequest) ([]*ps.Workflow, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetWorkflowRequest) (*ps.Workflow, error)
	GetFnInvoked bool

	CreateFn        func(context.Context, *ps.CreateWorkflowRequest) (*ps.Workflow, error)
	CreateFnInvoked bool

	CancelFn        func(context.Context, *ps.CancelWorkflowRequest) (*ps.Workflow, error)
	CancelFnInvoked bool

	CompleteFn        func(context.Context, *ps.CompleteWorkflowRequest) (*ps.Workflow, error)
	CompleteFnInvoked bool

	CutoverFn        func(context.Context, *ps.CutoverWorkflowRequest) (*ps.Workflow, error)
	CutoverFnInvoked bool

	RetryFn        func(context.Context, *ps.RetryWorkflowRequest) (*ps.Workflow, error)
	RetryFnInvoked bool

	ReverseCutoverFn        func(context.Context, *ps.ReverseCutoverWorkflowRequest) (*ps.Workflow, error)
	ReverseCutoverFnInvoked bool

	ReverseTrafficFn        func(context.Context, *ps.ReverseTrafficWorkflowRequest) (*ps.Workflow, error)
	ReverseTrafficFnInvoked bool

	SwitchPrimariesFn        func(context.Context, *ps.SwitchPrimariesWorkflowRequest) (*ps.Workflow, error)
	SwitchPrimariesFnInvoked bool

	SwitchReplicasFn        func(context.Context, *ps.SwitchReplicasWorkflowRequest) (*ps.Workflow, error)
	SwitchReplicasFnInvoked bool

	VerifyDataFn        func(context.Context, *ps.VerifyDataWorkflowRequest) (*ps.Workflow, error)
	VerifyDataFnInvoked bool
}

func (w *WorkflowsService) List(ctx context.Context, req *ps.ListWorkflowsRequest) ([]*ps.Workflow, error) {
	w.ListFnInvoked = true
	return w.ListFn(ctx, req)
}

func (w *WorkflowsService) Get(ctx context.Context, req *ps.GetWorkflowRequest) (*ps.Workflow, error) {
	w.GetFnInvoked = true
	return w.GetFn(ctx, req)
}

func (w *WorkflowsService) Create(ctx context.Context, req *ps.CreateWorkflowRequest) (*ps.Workflow, error) {
	w.CreateFnInvoked = true
	return w.CreateFn(ctx, req)
}

func (w *WorkflowsService) Cancel(ctx context.Context, req *ps.CancelWorkflowRequest) (*ps.Workflow, error) {
	w.CancelFnInvoked = true
	return w.CancelFn(ctx, req)
}

func (w *WorkflowsService) Complete(ctx context.Context, req *ps.CompleteWorkflowRequest) (*ps.Workflow, error) {
	w.CompleteFnInvoked = true
	return w.CompleteFn(ctx, req)
}

func (w *WorkflowsService) Cutover(ctx context.Context, req *ps.CutoverWorkflowRequest) (*ps.Workflow, error) {
	w.CutoverFnInvoked = true
	return w.CutoverFn(ctx, req)
}

func (w *WorkflowsService) Retry(ctx context.Context, req *ps.RetryWorkflowRequest) (*ps.Workflow, error) {
	w.RetryFnInvoked = true
	return w.RetryFn(ctx, req)
}

func (w *WorkflowsService) ReverseCutover(ctx context.Context, req *ps.ReverseCutoverWorkflowRequest) (*ps.Workflow, error) {
	w.ReverseCutoverFnInvoked = true
	return w.ReverseCutoverFn(ctx, req)
}

func (w *WorkflowsService) ReverseTraffic(ctx context.Context, req *ps.ReverseTrafficWorkflowRequest) (*ps.Workflow, error) {
	w.ReverseTrafficFnInvoked = true
	return w.ReverseTrafficFn(ctx, req)
}

func (w *WorkflowsService) SwitchPrimaries(ctx context.Context, req *ps.SwitchPrimariesWorkflowRequest) (*ps.Workflow, error) {
	w.SwitchPrimariesFnInvoked = true
	return w.SwitchPrimariesFn(ctx, req)
}

func (w *WorkflowsService) SwitchReplicas(ctx context.Context, req *ps.SwitchReplicasWorkflowRequest) (*ps.Workflow, error) {
	w.SwitchReplicasFnInvoked = true
	return w.SwitchReplicasFn(ctx, req)
}

func (w *WorkflowsService) VerifyData(ctx context.Context, req *ps.VerifyDataWorkflowRequest) (*ps.Workflow, error) {
	w.VerifyDataFnInvoked = true
	return w.VerifyDataFn(ctx, req)
}