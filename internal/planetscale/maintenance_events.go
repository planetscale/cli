package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// MaintenanceEvent represents scheduled maintenance for a Postgres branch.
type MaintenanceEvent struct {
	ID                   string             `json:"id"`
	State                string             `json:"state"`
	What                 string             `json:"what"`
	Why                  string             `json:"why"`
	ExpectedImpact       string             `json:"expected_impact"`
	ScheduledAt          *time.Time         `json:"scheduled_at"`
	NotAfter             time.Time          `json:"not_after"`
	CompletedAt          *time.Time         `json:"completed_at"`
	ReschedulesRemaining int                `json:"reschedules_remaining"`
	Branch               *MaintenanceBranch `json:"branch"`
	// AvailableTimes are the only times the event can be rescheduled to.
	AvailableTimes []time.Time `json:"available_times"`
}

type maintenanceEventsResponse struct {
	Events []*MaintenanceEvent `json:"data"`
}

// ListMaintenanceEventsRequest encapsulates listing maintenance events for a
// database. Branch and State are optional filters.
type ListMaintenanceEventsRequest struct {
	Organization string
	Database     string
	Branch       string
	State        string
}

// GetMaintenanceEventRequest encapsulates getting a single maintenance event.
type GetMaintenanceEventRequest struct {
	Organization string
	Database     string
	ID           string
}

// RescheduleMaintenanceEventRequest encapsulates rescheduling a maintenance
// event. ScheduledAt must be one of the event's AvailableTimes.
type RescheduleMaintenanceEventRequest struct {
	Organization string    `json:"-"`
	Database     string    `json:"-"`
	ID           string    `json:"-"`
	ScheduledAt  time.Time `json:"scheduled_at"`
}

// MaintenanceEventsService is an interface for the PlanetScale maintenance
// events API.
type MaintenanceEventsService interface {
	List(context.Context, *ListMaintenanceEventsRequest, ...ListOption) ([]*MaintenanceEvent, error)
	Get(context.Context, *GetMaintenanceEventRequest) (*MaintenanceEvent, error)
	Reschedule(context.Context, *RescheduleMaintenanceEventRequest) (*MaintenanceEvent, error)
}

type maintenanceEventsService struct {
	client *Client
}

var _ MaintenanceEventsService = &maintenanceEventsService{}

func (s *maintenanceEventsService) List(ctx context.Context, listReq *ListMaintenanceEventsRequest, opts ...ListOption) ([]*MaintenanceEvent, error) {
	listOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}
	if listReq.Branch != "" {
		listOpts.URLValues.Set("branch", listReq.Branch)
	}
	if listReq.State != "" {
		listOpts.URLValues.Set("state", listReq.State)
	}

	req, err := s.client.newRequest(http.MethodGet, maintenanceEventsAPIPath(listReq.Organization, listReq.Database), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list maintenance events: %w", err)
	}

	resp := &maintenanceEventsResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Events, nil
}

func (s *maintenanceEventsService) Get(ctx context.Context, getReq *GetMaintenanceEventRequest) (*MaintenanceEvent, error) {
	req, err := s.client.newRequest(http.MethodGet, maintenanceEventAPIPath(getReq.Organization, getReq.Database, getReq.ID), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get maintenance event: %w", err)
	}

	event := &MaintenanceEvent{}
	if err := s.client.do(ctx, req, &event); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *maintenanceEventsService) Reschedule(ctx context.Context, rescheduleReq *RescheduleMaintenanceEventRequest) (*MaintenanceEvent, error) {
	body := *rescheduleReq
	body.ScheduledAt = body.ScheduledAt.UTC()

	req, err := s.client.newRequest(http.MethodPatch, maintenanceEventAPIPath(rescheduleReq.Organization, rescheduleReq.Database, rescheduleReq.ID), &body)
	if err != nil {
		return nil, fmt.Errorf("error creating request for reschedule maintenance event: %w", err)
	}

	event := &MaintenanceEvent{}
	if err := s.client.do(ctx, req, &event); err != nil {
		return nil, err
	}

	return event, nil
}

func maintenanceEventsAPIPath(org, db string) string {
	return path.Join("v1/organizations", org, "databases", db, "maintenance-events")
}

func maintenanceEventAPIPath(org, db, id string) string {
	return path.Join(maintenanceEventsAPIPath(org, db), id)
}
