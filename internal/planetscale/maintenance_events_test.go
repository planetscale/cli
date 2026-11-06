package planetscale

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

const testMaintenanceEvent = `{
	"id":"event-1",
	"state":"scheduled",
	"what":"Postgres update",
	"why":"Security fixes",
	"expected_impact":"A brief failover",
	"scheduled_at":"2026-10-18T03:00:00.000Z",
	"not_after":"2026-10-30T00:00:00.000Z",
	"completed_at":null,
	"reschedules_remaining":3,
	"branch":{"id":"branch-1","name":"main","created_at":"2026-01-01T00:00:00.000Z","updated_at":"2026-01-01T00:00:00.000Z","deleted_at":null},
	"available_times":["2026-10-11T03:00:00.000Z","2026-10-25T03:00:00.000Z"]
}`

func TestMaintenanceEvents_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/maintenance-events")
		c.Assert(r.URL.RawQuery, qt.Equals, "branch=main&page=2&per_page=10&state=scheduled")
		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":[` + testMaintenanceEvent + `]}`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	events, err := client.MaintenanceEvents.List(context.Background(), &ListMaintenanceEventsRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
		State:        "scheduled",
	}, WithPage(2), WithPerPage(10))
	c.Assert(err, qt.IsNil)

	scheduledAt := time.Date(2026, time.October, 18, 3, 0, 0, 0, time.UTC)
	branchTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	c.Assert(events, qt.DeepEquals, []*MaintenanceEvent{{
		ID:                   "event-1",
		State:                "scheduled",
		What:                 "Postgres update",
		Why:                  "Security fixes",
		ExpectedImpact:       "A brief failover",
		ScheduledAt:          &scheduledAt,
		NotAfter:             time.Date(2026, time.October, 30, 0, 0, 0, 0, time.UTC),
		ReschedulesRemaining: 3,
		Branch:               &MaintenanceBranch{ID: "branch-1", Name: "main", CreatedAt: branchTime, UpdatedAt: branchTime},
		AvailableTimes: []time.Time{
			time.Date(2026, time.October, 11, 3, 0, 0, 0, time.UTC),
			time.Date(2026, time.October, 25, 3, 0, 0, 0, time.UTC),
		},
	}})
}

func TestMaintenanceEvents_ListWithoutFilters(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.RawQuery, qt.Equals, "per_page=100")
		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":[]}`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	events, err := client.MaintenanceEvents.List(context.Background(), &ListMaintenanceEventsRequest{
		Organization: testOrg,
		Database:     "my-db",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(events, qt.HasLen, 0)
}

func TestMaintenanceEvents_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/maintenance-events/event-1")
		w.WriteHeader(200)
		_, err := w.Write([]byte(testMaintenanceEvent))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	event, err := client.MaintenanceEvents.Get(context.Background(), &GetMaintenanceEventRequest{
		Organization: testOrg,
		Database:     "my-db",
		ID:           "event-1",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(event.ID, qt.Equals, "event-1")
	c.Assert(event.Branch.Name, qt.Equals, "main")
}

func TestMaintenanceEvents_Reschedule(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/maintenance-events/event-1")
		body, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(string(body), qt.JSONEquals, map[string]string{"scheduled_at": "2026-10-25T03:00:00Z"})
		w.WriteHeader(200)
		_, err = w.Write([]byte(testMaintenanceEvent))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	// A time given with an offset is still sent in UTC.
	event, err := client.MaintenanceEvents.Reschedule(context.Background(), &RescheduleMaintenanceEventRequest{
		Organization: testOrg,
		Database:     "my-db",
		ID:           "event-1",
		ScheduledAt:  time.Date(2026, time.October, 24, 20, 0, 0, 0, time.FixedZone("", -7*60*60)),
	})
	c.Assert(err, qt.IsNil)
	c.Assert(event.ID, qt.Equals, "event-1")
}
