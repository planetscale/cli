package branch

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func maintenanceEventTestHelper(format printer.Format, buf *bytes.Buffer, kind ps.DatabaseEngine, svc *mock.MaintenanceEventsService) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	if format == printer.Human {
		p.SetHumanOutput(buf)
	}
	p.SetResourceOutput(buf)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases:         postgresSwitchoverDatabaseService("app", kind),
				MaintenanceEvents: svc,
			}, nil
		},
	}
}

func testMaintenanceEvent(branch string, availableTimes ...time.Time) *ps.MaintenanceEvent {
	scheduledAt := time.Date(2026, 10, 18, 3, 0, 0, 0, time.UTC)
	return &ps.MaintenanceEvent{
		ID:                   "event-1",
		State:                "scheduled",
		What:                 "Postgres update",
		Why:                  "Security fixes",
		ExpectedImpact:       "A brief failover",
		ScheduledAt:          &scheduledAt,
		NotAfter:             time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC),
		ReschedulesRemaining: 3,
		Branch:               &ps.MaintenanceBranch{ID: "branch-" + branch, Name: branch},
		AvailableTimes:       availableTimes,
	}
}

func getMaintenanceEventFn(event *ps.MaintenanceEvent) func(context.Context, *ps.GetMaintenanceEventRequest) (*ps.MaintenanceEvent, error) {
	return func(context.Context, *ps.GetMaintenanceEventRequest) (*ps.MaintenanceEvent, error) {
		return event, nil
	}
}

var (
	maintenanceEventEarlier = time.Date(2026, 10, 11, 3, 0, 0, 0, time.UTC)
	maintenanceEventLater   = time.Date(2026, 10, 25, 3, 0, 0, 0, time.UTC)
)

func TestMaintenanceEventListCmd(t *testing.T) {
	c := qt.New(t)

	events := []*ps.MaintenanceEvent{testMaintenanceEvent("main")}
	svc := &mock.MaintenanceEventsService{
		ListFn: func(_ context.Context, req *ps.ListMaintenanceEventsRequest, opts ...ps.ListOption) ([]*ps.MaintenanceEvent, error) {
			c.Assert(req, qt.DeepEquals, &ps.ListMaintenanceEventsRequest{
				Organization: "acme",
				Database:     "app",
				Branch:       "main",
				State:        "scheduled",
			})
			listOpts := &ps.ListOptions{URLValues: &url.Values{}}
			for _, opt := range opts {
				c.Assert(opt(listOpts), qt.IsNil)
			}
			c.Assert(listOpts.URLValues.Encode(), qt.Equals, "page=2&per_page=10")
			return events, nil
		},
	}

	var buf bytes.Buffer
	cmd := MaintenanceEventListCmd(maintenanceEventTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc))
	cmd.SetArgs([]string{"app", "main", "--state", "scheduled", "--page", "2", "--per-page", "10"})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.JSONEquals, events)
}

func TestMaintenanceEventListCmd_HumanAndCSV(t *testing.T) {
	for _, format := range []printer.Format{printer.Human, printer.CSV} {
		t.Run(format.String(), func(t *testing.T) {
			c := qt.New(t)

			svc := &mock.MaintenanceEventsService{
				ListFn: func(context.Context, *ps.ListMaintenanceEventsRequest, ...ps.ListOption) ([]*ps.MaintenanceEvent, error) {
					return []*ps.MaintenanceEvent{testMaintenanceEvent("main")}, nil
				},
			}

			var buf bytes.Buffer
			cmd := MaintenanceEventListCmd(maintenanceEventTestHelper(format, &buf, ps.DatabaseEnginePostgres, svc))
			cmd.SetArgs([]string{"app", "main"})

			c.Assert(cmd.Execute(), qt.IsNil)
			c.Assert(buf.String(), qt.Matches, `(?s).*event-1.*main.*scheduled.*Postgres update.*2026-10-18 03:00 UTC.*2026-10-30 00:00 UTC.*`)
		})
	}
}

func TestMaintenanceEventListCmd_Empty(t *testing.T) {
	tests := []struct {
		name   string
		format printer.Format
		want   string
	}{
		{name: "human", format: printer.Human, want: "No maintenance events exist for branch main in app."},
		{name: "json", format: printer.JSON, want: "[]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			svc := &mock.MaintenanceEventsService{
				ListFn: func(context.Context, *ps.ListMaintenanceEventsRequest, ...ps.ListOption) ([]*ps.MaintenanceEvent, error) {
					return nil, nil
				},
			}

			var buf bytes.Buffer
			cmd := MaintenanceEventListCmd(maintenanceEventTestHelper(tt.format, &buf, ps.DatabaseEnginePostgres, svc))
			cmd.SetArgs([]string{"app", "main"})

			c.Assert(cmd.Execute(), qt.IsNil)
			c.Assert(strings.TrimSpace(buf.String()), qt.Satisfies, func(got string) bool {
				return strings.HasSuffix(got, tt.want)
			})
		})
	}
}

