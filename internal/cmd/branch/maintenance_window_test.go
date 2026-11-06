package branch

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func maintenanceWindowTestHelper(format printer.Format, buf *bytes.Buffer, kind ps.DatabaseEngine, svc *mock.MaintenanceSchedulesService) *cmdutil.Helper {
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
				Databases:            postgresSwitchoverDatabaseService("app", kind),
				MaintenanceSchedules: svc,
			}, nil
		},
	}
}

func weeklyMaintenanceSchedule(id, branch string, day, hour int) *ps.MaintenanceSchedule {
	return &ps.MaintenanceSchedule{
		ID:                 id,
		Name:               "Production branch maintenance window",
		Day:                day,
		Hour:               hour,
		Duration:           1,
		FrequencyValue:     1,
		FrequencyUnit:      "week",
		Enabled:            true,
		NextWindowDatetime: time.Date(2026, 10, 14, 14, 0, 0, 0, time.UTC),
		DatabaseBranch:     &ps.MaintenanceBranch{ID: "branch-" + branch, Name: branch},
	}
}

// listMaintenanceSchedules serves the schedules as the first page and nothing
// after it.
func listMaintenanceSchedules(schedules ...*ps.MaintenanceSchedule) func(context.Context, *ps.ListMaintenanceSchedulesRequest, ...ps.ListOption) ([]*ps.MaintenanceSchedule, error) {
	calls := 0
	return func(context.Context, *ps.ListMaintenanceSchedulesRequest, ...ps.ListOption) ([]*ps.MaintenanceSchedule, error) {
		calls++
		if calls > 1 {
			return nil, nil
		}
		return schedules, nil
	}
}

func TestMaintenanceWindowShowCmd_PicksTheBranchWeeklySchedule(t *testing.T) {
	c := qt.New(t)

	oneTime := weeklyMaintenanceSchedule("once-main", "main", 1, 2)
	oneTime.FrequencyUnit = "once"
	deleted := weeklyMaintenanceSchedule("weekly-deleted-main", "main", 1, 2)
	deletedAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	deleted.DatabaseBranch.DeletedAt = &deletedAt
	want := weeklyMaintenanceSchedule("weekly-main", "main", 3, 14)

	svc := &mock.MaintenanceSchedulesService{
		ListFn: func(_ context.Context, req *ps.ListMaintenanceSchedulesRequest, _ ...ps.ListOption) ([]*ps.MaintenanceSchedule, error) {
			c.Assert(req, qt.DeepEquals, &ps.ListMaintenanceSchedulesRequest{Organization: "acme", Database: "app"})
			return []*ps.MaintenanceSchedule{
				oneTime,
				deleted,
				weeklyMaintenanceSchedule("weekly-staging", "staging", 5, 6),
				want,
			}, nil
		},
	}

	var buf bytes.Buffer
	cmd := MaintenanceWindowShowCmd(maintenanceWindowTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc))
	cmd.SetArgs([]string{"app", "main"})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.JSONEquals, want)
}

func TestMaintenanceWindowShowCmd_Human(t *testing.T) {
	c := qt.New(t)

	svc := &mock.MaintenanceSchedulesService{
		ListFn: listMaintenanceSchedules(weeklyMaintenanceSchedule("weekly-main", "main", 3, 14)),
	}

	var buf bytes.Buffer
	cmd := MaintenanceWindowShowCmd(maintenanceWindowTestHelper(printer.Human, &buf, ps.DatabaseEnginePostgres, svc))
	cmd.SetArgs([]string{"app", "main"})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.Matches, `(?s).*main.*Wednesday.*14.*2026-10-14 14:00 UTC.*`)
}

