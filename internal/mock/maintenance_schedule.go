package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type MaintenanceSchedulesService struct {
	ListFn        func(context.Context, *ps.ListMaintenanceSchedulesRequest, ...ps.ListOption) ([]*ps.MaintenanceSchedule, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetMaintenanceScheduleRequest) (*ps.MaintenanceSchedule, error)
	GetFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdateMaintenanceScheduleRequest) (*ps.MaintenanceSchedule, error)
	UpdateFnInvoked bool

	ListWindowsFn        func(context.Context, *ps.ListMaintenanceWindowsRequest, ...ps.ListOption) ([]*ps.MaintenanceWindow, error)
	ListWindowsFnInvoked bool
}

func (s *MaintenanceSchedulesService) List(ctx context.Context, req *ps.ListMaintenanceSchedulesRequest, opts ...ps.ListOption) ([]*ps.MaintenanceSchedule, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req, opts...)
}

func (s *MaintenanceSchedulesService) Get(ctx context.Context, req *ps.GetMaintenanceScheduleRequest) (*ps.MaintenanceSchedule, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *MaintenanceSchedulesService) Update(ctx context.Context, req *ps.UpdateMaintenanceScheduleRequest) (*ps.MaintenanceSchedule, error) {
	s.UpdateFnInvoked = true
	return s.UpdateFn(ctx, req)
}

func (s *MaintenanceSchedulesService) ListWindows(ctx context.Context, req *ps.ListMaintenanceWindowsRequest, opts ...ps.ListOption) ([]*ps.MaintenanceWindow, error) {
	s.ListWindowsFnInvoked = true
	return s.ListWindowsFn(ctx, req, opts...)
}