func TestMaintenanceEventListCmd_RejectsUnknownState(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	svc := &mock.MaintenanceEventsService{}
	cmd := MaintenanceEventListCmd(maintenanceEventTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc))
	cmd.SetArgs([]string{"app", "main", "--state", "pending"})

	c.Assert(cmd.Execute(), qt.ErrorMatches, `invalid --state "pending": use one of scheduled, in_progress, completed, canceled, expired`)
	c.Assert(svc.ListFnInvoked, qt.IsFalse)
}

func TestMaintenanceEventListCmd_NotAvailable(t *testing.T) {
	c := qt.New(t)

	svc := &mock.MaintenanceEventsService{
		ListFn: func(context.Context, *ps.ListMaintenanceEventsRequest, ...ps.ListOption) ([]*ps.MaintenanceEvent, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound, APICode: "not_found"}
		},
	}

	var buf bytes.Buffer
	cmd := MaintenanceEventListCmd(maintenanceEventTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc))
	cmd.SetArgs([]string{"app", "main"})

	c.Assert(cmd.Execute(), qt.ErrorMatches, `maintenance events are not available for database .*app.*`)
}

func TestMaintenanceEventCmd_RejectsOtherEngines(t *testing.T) {
	commands := [][]string{
		{"list", "app", "main"},
		{"show", "app", "main", "event-1"},
		{"reschedule", "app", "main", "event-1", "--to", "2026-10-25T03:00:00Z"},
	}

	for _, args := range commands {
		t.Run(args[0], func(t *testing.T) {
			c := qt.New(t)

			var buf bytes.Buffer
			svc := &mock.MaintenanceEventsService{}
			cmd := MaintenanceEventCmd(maintenanceEventTestHelper(printer.JSON, &buf, ps.DatabaseEngineNeki, svc))
			cmd.SetArgs(args)

			c.Assert(cmd.Execute(), qt.ErrorMatches, `(?s).*only available for PostgreSQL.*neki.*`)
			c.Assert(svc.ListFnInvoked || svc.GetFnInvoked, qt.IsFalse)
		})
	}
}

func TestMaintenanceEventShowCmd(t *testing.T) {
	c := qt.New(t)

	event := testMaintenanceEvent("main", maintenanceEventEarlier, maintenanceEventLater)
	svc := &mock.MaintenanceEventsService{
		GetFn: func(_ context.Context, req *ps.GetMaintenanceEventRequest) (*ps.MaintenanceEvent, error) {
			c.Assert(req, qt.DeepEquals, &ps.GetMaintenanceEventRequest{Organization: "acme", Database: "app", ID: "event-1"})
			return event, nil
		},
	}

	var buf bytes.Buffer
	cmd := MaintenanceEventShowCmd(maintenanceEventTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc))
	cmd.SetArgs([]string{"app", "main", "event-1"})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.JSONEquals, event)
}

func TestMaintenanceEventShowCmd_HumanListsDetailsAndAvailableTimes(t *testing.T) {
	c := qt.New(t)

	svc := &mock.MaintenanceEventsService{
		GetFn: getMaintenanceEventFn(testMaintenanceEvent("main", maintenanceEventEarlier, maintenanceEventLater)),
	}

	var buf bytes.Buffer
	cmd := MaintenanceEventShowCmd(maintenanceEventTestHelper(printer.Human, &buf, ps.DatabaseEnginePostgres, svc))
	cmd.SetArgs([]string{"app", "main", "event-1"})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.Matches, `(?s).*event-1.*Why: Security fixes.*Expected impact: A brief failover.*2026-10-11T03:00:00Z.*2026-10-25T03:00:00Z.*`)
}

func TestMaintenanceEventShowCmd_NotFound(t *testing.T) {
	tests := []struct {
		name string
		get  func(context.Context, *ps.GetMaintenanceEventRequest) (*ps.MaintenanceEvent, error)
	}{
		{
			name: "unknown event",
			get: func(context.Context, *ps.GetMaintenanceEventRequest) (*ps.MaintenanceEvent, error) {
				return nil, &ps.Error{Code: ps.ErrNotFound, APICode: "not_found"}
			},
		},
		{
			name: "event of another branch",
			get:  getMaintenanceEventFn(testMaintenanceEvent("staging")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			var buf bytes.Buffer
			svc := &mock.MaintenanceEventsService{GetFn: tt.get}
			cmd := MaintenanceEventShowCmd(maintenanceEventTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc))
			cmd.SetArgs([]string{"app", "main", "event-1"})

			c.Assert(cmd.Execute(), qt.ErrorMatches, `maintenance event .*event-1.* does not exist for branch .*main.* in database .*app.*`)
			c.Assert(buf.String(), qt.Equals, "")
		})
	}
}

