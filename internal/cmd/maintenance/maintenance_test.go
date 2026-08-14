package maintenance

import (
	"bytes"
	"context"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestMaintenance_ListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	nextWindow := time.Date(2025, 1, 22, 4, 0, 0, 0, time.UTC)
	lastWindow := time.Date(2025, 1, 15, 4, 0, 0, 0, time.UTC)
	mysqlVersion := "8.0.40"

	schedules := []*ps.MaintenanceSchedule{{
		ID:                         "sched-1",
		Name:                       "Weekly maintenance",
		CreatedAt:                  createdAt,
		UpdatedAt:                  createdAt,
		LastWindowDatetime:         lastWindow,
		NextWindowDatetime:         nextWindow,
		Duration:                   2,
		Day:                        3,
		Hour:                       4,
		Week:                       0,
		FrequencyValue:             1,
		FrequencyUnit:              "week",
		Enabled:                    true,
		Required:                   false,
		PendingMySQLVersionUpdate:  true,
		PendingMySQLVersion:        &mysqlVersion,
		PendingVitessVersionUpdate: false,
	}}

	svc := &mock.MaintenanceSchedulesService{
		ListFn: func(ctx context.Context, req *ps.ListMaintenanceSchedulesRequest, opts ...ps.ListOption) ([]*ps.MaintenanceSchedule, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return schedules, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{MaintenanceSchedules: svc}, nil
		},
	}

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{db})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, []*MaintenanceSchedule{{orig: schedules[0]}})
}

func TestMaintenance_ShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	scheduleID := "sched-1"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	nextWindow := time.Date(2025, 1, 22, 4, 0, 0, 0, time.UTC)
	lastWindow := time.Date(2025, 1, 15, 4, 0, 0, 0, time.UTC)

	schedule := &ps.MaintenanceSchedule{
		ID:                 scheduleID,
		Name:               "Weekly maintenance",
		CreatedAt:          createdAt,
		UpdatedAt:          createdAt,
		LastWindowDatetime: lastWindow,
		NextWindowDatetime: nextWindow,
		Duration:           2,
		Day:                3,
		Hour:               4,
		FrequencyValue:     1,
		FrequencyUnit:      "week",
		Enabled:            true,
	}

	svc := &mock.MaintenanceSchedulesService{
		GetFn: func(ctx context.Context, req *ps.GetMaintenanceScheduleRequest) (*ps.MaintenanceSchedule, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Schedule, qt.Equals, scheduleID)
			return schedule, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{MaintenanceSchedules: svc}, nil
		},
	}

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, scheduleID})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &MaintenanceSchedule{orig: schedule})
}

func TestFormatDay(t *testing.T) {
	c := qt.New(t)
	c.Assert(formatDay(0), qt.Equals, "Sunday")
	c.Assert(formatDay(3), qt.Equals, "Wednesday")
	c.Assert(formatDay(7), qt.Equals, "every day")
	c.Assert(formatDay(99), qt.Equals, "99")
}

func TestFormatFrequency(t *testing.T) {
	c := qt.New(t)
	c.Assert(formatFrequency(1, "week"), qt.Equals, "week")
	c.Assert(formatFrequency(2, "week"), qt.Equals, "2 weeks")
	c.Assert(formatFrequency(1, "once"), qt.Equals, "once")
}

func TestMaintenance_WindowsCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	scheduleID := "sched-1"
	createdAt := time.Date(2025, 1, 15, 4, 0, 0, 0, time.UTC)
	startedAt := time.Date(2025, 1, 15, 4, 0, 0, 0, time.UTC)
	finishedAt := time.Date(2025, 1, 15, 5, 30, 0, 0, time.UTC)

	windows := []*ps.MaintenanceWindow{{
		ID:         "win-1",
		CreatedAt:  createdAt,
		UpdatedAt:  finishedAt,
		StartedAt:  &startedAt,
		FinishedAt: &finishedAt,
	}}

	svc := &mock.MaintenanceSchedulesService{
		ListWindowsFn: func(ctx context.Context, req *ps.ListMaintenanceWindowsRequest, opts ...ps.ListOption) ([]*ps.MaintenanceWindow, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Schedule, qt.Equals, scheduleID)
			return windows, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{MaintenanceSchedules: svc}, nil
		},
	}

	cmd := WindowsCmd(ch)
	cmd.SetArgs([]string{db, scheduleID})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListWindowsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, []*MaintenanceWindow{{orig: windows[0]}})
}
