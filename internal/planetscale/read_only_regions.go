package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"
)

// ReadOnlyRegion represents a Vitess read-only region for a database's default branch.
type ReadOnlyRegion struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Ready       bool      `json:"ready"`
	ReadyAt     time.Time `json:"ready_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Actor       *Actor    `json:"actor"`
	Region      Region    `json:"region"`
}

type ListReadOnlyRegionsRequest struct {
	Organization string
	Database     string
}

type readOnlyRegionsResponse struct {
	Data []*ReadOnlyRegion `json:"data"`
}

// ReadOnlyRegionsService lists read-only regions for a database.
type ReadOnlyRegionsService interface {
	List(ctx context.Context, req *ListReadOnlyRegionsRequest, opts ...ListOption) ([]*ReadOnlyRegion, error)
}

type readOnlyRegionsService struct {
	client *Client
}

var _ ReadOnlyRegionsService = &readOnlyRegionsService{}

func (s *readOnlyRegionsService) List(ctx context.Context, listReq *ListReadOnlyRegionsRequest, opts ...ListOption) ([]*ReadOnlyRegion, error) {
	pathStr := path.Join(databasesAPIPath(listReq.Organization), listReq.Database, "read-only-regions")

	defaultOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(defaultOpts); err != nil {
			return nil, err
		}
	}

	req, err := s.client.newRequest(http.MethodGet, pathStr, nil, WithQueryParams(*defaultOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list read-only regions: %w", err)
	}

	resp := &readOnlyRegionsResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// FindReadOnlyRegion matches a read-only region by public id, region slug, or display name.
func FindReadOnlyRegion(regions []*ReadOnlyRegion, name string) (*ReadOnlyRegion, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("read-only region name cannot be empty")
	}

	var matches []*ReadOnlyRegion
	for _, r := range regions {
		if r.ID == name || r.Region.Slug == name || r.DisplayName == name {
			matches = append(matches, r)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("read-only region %q not found", name)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("read-only region %q matches multiple regions; use the region id", name)
	}
}
