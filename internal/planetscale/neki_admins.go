package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// NekiAdminSKU represents the size of a Neki admin.
type NekiAdminSKU struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	CPU         string `json:"cpu"`
	RAM         int64  `json:"ram"`
	SortOrder   int    `json:"sort_order"`
}

// NekiAdmin represents the failover and recovery admin for a Neki cluster.
type NekiAdmin struct {
	Type      string        `json:"type,omitempty"`
	ID        string        `json:"id"`
	SKU       *NekiAdminSKU `json:"sku,omitempty"`
	AdminSize string        `json:"admin_size"`
	State     string        `json:"state"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type GetNekiAdminRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

type UpdateNekiAdminRequest struct {
	Organization string                       `json:"-"`
	Database     string                       `json:"-"`
	Branch       string                       `json:"-"`
	AdminSize    *string                      `json:"admin_size,omitempty"`
	Parameters   map[string]map[string]string `json:"parameters,omitempty"`
}

type ListNekiAdminParametersRequest = GetNekiAdminRequest

type ListNekiAdminSizeSKUsRequest = GetNekiAdminRequest

type NekiAdminChange struct {
	Type                         string                       `json:"type,omitempty"`
	ID                           string                       `json:"id"`
	State                        string                       `json:"state"`
	StartedAt                    *time.Time                   `json:"started_at"`
	CompletedAt                  *time.Time                   `json:"completed_at"`
	CreatedAt                    time.Time                    `json:"created_at"`
	UpdatedAt                    time.Time                    `json:"updated_at"`
	AdminSize                    string                       `json:"admin_size"`
	AdminSizeDisplayName         string                       `json:"admin_size_display_name"`
	PreviousAdminSize            string                       `json:"previous_admin_size"`
	PreviousAdminSizeDisplayName string                       `json:"previous_admin_size_display_name"`
	Parameters                   map[string]map[string]string `json:"parameters"`
	PreviousParameters           map[string]map[string]string `json:"previous_parameters"`
	Actor                        *Actor                       `json:"actor,omitempty"`
}

type ListNekiAdminChangesRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Period       string `json:"-"`
	CompletedAt  string `json:"-"`
	Page         int    `json:"-"`
	PerPage      int    `json:"-"`
}

type GetNekiAdminChangeRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Change       string `json:"-"`
}

type CancelNekiAdminChangeRequest = GetNekiAdminChangeRequest

type nekiAdminChangesResponse struct {
	Changes []*NekiAdminChange `json:"data"`
}

// NekiAdminsService manages the admin belonging to a Neki branch.
// The admin is created and deleted with the cluster; this service exposes
// get and update only.
type NekiAdminsService interface {
	Get(context.Context, *GetNekiAdminRequest) (*NekiAdmin, error)
	Update(context.Context, *UpdateNekiAdminRequest) (*NekiAdmin, error)
	ListParameters(context.Context, *ListNekiAdminParametersRequest) ([]*NekiParameter, error)
	ListSizeSKUs(context.Context, *ListNekiAdminSizeSKUsRequest) ([]*NekiAdminSKU, error)
	ListChanges(context.Context, *ListNekiAdminChangesRequest) ([]*NekiAdminChange, error)
	GetChange(context.Context, *GetNekiAdminChangeRequest) (*NekiAdminChange, error)
	CancelChange(context.Context, *CancelNekiAdminChangeRequest) error
}

type nekiAdminsService struct{ client *Client }

var _ NekiAdminsService = &nekiAdminsService{}

func (s *nekiAdminsService) Get(ctx context.Context, r *GetNekiAdminRequest) (*NekiAdmin, error) {
	request, err := s.client.newRequest(http.MethodGet, nekiAdminAPIPath(r.Organization, r.Database, r.Branch), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to get admin: %w", err)
	}
	admin := &NekiAdmin{}
	if err := s.client.do(ctx, request, admin); err != nil {
		return nil, err
	}
	return admin, nil
}

func (s *nekiAdminsService) Update(ctx context.Context, r *UpdateNekiAdminRequest) (*NekiAdmin, error) {
	request, err := s.client.newRequest(http.MethodPatch, nekiAdminAPIPath(r.Organization, r.Database, r.Branch), r)
	if err != nil {
		return nil, fmt.Errorf("error creating request to update admin: %w", err)
	}
	admin := &NekiAdmin{}
	if err := s.client.do(ctx, request, admin); err != nil {
		return nil, err
	}
	return admin, nil
}

func (s *nekiAdminsService) ListSizeSKUs(ctx context.Context, r *ListNekiAdminSizeSKUsRequest) ([]*NekiAdminSKU, error) {
	request, err := s.client.newRequest(http.MethodGet, nekiAdminSizeSKUsAPIPath(r.Organization, r.Database, r.Branch), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to list admin size SKUs: %w", err)
	}
	skus := []*NekiAdminSKU{}
	if err := s.client.do(ctx, request, &skus); err != nil {
		return nil, err
	}
	return skus, nil
}

func (s *nekiAdminsService) ListParameters(ctx context.Context, r *ListNekiAdminParametersRequest) ([]*NekiParameter, error) {
	request, err := s.client.newRequest(http.MethodGet, path.Join(nekiAdminAPIPath(r.Organization, r.Database, r.Branch), "parameters"), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to list admin parameters: %w", err)
	}
	parameters := []*NekiParameter{}
	if err := s.client.do(ctx, request, &parameters); err != nil {
		return nil, err
	}
	return parameters, nil
}

func (s *nekiAdminsService) ListChanges(ctx context.Context, r *ListNekiAdminChangesRequest) ([]*NekiAdminChange, error) {
	values := defaultListOptions(WithPeriod(r.Period), WithPage(r.Page), WithPerPage(r.PerPage))
	if r.CompletedAt != "" {
		values.URLValues.Set("completed_at", r.CompletedAt)
	}
	request, err := s.client.newRequest(http.MethodGet, nekiAdminChangesAPIPath(r.Organization, r.Database, r.Branch), nil, WithQueryParams(*values.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request to list admin changes: %w", err)
	}
	response := &nekiAdminChangesResponse{}
	if err := s.client.do(ctx, request, response); err != nil {
		return nil, err
	}
	return response.Changes, nil
}

func (s *nekiAdminsService) GetChange(ctx context.Context, r *GetNekiAdminChangeRequest) (*NekiAdminChange, error) {
	request, err := s.client.newRequest(http.MethodGet, nekiAdminChangeAPIPath(r.Organization, r.Database, r.Branch, r.Change), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to get admin change: %w", err)
	}
	change := &NekiAdminChange{}
	if err := s.client.do(ctx, request, change); err != nil {
		return nil, err
	}
	return change, nil
}

func (s *nekiAdminsService) CancelChange(ctx context.Context, r *CancelNekiAdminChangeRequest) error {
	request, err := s.client.newRequest(http.MethodDelete, nekiAdminChangeAPIPath(r.Organization, r.Database, r.Branch, r.Change), nil)
	if err != nil {
		return fmt.Errorf("error creating request to cancel admin change: %w", err)
	}
	return s.client.do(ctx, request, nil)
}

func nekiAdminSizeSKUsAPIPath(org, database, branch string) string {
	return path.Join(databaseBranchAPIPath(org, database, branch), "admin-size-skus")
}

func nekiAdminAPIPath(org, database, branch string) string {
	return path.Join(databaseBranchAPIPath(org, database, branch), "admin")
}

func nekiAdminChangesAPIPath(org, database, branch string) string {
	return path.Join(nekiAdminAPIPath(org, database, branch), "changes")
}

func nekiAdminChangeAPIPath(org, database, branch, change string) string {
	return path.Join(nekiAdminChangesAPIPath(org, database, branch), change)
}
