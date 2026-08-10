package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// PostgresCIDR represents a Postgres IP restriction (CIDR restriction) entry.
type PostgresCIDR struct {
	ID          string     `json:"id"`
	Schema      string     `json:"schema"`
	Role        string     `json:"role"`
	CIDRs       []string   `json:"cidrs"`
	Description *string    `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
	Actor       Actor      `json:"actor"`
}

type postgresCIDRsResponse struct {
	CIDRs []*PostgresCIDR `json:"data"`
}

// ListPostgresCIDRsRequest encapsulates listing IP restriction entries for a database.
type ListPostgresCIDRsRequest struct {
	Organization string
	Database     string
}

// GetPostgresCIDRRequest encapsulates getting a single IP restriction entry.
type GetPostgresCIDRRequest struct {
	Organization string
	Database     string
	ID           string
}

// CreatePostgresCIDRRequest encapsulates creating an IP restriction entry.
type CreatePostgresCIDRRequest struct {
	Organization string   `json:"-"`
	Database     string   `json:"-"`
	Schema       string   `json:"schema,omitempty"`
	Role         string   `json:"role,omitempty"`
	CIDRs        []string `json:"cidrs"`
	Description  string   `json:"description,omitempty"`
}

// UpdatePostgresCIDRRequest encapsulates updating an IP restriction entry.
// Only set fields are sent. Pass an empty string for Description to clear it.
type UpdatePostgresCIDRRequest struct {
	Organization string   `json:"-"`
	Database     string   `json:"-"`
	ID           string   `json:"-"`
	Schema       *string  `json:"schema,omitempty"`
	Role         *string  `json:"role,omitempty"`
	CIDRs        []string `json:"cidrs,omitempty"`
	Description  *string  `json:"description,omitempty"`
}

// DeletePostgresCIDRRequest encapsulates deleting an IP restriction entry.
type DeletePostgresCIDRRequest struct {
	Organization string
	Database     string
	ID           string
}

// PostgresCIDRsService is an interface for the PlanetScale Postgres IP restriction API.
type PostgresCIDRsService interface {
	List(context.Context, *ListPostgresCIDRsRequest, ...ListOption) ([]*PostgresCIDR, error)
	Get(context.Context, *GetPostgresCIDRRequest) (*PostgresCIDR, error)
	Create(context.Context, *CreatePostgresCIDRRequest) (*PostgresCIDR, error)
	Update(context.Context, *UpdatePostgresCIDRRequest) (*PostgresCIDR, error)
	Delete(context.Context, *DeletePostgresCIDRRequest) error
}

type postgresCIDRsService struct {
	client *Client
}

var _ PostgresCIDRsService = &postgresCIDRsService{}

// NewPostgresCIDRsService returns a PostgresCIDRsService that uses the given client.
func NewPostgresCIDRsService(client *Client) *postgresCIDRsService {
	return &postgresCIDRsService{client: client}
}

func (s *postgresCIDRsService) List(ctx context.Context, listReq *ListPostgresCIDRsRequest, opts ...ListOption) ([]*PostgresCIDR, error) {
	listOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}

	req, err := s.client.newRequest(http.MethodGet, postgresCIDRsAPIPath(listReq.Organization, listReq.Database), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list postgres cidrs: %w", err)
	}

	resp := &postgresCIDRsResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.CIDRs, nil
}

func (s *postgresCIDRsService) Get(ctx context.Context, getReq *GetPostgresCIDRRequest) (*PostgresCIDR, error) {
	req, err := s.client.newRequest(http.MethodGet, postgresCIDRAPIPath(getReq.Organization, getReq.Database, getReq.ID), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get postgres cidr: %w", err)
	}

	entry := &PostgresCIDR{}
	if err := s.client.do(ctx, req, &entry); err != nil {
		return nil, err
	}

	return entry, nil
}

func (s *postgresCIDRsService) Create(ctx context.Context, createReq *CreatePostgresCIDRRequest) (*PostgresCIDR, error) {
	req, err := s.client.newRequest(http.MethodPost, postgresCIDRsAPIPath(createReq.Organization, createReq.Database), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for create postgres cidr: %w", err)
	}

	entry := &PostgresCIDR{}
	if err := s.client.do(ctx, req, &entry); err != nil {
		return nil, err
	}

	return entry, nil
}

func (s *postgresCIDRsService) Update(ctx context.Context, updateReq *UpdatePostgresCIDRRequest) (*PostgresCIDR, error) {
	req, err := s.client.newRequest(http.MethodPatch, postgresCIDRAPIPath(updateReq.Organization, updateReq.Database, updateReq.ID), updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for update postgres cidr: %w", err)
	}

	entry := &PostgresCIDR{}
	if err := s.client.do(ctx, req, &entry); err != nil {
		return nil, err
	}

	return entry, nil
}

func (s *postgresCIDRsService) Delete(ctx context.Context, deleteReq *DeletePostgresCIDRRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, postgresCIDRAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.ID), nil)
	if err != nil {
		return fmt.Errorf("error creating request for delete postgres cidr: %w", err)
	}

	return s.client.do(ctx, req, nil)
}

func postgresCIDRsAPIPath(org, db string) string {
	return path.Join("v1/organizations", org, "databases", db, "cidrs")
}

func postgresCIDRAPIPath(org, db, id string) string {
	return path.Join(postgresCIDRsAPIPath(org, db), id)
}
