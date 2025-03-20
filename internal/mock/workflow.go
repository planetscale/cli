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
}

func (w *WorkflowsService) List(ctx context.Context, req *ps.ListWorkflowsRequest) ([]*ps.Workflow, error) {
	w.ListFnInvoked = true
	return w.ListFn(ctx, req)
}

func (w *WorkflowsService) Get(ctx context.Context, req *ps.GetWorkflowRequest) (*ps.Workflow, error) {
	w.GetFnInvoked = true
	return w.GetFn(ctx, req)
}
