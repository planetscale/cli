package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// NekiRouterSKU represents the size of a Neki router.
type NekiRouterSKU struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	CPU         string `json:"cpu"`
	RAM         int64  `json:"ram"`
	SortOrder   int    `json:"sort_order"`
	Rate        *int64 `json:"rate,omitempty"`
}

// NekiRouter represents a router group for a Neki branch.
type NekiRouter struct {
	Type                 string         `json:"type,omitempty"`
	Name                 string         `json:"name"`
	Default              bool           `json:"default"`
	SKU                  *NekiRouterSKU `json:"sku,omitempty"`
	ReplicasPerCell      int            `json:"replicas_per_cell"`
	Autoscaling          bool           `json:"autoscaling"`
	MaxReplicasPerCell   *int           `json:"max_replicas_per_cell"`
	TargetCPUUtilization *int           `json:"target_cpu_utilization"`
	State                string         `json:"state"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

type ListNekiRoutersRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

type GetNekiRouterRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Router       string `json:"-"`
}

type CreateNekiRouterRequest struct {
	Organization    string `json:"-"`
	Database        string `json:"-"`
	Branch          string `json:"-"`
	Name            string `json:"name"`
	RouterSize      string `json:"router_size,omitempty"`
	ReplicasPerCell *int   `json:"replicas_per_cell,omitempty"`
}

// UpdateNekiRouterRequest changes a router through the change-request flow;
// only set fields are sent.
type UpdateNekiRouterRequest struct {
	Organization         string                       `json:"-"`
	Database             string                       `json:"-"`
	Branch               string                       `json:"-"`
	Router               string                       `json:"-"`
	RouterSize           *string                      `json:"router_size,omitempty"`
	ReplicasPerCell      *int                         `json:"replicas_per_cell,omitempty"`
	Autoscaling          *bool                        `json:"autoscaling,omitempty"`
	MaxReplicasPerCell   *int                         `json:"max_replicas_per_cell,omitempty"`
	TargetCPUUtilization *int                         `json:"target_cpu_utilization,omitempty"`
	Parameters           map[string]map[string]string `json:"parameters,omitempty"`
}

type DeleteNekiRouterRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Router       string `json:"-"`
}

type ListNekiRouterParametersRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Router       string `json:"-"`
}

type ListNekiRouterSizeSKUsRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Rates        bool   `json:"-"`
}

type NekiRouterChange struct {
	Type                          string                       `json:"type,omitempty"`
	ID                            string                       `json:"id"`
	State                         string                       `json:"state"`
	StartedAt                     *time.Time                   `json:"started_at"`
	CompletedAt                   *time.Time                   `json:"completed_at"`
	CreatedAt                     time.Time                    `json:"created_at"`
	UpdatedAt                     time.Time                    `json:"updated_at"`
	RouterSize                    string                       `json:"router_size"`
	RouterSizeDisplayName         string                       `json:"router_size_display_name"`
	ReplicasPerCell               int                          `json:"replicas_per_cell"`
	Autoscaling                   *bool                        `json:"autoscaling"`
	MaxReplicasPerCell            *int                         `json:"max_replicas_per_cell"`
	TargetCPUUtilization          *int                         `json:"target_cpu_utilization"`
	Parameters                    map[string]map[string]string `json:"parameters"`
	PreviousRouterSize            string                       `json:"previous_router_size"`
	PreviousRouterSizeDisplayName string                       `json:"previous_router_size_display_name"`
	PreviousReplicasPerCell       int                          `json:"previous_replicas_per_cell"`
	PreviousAutoscaling           *bool                        `json:"previous_autoscaling"`
	PreviousMaxReplicasPerCell    *int                         `json:"previous_max_replicas_per_cell"`
	PreviousTargetCPUUtilization  *int                         `json:"previous_target_cpu_utilization"`
	PreviousParameters            map[string]map[string]string `json:"previous_parameters"`
	Actor                         *Actor                       `json:"actor,omitempty"`
}

type ListNekiRouterChangesRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Router       string `json:"-"`
	Period       string `json:"-"`
	CompletedAt  string `json:"-"`
	Page         int    `json:"-"`
	PerPage      int    `json:"-"`
}

type GetNekiRouterChangeRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Router       string `json:"-"`
	Change       string `json:"-"`
}

type CancelNekiRouterChangeRequest = GetNekiRouterChangeRequest

type nekiRouterChangesResponse struct {
	Changes []*NekiRouterChange `json:"data"`
}

// NekiRoutersService manages router groups belonging to Neki branches.
type NekiRoutersService interface {
	List(context.Context, *ListNekiRoutersRequest) ([]*NekiRouter, error)
	Get(context.Context, *GetNekiRouterRequest) (*NekiRouter, error)
	Create(context.Context, *CreateNekiRouterRequest) (*NekiRouter, error)
	Update(context.Context, *UpdateNekiRouterRequest) (*NekiRouter, error)
	Delete(context.Context, *DeleteNekiRouterRequest) error
	ListParameters(context.Context, *ListNekiRouterParametersRequest) ([]*NekiParameter, error)
	ListSizeSKUs(context.Context, *ListNekiRouterSizeSKUsRequest) ([]*NekiRouterSKU, error)
	ListChanges(context.Context, *ListNekiRouterChangesRequest) ([]*NekiRouterChange, error)
	GetChange(context.Context, *GetNekiRouterChangeRequest) (*NekiRouterChange, error)
	CancelChange(context.Context, *CancelNekiRouterChangeRequest) error
}

type nekiRoutersService struct{ client *Client }

var _ NekiRoutersService = &nekiRoutersService{}

func (s *nekiRoutersService) List(ctx context.Context, r *ListNekiRoutersRequest) ([]*NekiRouter, error) {
	request, err := s.client.newRequest(http.MethodGet, nekiRoutersAPIPath(r.Organization, r.Database, r.Branch), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to list routers: %w", err)
	}
	routers := []*NekiRouter{}
	if err := s.client.do(ctx, request, &routers); err != nil {
		return nil, err
	}
	return routers, nil
}

func (s *nekiRoutersService) Get(ctx context.Context, r *GetNekiRouterRequest) (*NekiRouter, error) {
	request, err := s.client.newRequest(http.MethodGet, nekiRouterAPIPath(r.Organization, r.Database, r.Branch, r.Router), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to get router: %w", err)
	}
	router := &NekiRouter{}
	if err := s.client.do(ctx, request, router); err != nil {
		return nil, err
	}
	return router, nil
}

func (s *nekiRoutersService) Create(ctx context.Context, r *CreateNekiRouterRequest) (*NekiRouter, error) {
	request, err := s.client.newRequest(http.MethodPost, nekiRoutersAPIPath(r.Organization, r.Database, r.Branch), r)
	if err != nil {
		return nil, fmt.Errorf("error creating request to create router: %w", err)
	}
	router := &NekiRouter{}
	if err := s.client.do(ctx, request, router); err != nil {
		return nil, err
	}
	return router, nil
}

func (s *nekiRoutersService) Update(ctx context.Context, r *UpdateNekiRouterRequest) (*NekiRouter, error) {
	request, err := s.client.newRequest(http.MethodPatch, nekiRouterAPIPath(r.Organization, r.Database, r.Branch, r.Router), r)
	if err != nil {
		return nil, fmt.Errorf("error creating request to update router: %w", err)
	}
	router := &NekiRouter{}
	if err := s.client.do(ctx, request, router); err != nil {
		return nil, err
	}
	return router, nil
}

func (s *nekiRoutersService) Delete(ctx context.Context, r *DeleteNekiRouterRequest) error {
	request, err := s.client.newRequest(http.MethodDelete, nekiRouterAPIPath(r.Organization, r.Database, r.Branch, r.Router), nil)
	if err != nil {
		return fmt.Errorf("error creating request to delete router: %w", err)
	}
	return s.client.do(ctx, request, nil)
}

func (s *nekiRoutersService) ListSizeSKUs(ctx context.Context, r *ListNekiRouterSizeSKUsRequest) ([]*NekiRouterSKU, error) {
	opts := []RequestOption{}
	if r.Rates {
		opts = append(opts, WithQueryParams(*defaultListOptions(WithRates()).URLValues))
	}
	request, err := s.client.newRequest(http.MethodGet, nekiRouterSizeSKUsAPIPath(r.Organization, r.Database, r.Branch), nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating request to list router size SKUs: %w", err)
	}
	skus := []*NekiRouterSKU{}
	if err := s.client.do(ctx, request, &skus); err != nil {
		return nil, err
	}
	return skus, nil
}

func (s *nekiRoutersService) ListParameters(ctx context.Context, r *ListNekiRouterParametersRequest) ([]*NekiParameter, error) {
	request, err := s.client.newRequest(http.MethodGet, path.Join(nekiRouterAPIPath(r.Organization, r.Database, r.Branch, r.Router), "parameters"), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to list router parameters: %w", err)
	}
	parameters := []*NekiParameter{}
	if err := s.client.do(ctx, request, &parameters); err != nil {
		return nil, err
	}
	return parameters, nil
}

func (s *nekiRoutersService) ListChanges(ctx context.Context, r *ListNekiRouterChangesRequest) ([]*NekiRouterChange, error) {
	values := defaultListOptions(WithPeriod(r.Period), WithPage(r.Page), WithPerPage(r.PerPage))
	if r.CompletedAt != "" {
		values.URLValues.Set("completed_at", r.CompletedAt)
	}
	request, err := s.client.newRequest(http.MethodGet, nekiRouterChangesAPIPath(r.Organization, r.Database, r.Branch, r.Router), nil, WithQueryParams(*values.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request to list router changes: %w", err)
	}
	response := &nekiRouterChangesResponse{}
	if err := s.client.do(ctx, request, response); err != nil {
		return nil, err
	}
	return response.Changes, nil
}

func (s *nekiRoutersService) GetChange(ctx context.Context, r *GetNekiRouterChangeRequest) (*NekiRouterChange, error) {
	request, err := s.client.newRequest(http.MethodGet, nekiRouterChangeAPIPath(r.Organization, r.Database, r.Branch, r.Router, r.Change), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to get router change: %w", err)
	}
	change := &NekiRouterChange{}
	if err := s.client.do(ctx, request, change); err != nil {
		return nil, err
	}
	return change, nil
}

func (s *nekiRoutersService) CancelChange(ctx context.Context, r *CancelNekiRouterChangeRequest) error {
	request, err := s.client.newRequest(http.MethodDelete, nekiRouterChangeAPIPath(r.Organization, r.Database, r.Branch, r.Router, r.Change), nil)
	if err != nil {
		return fmt.Errorf("error creating request to cancel router change: %w", err)
	}
	return s.client.do(ctx, request, nil)
}

func nekiRouterSizeSKUsAPIPath(org, database, branch string) string {
	return path.Join(databaseBranchAPIPath(org, database, branch), "router-size-skus")
}

func nekiRoutersAPIPath(org, database, branch string) string {
	return path.Join(databaseBranchAPIPath(org, database, branch), "routers")
}

func nekiRouterAPIPath(org, database, branch, router string) string {
	return path.Join(nekiRoutersAPIPath(org, database, branch), router)
}

func nekiRouterChangesAPIPath(org, database, branch, router string) string {
	return path.Join(nekiRouterAPIPath(org, database, branch, router), "changes")
}

func nekiRouterChangeAPIPath(org, database, branch, router, change string) string {
	return path.Join(nekiRouterChangesAPIPath(org, database, branch, router), change)
}
