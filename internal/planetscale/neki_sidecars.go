package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// NekiSidecar represents the connection-pool sidecar for one Neki configuration profile.
type NekiSidecar struct {
	Type                 string    `json:"type,omitempty"`
	ID                   string    `json:"id"`
	ConfigurationProfile string    `json:"configuration_profile"`
	State                string    `json:"state"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type ListNekiSidecarsRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

type GetNekiSidecarRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Sidecar      string `json:"-"`
}

type UpdateNekiSidecarRequest struct {
	Organization string                       `json:"-"`
	Database     string                       `json:"-"`
	Branch       string                       `json:"-"`
	Sidecar      string                       `json:"-"`
	Parameters   map[string]map[string]string `json:"parameters,omitempty"`
}

type ListNekiSidecarParametersRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Sidecar      string `json:"-"`
}

type NekiSidecarChange struct {
	Type               string                       `json:"type,omitempty"`
	ID                 string                       `json:"id"`
	State              string                       `json:"state"`
	StartedAt          *time.Time                   `json:"started_at"`
	CompletedAt        *time.Time                   `json:"completed_at"`
	CreatedAt          time.Time                    `json:"created_at"`
	UpdatedAt          time.Time                    `json:"updated_at"`
	Parameters         map[string]map[string]string `json:"parameters"`
	PreviousParameters map[string]map[string]string `json:"previous_parameters"`
	Actor              *Actor                       `json:"actor,omitempty"`
}

type ListNekiSidecarChangesRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Sidecar      string `json:"-"`
	Period       string `json:"-"`
	CompletedAt  string `json:"-"`
	Page         int    `json:"-"`
	PerPage      int    `json:"-"`
}

type GetNekiSidecarChangeRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Sidecar      string `json:"-"`
	Change       string `json:"-"`
}

type CancelNekiSidecarChangeRequest = GetNekiSidecarChangeRequest

type nekiSidecarChangesResponse struct {
	Changes []*NekiSidecarChange `json:"data"`
}

// NekiSidecarsService manages sidecars belonging to Neki branches.
// Sidecars are created and deleted with their configuration profile; this
// service exposes list, get, and update only.
type NekiSidecarsService interface {
	List(context.Context, *ListNekiSidecarsRequest) ([]*NekiSidecar, error)
	Get(context.Context, *GetNekiSidecarRequest) (*NekiSidecar, error)
	Update(context.Context, *UpdateNekiSidecarRequest) (*NekiSidecar, error)
	ListParameters(context.Context, *ListNekiSidecarParametersRequest) ([]*NekiParameter, error)
	ListChanges(context.Context, *ListNekiSidecarChangesRequest) ([]*NekiSidecarChange, error)
	GetChange(context.Context, *GetNekiSidecarChangeRequest) (*NekiSidecarChange, error)
	CancelChange(context.Context, *CancelNekiSidecarChangeRequest) error
}

type nekiSidecarsService struct{ client *Client }

var _ NekiSidecarsService = &nekiSidecarsService{}

func (s *nekiSidecarsService) List(ctx context.Context, r *ListNekiSidecarsRequest) ([]*NekiSidecar, error) {
	request, err := s.client.newRequest(http.MethodGet, nekiSidecarsAPIPath(r.Organization, r.Database, r.Branch), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to list sidecars: %w", err)
	}
	sidecars := []*NekiSidecar{}
	if err := s.client.do(ctx, request, &sidecars); err != nil {
		return nil, err
	}
	return sidecars, nil
}

func (s *nekiSidecarsService) Get(ctx context.Context, r *GetNekiSidecarRequest) (*NekiSidecar, error) {
	request, err := s.client.newRequest(http.MethodGet, nekiSidecarAPIPath(r.Organization, r.Database, r.Branch, r.Sidecar), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to get sidecar: %w", err)
	}
	sidecar := &NekiSidecar{}
	if err := s.client.do(ctx, request, sidecar); err != nil {
		return nil, err
	}
	return sidecar, nil
}

func (s *nekiSidecarsService) Update(ctx context.Context, r *UpdateNekiSidecarRequest) (*NekiSidecar, error) {
	request, err := s.client.newRequest(http.MethodPatch, nekiSidecarAPIPath(r.Organization, r.Database, r.Branch, r.Sidecar), r)
	if err != nil {
		return nil, fmt.Errorf("error creating request to update sidecar: %w", err)
	}
	sidecar := &NekiSidecar{}
	if err := s.client.do(ctx, request, sidecar); err != nil {
		return nil, err
	}
	return sidecar, nil
}

func (s *nekiSidecarsService) ListParameters(ctx context.Context, r *ListNekiSidecarParametersRequest) ([]*NekiParameter, error) {
	request, err := s.client.newRequest(http.MethodGet, path.Join(nekiSidecarAPIPath(r.Organization, r.Database, r.Branch, r.Sidecar), "parameters"), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to list sidecar parameters: %w", err)
	}
	parameters := []*NekiParameter{}
	if err := s.client.do(ctx, request, &parameters); err != nil {
		return nil, err
	}
	return parameters, nil
}

func (s *nekiSidecarsService) ListChanges(ctx context.Context, r *ListNekiSidecarChangesRequest) ([]*NekiSidecarChange, error) {
	values := defaultListOptions(WithPeriod(r.Period), WithPage(r.Page), WithPerPage(r.PerPage))
	if r.CompletedAt != "" {
		values.URLValues.Set("completed_at", r.CompletedAt)
	}
	request, err := s.client.newRequest(http.MethodGet, nekiSidecarChangesAPIPath(r.Organization, r.Database, r.Branch, r.Sidecar), nil, WithQueryParams(*values.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request to list sidecar changes: %w", err)
	}
	response := &nekiSidecarChangesResponse{}
	if err := s.client.do(ctx, request, response); err != nil {
		return nil, err
	}
	return response.Changes, nil
}

func (s *nekiSidecarsService) GetChange(ctx context.Context, r *GetNekiSidecarChangeRequest) (*NekiSidecarChange, error) {
	request, err := s.client.newRequest(http.MethodGet, nekiSidecarChangeAPIPath(r.Organization, r.Database, r.Branch, r.Sidecar, r.Change), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to get sidecar change: %w", err)
	}
	change := &NekiSidecarChange{}
	if err := s.client.do(ctx, request, change); err != nil {
		return nil, err
	}
	return change, nil
}

func (s *nekiSidecarsService) CancelChange(ctx context.Context, r *CancelNekiSidecarChangeRequest) error {
	request, err := s.client.newRequest(http.MethodDelete, nekiSidecarChangeAPIPath(r.Organization, r.Database, r.Branch, r.Sidecar, r.Change), nil)
	if err != nil {
		return fmt.Errorf("error creating request to cancel sidecar change: %w", err)
	}
	return s.client.do(ctx, request, nil)
}

func nekiSidecarsAPIPath(org, database, branch string) string {
	return path.Join(databaseBranchAPIPath(org, database, branch), "sidecars")
}

func nekiSidecarAPIPath(org, database, branch, sidecar string) string {
	return path.Join(nekiSidecarsAPIPath(org, database, branch), sidecar)
}

func nekiSidecarChangesAPIPath(org, database, branch, sidecar string) string {
	return path.Join(nekiSidecarAPIPath(org, database, branch, sidecar), "changes")
}

func nekiSidecarChangeAPIPath(org, database, branch, sidecar, change string) string {
	return path.Join(nekiSidecarChangesAPIPath(org, database, branch, sidecar), change)
}
