package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestMaintenanceSchedules_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/maintenance-schedules")
		w.WriteHeader(200)
		out := `{
			"data":[{
				"id":"sched-1",
				"name":"Weekly maintenance",
				"created_at":"2021-01-14T10:19:23.000Z",
				"updated_at":"2021-01-14T10:19:23.000Z",
				"last_window_datetime":"2021-01-07T04:00:00.000Z",
				"next_window_datetime":"2021-01-14T04:00:00.000Z",
				"duration":2,
				"day":3,
				"hour":4,
				"week":0,
				"frequency_value":1,
				"frequency_unit":"week",
				"enabled":true,
				"expires_at":null,
				"deadline_at":null,
				"required":false,
				"pending_vitess_version_update":false,
				"pending_vitess_version":null,
				"pending_mysql_version_update":true,
				"pending_mysql_version":"8.0.40"
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	schedules, err := client.MaintenanceSchedules.List(context.Background(), &ListMaintenanceSchedulesRequest{
		Organization: testOrg,
		Database:     "my-db",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(schedules, qt.HasLen, 1)
	c.Assert(schedules[0].ID, qt.Equals, "sched-1")
	c.Assert(schedules[0].Name, qt.Equals, "Weekly maintenance")
	c.Assert(schedules[0].Enabled, qt.IsTrue)
	c.Assert(schedules[0].Day, qt.Equals, 3)
	c.Assert(schedules[0].Hour, qt.Equals, 4)
	c.Assert(schedules[0].PendingMySQLVersionUpdate, qt.IsTrue)
	c.Assert(schedules[0].PendingMySQLVersion, qt.IsNotNil)
	c.Assert(*schedules[0].PendingMySQLVersion, qt.Equals, "8.0.40")
	c.Assert(schedules[0].NextWindowDatetime.Equal(time.Date(2021, time.January, 14, 4, 0, 0, 0, time.UTC)), qt.IsTrue)
}

func TestMaintenanceSchedules_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/maintenance-schedules/sched-1")
		w.WriteHeader(200)
		out := `{
			"id":"sched-1",
			"name":"Weekly maintenance",
			"created_at":"2021-01-14T10:19:23.000Z",
			"updated_at":"2021-01-14T10:19:23.000Z",
			"last_window_datetime":"2021-01-07T04:00:00.000Z",
			"next_window_datetime":"2021-01-14T04:00:00.000Z",
			"duration":2,
			"day":3,
			"hour":4,
			"week":0,
			"frequency_value":1,
			"frequency_unit":"week",
			"enabled":true,
			"expires_at":null,
			"deadline_at":null,
			"required":false,
			"pending_vitess_version_update":false,
			"pending_vitess_version":null,
			"pending_mysql_version_update":false,
			"pending_mysql_version":null
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	schedule, err := client.MaintenanceSchedules.Get(context.Background(), &GetMaintenanceScheduleRequest{
		Organization: testOrg,
		Database:     "my-db",
		Schedule:     "sched-1",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(schedule.Name, qt.Equals, "Weekly maintenance")
	c.Assert(schedule.FrequencyUnit, qt.Equals, "week")
}

func TestMaintenanceSchedules_ListWindows(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/maintenance-schedules/sched-1/windows")
		w.WriteHeader(200)
		out := `{
			"data":[{
				"id":"win-1",
				"created_at":"2021-01-07T04:00:00.000Z",
				"updated_at":"2021-01-07T06:00:00.000Z",
				"started_at":"2021-01-07T04:00:00.000Z",
				"finished_at":"2021-01-07T05:30:00.000Z"
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	windows, err := client.MaintenanceSchedules.ListWindows(context.Background(), &ListMaintenanceWindowsRequest{
		Organization: testOrg,
		Database:     "my-db",
		Schedule:     "sched-1",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(windows, qt.HasLen, 1)
	c.Assert(windows[0].ID, qt.Equals, "win-1")
	c.Assert(windows[0].StartedAt, qt.IsNotNil)
	c.Assert(windows[0].FinishedAt, qt.IsNotNil)
}