func TestMaintenanceWindowShowCmd_PagesThroughSchedules(t *testing.T) {
	c := qt.New(t)

	// The first page is short, which must not end the search.
	firstPage := []*ps.MaintenanceSchedule{weeklyMaintenanceSchedule("weekly-other", "other", 1, 2)}
	want := weeklyMaintenanceSchedule("weekly-main", "main", 3, 14)

	var queries []string
	svc := &mock.MaintenanceSchedulesService{
		ListFn: func(_ context.Context, _ *ps.ListMaintenanceSchedulesRequest, opts ...ps.ListOption) ([]*ps.MaintenanceSchedule, error) {
			listOpts := &ps.ListOptions{URLValues: &url.Values{}}
			for _, opt := range opts {
				c.Assert(opt(listOpts), qt.IsNil)
			}
			queries = append(queries, listOpts.URLValues.Encode())
			if len(queries) == 1 {
				return firstPage, nil
			}
			return []*ps.MaintenanceSchedule{want}, nil
		},
	}

	var buf bytes.Buffer
	cmd := MaintenanceWindowShowCmd(maintenanceWindowTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc))
	cmd.SetArgs([]string{"app", "main"})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(queries, qt.DeepEquals, []string{"page=1&per_page=100", "page=2&per_page=100"})
	c.Assert(buf.String(), qt.JSONEquals, want)
}

func TestMaintenanceWindowShowCmd_Errors(t *testing.T) {
	tests := []struct {
		name string
		list func(context.Context, *ps.ListMaintenanceSchedulesRequest, ...ps.ListOption) ([]*ps.MaintenanceSchedule, error)
		want string
	}{
		{
			name: "branch without a window",
			list: listMaintenanceSchedules(weeklyMaintenanceSchedule("weekly-staging", "staging", 5, 6)),
			want: `.*branch .*main.* in database .*app.* has no maintenance window`,
		},
		{
			name: "not available for the database",
			list: func(context.Context, *ps.ListMaintenanceSchedulesRequest, ...ps.ListOption) ([]*ps.MaintenanceSchedule, error) {
				return nil, &ps.Error{APICode: "method_not_allowed"}
			},
			want: `maintenance windows are not available for database .*app.*`,
		},
		{
			name: "database not found",
			list: func(context.Context, *ps.ListMaintenanceSchedulesRequest, ...ps.ListOption) ([]*ps.MaintenanceSchedule, error) {
				return nil, &ps.Error{Code: ps.ErrNotFound, APICode: "not_found"}
			},
			want: `database .*app.* does not exist in organization .*acme.*`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			var buf bytes.Buffer
			svc := &mock.MaintenanceSchedulesService{ListFn: tt.list}
			cmd := MaintenanceWindowShowCmd(maintenanceWindowTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc))
			cmd.SetArgs([]string{"app", "main"})

			c.Assert(cmd.Execute(), qt.ErrorMatches, tt.want)
		})
	}
}

func TestMaintenanceWindowCmd_RejectsOtherEngines(t *testing.T) {
	for _, args := range [][]string{{"show", "app", "main"}, {"set", "app", "main", "--hour", "4"}} {
		t.Run(args[0], func(t *testing.T) {
			c := qt.New(t)

			var buf bytes.Buffer
			svc := &mock.MaintenanceSchedulesService{}
			cmd := MaintenanceWindowCmd(maintenanceWindowTestHelper(printer.JSON, &buf, ps.DatabaseEngineNeki, svc))
			cmd.SetArgs(args)

			c.Assert(cmd.Execute(), qt.ErrorMatches, `(?s).*only available for PostgreSQL.*neki.*`)
			c.Assert(svc.ListFnInvoked, qt.IsFalse)
		})
	}
}

func TestMaintenanceWindowSetCmd(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	tests := []struct {
		name     string
		args     []string
		wantDay  *int
		wantHour *int
	}{
		{name: "day name", args: []string{"--day", "Monday"}, wantDay: intPtr(1)},
		{name: "day number", args: []string{"--day", "0"}, wantDay: intPtr(0)},
		{name: "hour", args: []string{"--hour", "0"}, wantHour: intPtr(0)},
		{name: "day and hour", args: []string{"--day", "saturday", "--hour", "23"}, wantDay: intPtr(6), wantHour: intPtr(23)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			updated := weeklyMaintenanceSchedule("weekly-main", "main", 6, 23)
			svc := &mock.MaintenanceSchedulesService{
				ListFn: listMaintenanceSchedules(weeklyMaintenanceSchedule("weekly-main", "main", 3, 14)),
				UpdateFn: func(_ context.Context, req *ps.UpdateMaintenanceScheduleRequest) (*ps.MaintenanceSchedule, error) {
					c.Assert(req, qt.DeepEquals, &ps.UpdateMaintenanceScheduleRequest{
						Organization: "acme",
						Database:     "app",
						Schedule:     "weekly-main",
						Day:          tt.wantDay,
						Hour:         tt.wantHour,
					})
					return updated, nil
				},
			}

			var buf bytes.Buffer
			cmd := MaintenanceWindowSetCmd(maintenanceWindowTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc))
			cmd.SetArgs(append([]string{"app", "main"}, tt.args...))

			c.Assert(cmd.Execute(), qt.IsNil)
			c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
			c.Assert(buf.String(), qt.JSONEquals, updated)
		})
	}
}

