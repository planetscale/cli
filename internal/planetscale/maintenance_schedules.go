package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// MaintenanceSchedule represents a database maintenance schedule. Available
// for PostgreSQL databases and for Vitess databases on Enterprise plans.
type MaintenanceSchedule struct {
	ID                         string     `json:"id"`
	Name                       string     `json:"name"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
	LastWindowDatetime         time.Time  `json:"last_window_datetime"`
	NextWindowDatetime         time.Time  `json:"next_window_datetime"`
	Duration                   int        `json:"duration"`
	Day                        int        `json:"day"`
	Hour                       int        `json:"hour"`
	Week                       int        `json:"week"`
	FrequencyValue             int        `json:"frequency_value"`
	FrequencyUnit              string     `json:"frequency_unit"`
	Enabled                    bool       `json:"enabled"`
	ExpiresAt                  *time.Time `json:"expires_at"`
	DeadlineAt                 *time.Time `json:"deadline_at"`
	Required                   bool       `json:"required"`
	PendingVitessVersionUpdate bool       `json:"pending_vitess_version_update"`
	PendingVitessVersion       *string    `json:"pending_vitess_version"`
	PendingMySQLVersionUpdate  bool       `json:"pending_mysql_version_update"`
	PendingMySQLVersion        *string    `json:"pending_mysql_version"`

	DatabaseBranch *MaintenanceBranch `json:"database_branch"`
}

// MaintenanceBranch identifies the branch a maintenance schedule or
// maintenance event belongs to. It is not a full DatabaseBranch: the API only
// returns the branch's identity here.
type MaintenanceBranch struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

// MaintenanceWindow represents a past or ongoing maintenance window for a schedule.
type MaintenanceWindow struct {
	ID         string     `json:"id"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

type maintenanceSchedulesResponse struct {
	Schedules []*MaintenanceSchedule `json:"data"`
}

type maintenanceWindowsResponse struct {
	Windows []*MaintenanceWindow `json:"data"`
}

// ListMaintenanceSchedulesRequest encapsulates listing maintenance schedules for a database.
type ListMaintenanceSchedulesRequest struct {
	Organization string
	Database     string
}

// GetMaintenanceScheduleRequest encapsulates getting a single maintenance schedule.
type GetMaintenanceScheduleRequest struct {
	Organization string
	Database     string
	Schedule     string
}

// ListMaintenanceWindowsRequest encapsulates listing windows for a maintenance schedule.
type ListMaintenanceWindowsRequest struct {
	Organization string
	Database     string
	Schedule     string
}

// UpdateMaintenanceScheduleRequest encapsulates changing when a weekly
// maintenance schedule opens. Day and Hour are in UTC; a nil field is left
// unchanged.
type UpdateMaintenanceScheduleRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Schedule     string `json:"-"`
	Day          *int   `json:"day,omitempty"`
	Hour         *int   `json:"hour,omitempty"`
}

// MaintenanceSchedulesService is an interface for the PlanetScale maintenance schedules API.
type MaintenanceSchedulesService interface {
	List(context.Context, *ListMaintenanceSchedulesRequest, ...ListOption) ([]*MaintenanceSchedule, error)
	Get(context.Context, *GetMaintenanceScheduleRequest) (*MaintenanceSchedule, error)
	Update(context.Context, *UpdateMaintenanceScheduleRequest) (*MaintenanceSchedule, error)
	ListWindows(context.Context, *ListMaintenanceWindowsRequest, ...ListOption) ([]*MaintenanceWindow, error)
}

type maintenanceSchedulesService struct {
	client *Client
}

var _ MaintenanceSchedulesService = &maintenanceSchedulesService{}

func (s *maintenanceSchedulesService) List(ctx context.Context, listReq *ListMaintenanceSchedulesRequest, opts ...ListOption) ([]*MaintenanceSchedule, error) {
	listOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}

	req, err := s.client.newRequest(http.MethodGet, maintenanceSchedulesAPIPath(listReq.Organization, listReq.Database), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list maintenance schedules: %w", err)
	}

	resp := &maintenanceSchedulesResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Schedules, nil
}

func (s *maintenanceSchedulesService) Get(ctx context.Context, getReq *GetMaintenanceScheduleRequest) (*MaintenanceSchedule, error) {
	req, err := s.client.newRequest(http.MethodGet, maintenanceScheduleAPIPath(getReq.Organization, getReq.Database, getReq.Schedule), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get maintenance schedule: %w", err)
	}

	schedule := &MaintenanceSchedule{}
	if err := s.client.do(ctx, req, &schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

func (s *maintenanceSchedulesService) Update(ctx context.Context, updateReq *UpdateMaintenanceScheduleRequest) (*MaintenanceSchedule, error) {
	req, err := s.client.newRequest(http.MethodPatch, maintenanceScheduleAPIPath(updateReq.Organization, updateReq.Database, updateReq.Schedule), updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for update maintenance schedule: %w", err)
	}

	schedule := &MaintenanceSchedule{}
	if err := s.client.do(ctx, req, &schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

func (s *maintenanceSchedulesService) ListWindows(ctx context.Context, listReq *ListMaintenanceWindowsRequest, opts ...ListOption) ([]*MaintenanceWindow, error) {
	listOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}

	req, err := s.client.newRequest(http.MethodGet, maintenanceWindowsAPIPath(listReq.Organization, listReq.Database, listReq.Schedule), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list maintenance windows: %w", err)
	}

	resp := &maintenanceWindowsResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Windows, nil
}

func maintenanceSchedulesAPIPath(org, db string) string {
	return path.Join("v1/organizations", org, "databases", db, "maintenance-schedules")
}

func maintenanceScheduleAPIPath(org, db, schedule string) string {
	return path.Join(maintenanceSchedulesAPIPath(org, db), schedule)
}

func maintenanceWindowsAPIPath(org, db, schedule string) string {
	return path.Join(maintenanceScheduleAPIPath(org, db, schedule), "windows")
}
