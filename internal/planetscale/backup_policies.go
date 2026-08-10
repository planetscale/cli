package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// BackupPolicy represents a scheduled backup policy for a database.
type BackupPolicy struct {
	ID             string     `json:"id"`
	DisplayName    string     `json:"display_name"`
	Name           string     `json:"name"`
	Target         string     `json:"target"`
	RetentionValue int        `json:"retention_value"`
	RetentionUnit  string     `json:"retention_unit"`
	FrequencyValue int        `json:"frequency_value"`
	FrequencyUnit  string     `json:"frequency_unit"`
	ScheduleTime   string     `json:"schedule_time"`
	ScheduleDay    *int       `json:"schedule_day"`
	ScheduleWeek   *int       `json:"schedule_week"`
	Required       bool       `json:"required"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	LastRanAt      *time.Time `json:"last_ran_at"`
	NextRunAt      *time.Time `json:"next_run_at"`
}

type backupPoliciesResponse struct {
	Policies []*BackupPolicy `json:"data"`
}

// ListBackupPoliciesRequest encapsulates listing backup policies for a database.
type ListBackupPoliciesRequest struct {
	Organization string
	Database     string
}

// GetBackupPolicyRequest encapsulates getting a single backup policy.
type GetBackupPolicyRequest struct {
	Organization string
	Database     string
	Policy       string
}

// CreateBackupPolicyRequest encapsulates creating a backup policy.
type CreateBackupPolicyRequest struct {
	Organization   string `json:"-"`
	Database       string `json:"-"`
	Name           string `json:"name,omitempty"`
	Target         string `json:"target"`
	RetentionValue int    `json:"retention_value"`
	RetentionUnit  string `json:"retention_unit"`
	FrequencyValue int    `json:"frequency_value"`
	FrequencyUnit  string `json:"frequency_unit"`
	ScheduleTime   string `json:"schedule_time"`
	ScheduleDay    *int   `json:"schedule_day,omitempty"`
	ScheduleWeek   *int   `json:"schedule_week,omitempty"`
}

// UpdateBackupPolicyRequest encapsulates updating a backup policy.
type UpdateBackupPolicyRequest struct {
	Organization   string  `json:"-"`
	Database       string  `json:"-"`
	Policy         string  `json:"-"`
	Name           *string `json:"name,omitempty"`
	Target         *string `json:"target,omitempty"`
	RetentionValue *int    `json:"retention_value,omitempty"`
	RetentionUnit  *string `json:"retention_unit,omitempty"`
	FrequencyValue *int    `json:"frequency_value,omitempty"`
	FrequencyUnit  *string `json:"frequency_unit,omitempty"`
	ScheduleTime   *string `json:"schedule_time,omitempty"`
	ScheduleDay    *int    `json:"schedule_day,omitempty"`
	ScheduleWeek   *int    `json:"schedule_week,omitempty"`
}

// DeleteBackupPolicyRequest encapsulates deleting a backup policy.
type DeleteBackupPolicyRequest struct {
	Organization string
	Database     string
	Policy       string
}

// BackupPoliciesService is an interface for the PlanetScale backup policies API.
type BackupPoliciesService interface {
	List(context.Context, *ListBackupPoliciesRequest, ...ListOption) ([]*BackupPolicy, error)
	Get(context.Context, *GetBackupPolicyRequest) (*BackupPolicy, error)
	Create(context.Context, *CreateBackupPolicyRequest) (*BackupPolicy, error)
	Update(context.Context, *UpdateBackupPolicyRequest) (*BackupPolicy, error)
	Delete(context.Context, *DeleteBackupPolicyRequest) error
}

type backupPoliciesService struct {
	client *Client
}

var _ BackupPoliciesService = &backupPoliciesService{}

func NewBackupPoliciesService(client *Client) *backupPoliciesService {
	return &backupPoliciesService{client: client}
}

func (s *backupPoliciesService) List(ctx context.Context, listReq *ListBackupPoliciesRequest, opts ...ListOption) ([]*BackupPolicy, error) {
	listOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}

	req, err := s.client.newRequest(http.MethodGet, backupPoliciesAPIPath(listReq.Organization, listReq.Database), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list backup policies: %w", err)
	}

	resp := &backupPoliciesResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Policies, nil
}

func (s *backupPoliciesService) Get(ctx context.Context, getReq *GetBackupPolicyRequest) (*BackupPolicy, error) {
	req, err := s.client.newRequest(http.MethodGet, backupPolicyAPIPath(getReq.Organization, getReq.Database, getReq.Policy), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get backup policy: %w", err)
	}

	policy := &BackupPolicy{}
	if err := s.client.do(ctx, req, &policy); err != nil {
		return nil, err
	}

	return policy, nil
}

func (s *backupPoliciesService) Create(ctx context.Context, createReq *CreateBackupPolicyRequest) (*BackupPolicy, error) {
	req, err := s.client.newRequest(http.MethodPost, backupPoliciesAPIPath(createReq.Organization, createReq.Database), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for create backup policy: %w", err)
	}

	policy := &BackupPolicy{}
	if err := s.client.do(ctx, req, &policy); err != nil {
		return nil, err
	}

	return policy, nil
}

func (s *backupPoliciesService) Update(ctx context.Context, updateReq *UpdateBackupPolicyRequest) (*BackupPolicy, error) {
	req, err := s.client.newRequest(http.MethodPatch, backupPolicyAPIPath(updateReq.Organization, updateReq.Database, updateReq.Policy), updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for update backup policy: %w", err)
	}

	policy := &BackupPolicy{}
	if err := s.client.do(ctx, req, &policy); err != nil {
		return nil, err
	}

	return policy, nil
}

func (s *backupPoliciesService) Delete(ctx context.Context, deleteReq *DeleteBackupPolicyRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, backupPolicyAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.Policy), nil)
	if err != nil {
		return fmt.Errorf("error creating request for delete backup policy: %w", err)
	}

	return s.client.do(ctx, req, nil)
}

func backupPoliciesAPIPath(org, db string) string {
	return path.Join("v1/organizations", org, "databases", db, "backup-policies")
}

func backupPolicyAPIPath(org, db, policy string) string {
	return path.Join(backupPoliciesAPIPath(org, db), policy)
}
