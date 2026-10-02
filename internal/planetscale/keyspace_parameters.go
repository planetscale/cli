package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// VitessParameter is a VTTablet or MySQL parameter that can be set on a keyspace.
type VitessParameter struct {
	Name          string   `json:"name"`
	Title         string   `json:"title,omitempty"`
	Component     string   `json:"component"`
	Description   string   `json:"description"`
	Category      string   `json:"category"`
	ParameterType string   `json:"parameter_type"`
	DefaultValue  *string  `json:"default_value"`
	Value         *string  `json:"value"`
	Override      bool     `json:"override"`
	Min           any      `json:"min,omitempty"`
	Max           any      `json:"max,omitempty"`
	Options       []string `json:"options,omitempty"`
}

// KeyspaceParameters are the parameters of a keyspace, grouped by component.
type KeyspaceParameters struct {
	VTTablet []*VitessParameter `json:"vttablet"`
	MySQL    []*VitessParameter `json:"mysqld"`
}

// KeyspaceConfigChange is a request to change a keyspace's parameters.
type KeyspaceConfigChange struct {
	ID              string             `json:"id"`
	State           string             `json:"state"`
	ChangeType      string             `json:"change_type"`
	KeyspaceName    string             `json:"keyspace_name"`
	PreviousOptions map[string]*string `json:"previous_options"`
	NewOptions      map[string]*string `json:"new_options"`
	ErrorMessage    *string            `json:"error_message"`
	QueuedUntil     *time.Time         `json:"queued_until"`
	StartedAt       *time.Time         `json:"started_at"`
	CompletedAt     *time.Time         `json:"completed_at"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	Actor           *Actor             `json:"actor"`
}

type ListKeyspaceParametersRequest struct {
	Organization string
	Database     string
	Branch       string
	Keyspace     string
}

// CreateKeyspaceConfigChangeRequest creates a draft change for one component.
// A nil option value resets that parameter to its default.
type CreateKeyspaceConfigChangeRequest struct {
	Organization string             `json:"-"`
	Database     string             `json:"-"`
	Branch       string             `json:"-"`
	Keyspace     string             `json:"-"`
	ChangeType   string             `json:"change_type"`
	Options      map[string]*string `json:"options"`
}

// SubmitConfigChangesRequest applies draft config changes on a branch.
type SubmitConfigChangesRequest struct {
	Organization string   `json:"-"`
	Database     string   `json:"-"`
	Branch       string   `json:"-"`
	IDs          []string `json:"ids"`
}

type ListKeyspaceConfigChangesRequest struct {
	Organization string
	Database     string
	Branch       string
	Keyspace     string
	Page         int
	PerPage      int
}

type GetKeyspaceConfigChangeRequest struct {
	Organization string
	Database     string
	Branch       string
	Keyspace     string
	ID           string
}

type CancelKeyspaceConfigChangeRequest struct {
	Organization string
	Database     string
	Branch       string
	Keyspace     string
	ID           string
}

type branchVitessParametersResponse struct {
	Keyspaces map[string]*KeyspaceParameters `json:"keyspaces"`
}

type keyspaceConfigChangesResponse struct {
	ConfigChanges []*KeyspaceConfigChange `json:"data"`
}

// ListParameters returns the VTTablet and MySQL parameters of a keyspace, or
// nil when the branch has no keyspace with that name.
func (s *keyspacesService) ListParameters(ctx context.Context, listReq *ListKeyspaceParametersRequest) (*KeyspaceParameters, error) {
	req, err := s.client.newRequest(http.MethodGet, path.Join(databaseBranchAPIPath(listReq.Organization, listReq.Database, listReq.Branch), "vitess-parameters"), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp := &branchVitessParametersResponse{}
	if err := s.client.do(ctx, req, resp); err != nil {
		return nil, err
	}

	return resp.Keyspaces[listReq.Keyspace], nil
}

func (s *keyspacesService) CreateConfigChange(ctx context.Context, createReq *CreateKeyspaceConfigChangeRequest) (*KeyspaceConfigChange, error) {
	req, err := s.client.newRequest(http.MethodPost, keyspaceConfigChangesAPIPath(createReq.Organization, createReq.Database, createReq.Branch, createReq.Keyspace), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	change := &KeyspaceConfigChange{}
	if err := s.client.do(ctx, req, change); err != nil {
		return nil, err
	}

	return change, nil
}

func (s *keyspacesService) SubmitConfigChanges(ctx context.Context, submitReq *SubmitConfigChangesRequest) error {
	req, err := s.client.newRequest(http.MethodPost, path.Join(databaseBranchAPIPath(submitReq.Organization, submitReq.Database, submitReq.Branch), "config-changes", "submit"), submitReq)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	return s.client.do(ctx, req, nil)
}

func (s *keyspacesService) ListConfigChanges(ctx context.Context, listReq *ListKeyspaceConfigChangesRequest) ([]*KeyspaceConfigChange, error) {
	values := defaultListOptions(WithPage(listReq.Page), WithPerPage(listReq.PerPage))
	req, err := s.client.newRequest(http.MethodGet, keyspaceConfigChangesAPIPath(listReq.Organization, listReq.Database, listReq.Branch, listReq.Keyspace), nil, WithQueryParams(*values.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp := &keyspaceConfigChangesResponse{}
	if err := s.client.do(ctx, req, resp); err != nil {
		return nil, err
	}

	return resp.ConfigChanges, nil
}

func (s *keyspacesService) GetConfigChange(ctx context.Context, getReq *GetKeyspaceConfigChangeRequest) (*KeyspaceConfigChange, error) {
	req, err := s.client.newRequest(http.MethodGet, path.Join(keyspaceConfigChangesAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.Keyspace), getReq.ID), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	change := &KeyspaceConfigChange{}
	if err := s.client.do(ctx, req, change); err != nil {
		return nil, err
	}

	return change, nil
}

func (s *keyspacesService) CancelConfigChange(ctx context.Context, cancelReq *CancelKeyspaceConfigChangeRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, path.Join(keyspaceConfigChangesAPIPath(cancelReq.Organization, cancelReq.Database, cancelReq.Branch, cancelReq.Keyspace), cancelReq.ID), nil)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	return s.client.do(ctx, req, nil)
}

func keyspaceConfigChangesAPIPath(org, db, branch, keyspace string) string {
	return path.Join(keyspaceAPIPath(org, db, branch, keyspace), "config-changes")
}