func TestMaintenanceWindowSetCmd_RejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "every day", args: []string{"--day", "7"}, want: `invalid --day "7".*`},
		{name: "unknown day", args: []string{"--day", "funday"}, want: `invalid --day "funday".*`},
		{name: "hour out of range", args: []string{"--hour", "24"}, want: `invalid --hour 24.*`},
		{name: "no flags", want: `at least one of --day or --hour is required`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			var buf bytes.Buffer
			svc := &mock.MaintenanceSchedulesService{}
			cmd := MaintenanceWindowSetCmd(maintenanceWindowTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc))
			cmd.SetArgs(append([]string{"app", "main"}, tt.args...))

			c.Assert(cmd.Execute(), qt.ErrorMatches, tt.want)
			c.Assert(svc.ListFnInvoked, qt.IsFalse)
		})
	}
}

func TestMaintenanceWindowSetCmd_SkipsUnchangedWindow(t *testing.T) {
	c := qt.New(t)

	current := weeklyMaintenanceSchedule("weekly-main", "main", 3, 14)
	svc := &mock.MaintenanceSchedulesService{ListFn: listMaintenanceSchedules(current)}

	var buf bytes.Buffer
	cmd := MaintenanceWindowSetCmd(maintenanceWindowTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc))
	cmd.SetArgs([]string{"app", "main", "--day", "wednesday", "--hour", "14"})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsFalse)
	c.Assert(buf.String(), qt.JSONEquals, current)
}

func TestMaintenanceWindowSetCmd_UpdateErrors(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		serviceToken bool
		want         string
	}{
		{
			name: "forbidden",
			err:  &ps.Error{APICode: "forbidden"},
			want: `you do not have permission to change the maintenance window of branch .*main.* in database .*app.*`,
		},
		{
			name:         "forbidden service token",
			err:          &ps.Error{APICode: "forbidden"},
			serviceToken: true,
			want:         `(?s)you do not have permission.*'write_database' permission.*`,
		},
		{
			name: "schedule gone",
			err:  &ps.Error{Code: ps.ErrNotFound, APICode: "not_found"},
			want: `the maintenance window of branch .*main.* could not be found.*run the command again`,
		},
		{
			name: "refused by the API",
			err:  errors.New("Cannot update schedule while maintenance window is open"),
			want: `Cannot update schedule while maintenance window is open`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			svc := &mock.MaintenanceSchedulesService{
				ListFn: listMaintenanceSchedules(weeklyMaintenanceSchedule("weekly-main", "main", 3, 14)),
				UpdateFn: func(context.Context, *ps.UpdateMaintenanceScheduleRequest) (*ps.MaintenanceSchedule, error) {
					return nil, tt.err
				},
			}

			var buf bytes.Buffer
			ch := maintenanceWindowTestHelper(printer.JSON, &buf, ps.DatabaseEnginePostgres, svc)
			if tt.serviceToken {
				ch.Config.ServiceTokenID = "token-id"
				ch.Config.ServiceToken = "token"
			}
			cmd := MaintenanceWindowSetCmd(ch)
			cmd.SetArgs([]string{"app", "main", "--hour", "4"})

			c.Assert(cmd.Execute(), qt.ErrorMatches, tt.want)
			c.Assert(buf.String(), qt.Equals, "")
		})
	}
}