func TestMaintenanceEventRescheduleCmd(t *testing.T) {
	// The second case names the same instant with an offset.
	for _, to := range []string{"2026-10-25T03:00:00Z", "2026-10-24T20:00:00-07:00"} {
		t.Run(to, func(t *testing.T) {
			c := qt.New(t)

			rescheduled := testMaintenanceEvent("main", maintenanceEventEarlier)
			rescheduled.ScheduledAt = &maintenanceEventLater
			rescheduled.ReschedulesRemaining = 2

			svc := &mock.MaintenanceEventsService{
				GetFn: getMaintenanceEventFn(testMaintenanceEvent("main", maintenanceEventEarlier, maintenanceEventLater)),
				RescheduleFn: func(_ context.Context, req *ps.RescheduleMaintenanceEventRequest) (*ps.MaintenanceEvent, error) {
					c.Assert(req, qt.DeepEquals, &ps.RescheduleMaintenanceEventRequest{
						Organization: "acme",
						Database:     "app",
						ID:           "event-1",
						ScheduledAt:  maintenanceEventLater,
					})
					return rescheduled, nil
				},
			}

			var buf bytes.Buffer
			cmd := MaintenanceEventRescheduleCmd(maintenanceEventTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc))
			cmd.SetArgs([]string{"app", "main", "event-1", "--to", to})

			c.Assert(cmd.Execute(), qt.IsNil)
			c.Assert(svc.RescheduleFnInvoked, qt.IsTrue)
			c.Assert(buf.String(), qt.JSONEquals, rescheduled)
		})
	}
}

func TestMaintenanceEventRescheduleCmd_RefusesBeforeTheRequest(t *testing.T) {
	completed := testMaintenanceEvent("main")
	completed.State = "completed"
	completed.ReschedulesRemaining = 0
	outOfReschedules := testMaintenanceEvent("main")
	outOfReschedules.ReschedulesRemaining = 0

	tests := []struct {
		name  string
		event *ps.MaintenanceEvent
		to    string
		want  string
	}{
		{
			name:  "time is not available",
			event: testMaintenanceEvent("main", maintenanceEventEarlier, maintenanceEventLater),
			to:    "2026-10-25T04:00:00Z",
			want:  `(?s)maintenance event .*event-1.* cannot be rescheduled to 2026-10-25T04:00:00Z; it can be rescheduled to:\n  2026-10-11T03:00:00Z\n  2026-10-25T03:00:00Z`,
		},
		{
			name:  "not scheduled",
			event: completed,
			to:    "2026-10-25T03:00:00Z",
			want:  `maintenance event .*event-1.* is completed and cannot be rescheduled`,
		},
		{
			name:  "no reschedules remaining",
			event: outOfReschedules,
			to:    "2026-10-25T03:00:00Z",
			want:  `maintenance event .*event-1.* has no reschedules remaining`,
		},
		{
			name:  "no available times",
			event: testMaintenanceEvent("main"),
			to:    "2026-10-25T03:00:00Z",
			want:  `maintenance event .*event-1.* cannot be rescheduled right now`,
		},
		{
			name:  "event of another branch",
			event: testMaintenanceEvent("staging", maintenanceEventLater),
			to:    "2026-10-25T03:00:00Z",
			want:  `maintenance event .*event-1.* does not exist for branch .*main.*`,
		},
		{
			name:  "not a time",
			event: testMaintenanceEvent("main", maintenanceEventLater),
			to:    "sunday",
			want:  `invalid --to "sunday": use an RFC 3339 time such as .*`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			var buf bytes.Buffer
			svc := &mock.MaintenanceEventsService{GetFn: getMaintenanceEventFn(tt.event)}
			cmd := MaintenanceEventRescheduleCmd(maintenanceEventTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc))
			cmd.SetArgs([]string{"app", "main", "event-1", "--to", tt.to})

			c.Assert(cmd.Execute(), qt.ErrorMatches, tt.want)
			c.Assert(svc.RescheduleFnInvoked, qt.IsFalse)
		})
	}
}

func TestMaintenanceEventRescheduleCmd_RescheduleErrors(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		serviceToken bool
		want         string
	}{
		{
			name: "forbidden",
			err:  &ps.Error{APICode: "forbidden"},
			want: `you do not have permission to reschedule maintenance for branch .*main.* in database .*app.*`,
		},
		{
			name:         "forbidden service token",
			err:          &ps.Error{APICode: "forbidden"},
			serviceToken: true,
			want:         `(?s)you do not have permission.*'write_database' permission.*`,
		},
		{
			name: "refused by the API",
			err:  errors.New("Maintenance cannot be rescheduled while the maintenance window is open"),
			want: `Maintenance cannot be rescheduled while the maintenance window is open`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			svc := &mock.MaintenanceEventsService{
				GetFn: getMaintenanceEventFn(testMaintenanceEvent("main", maintenanceEventLater)),
				RescheduleFn: func(context.Context, *ps.RescheduleMaintenanceEventRequest) (*ps.MaintenanceEvent, error) {
					return nil, tt.err
				},
			}

			var buf bytes.Buffer
			ch := maintenanceEventTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc)
			if tt.serviceToken {
				ch.Config.ServiceTokenID = "token-id"
				ch.Config.ServiceToken = "token"
			}
			cmd := MaintenanceEventRescheduleCmd(ch)
			cmd.SetArgs([]string{"app", "main", "event-1", "--to", "2026-10-25T03:00:00Z"})

			c.Assert(cmd.Execute(), qt.ErrorMatches, tt.want)
			c.Assert(buf.String(), qt.Equals, "")
		})
	}
}
