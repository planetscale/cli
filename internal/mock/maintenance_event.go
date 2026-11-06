package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type MaintenanceEventsService struct {
	ListFn        func(context.Context, *ps.ListMaintenanceEventsRequest, ...ps.ListOption) ([]*ps.MaintenanceEvent, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetMaintenanceEventRequest) (*ps.MaintenanceEvent, error)
	GetFnInvoked bool

	RescheduleFn        func(context.Context, *ps.RescheduleMaintenanceEventRequest) (*ps.MaintenanceEvent, error)
	RescheduleFnInvoked bool
}

func (s *MaintenanceEventsService) List(ctx context.Context, req *ps.ListMaintenanceEventsRequest, opts ...ps.ListOption) ([]*ps.MaintenanceEvent, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req, opts...)
}

func (s *MaintenanceEventsService) Get(ctx context.Context, req *ps.GetMaintenanceEventRequest) (*ps.MaintenanceEvent, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *MaintenanceEventsService) Reschedule(ctx context.Context, req *ps.RescheduleMaintenanceEventRequest) (*ps.MaintenanceEvent, error) {
	s.RescheduleFnInvoked = true
	return s.RescheduleFn(ctx, req)
}
