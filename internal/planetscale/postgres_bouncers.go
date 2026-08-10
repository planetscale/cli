package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// PostgresBouncerSKU represents the size of a dedicated PgBouncer.
type PostgresBouncerSKU struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	CPU         string `json:"cpu"`
	RAM         int64  `json:"ram"`
	SortOrder   int    `json:"sort_order"`
}

// PostgresBouncerBranch is the branch identity embedded on a bouncer response.
type PostgresBouncerBranch struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

// PostgresBouncerParameter is a configured PgBouncer parameter on a bouncer.
type PostgresBouncerParameter struct {
	ID            string    `json:"id"`
	Namespace     string    `json:"namespace"`
	Name          string    `json:"name"`
	DisplayName   string    `json:"display_name"`
	Category      string    `json:"category"`
	Description   string    `json:"description"`
	Immutable     bool      `json:"immutable"`
	ParameterType string    `json:"parameter_type"`
	DefaultValue  string    `json:"default_value"`
	Value         string    `json:"value"`
	Required      bool      `json:"required"`
	Restart       bool      `json:"restart"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// PostgresBouncer represents a dedicated PgBouncer for a Postgres branch.
type PostgresBouncer struct {
	ID              string                      `json:"id"`
	Name            string                      `json:"name"`
	SKU             *PostgresBouncerSKU         `json:"sku"`
	Target          string                      `json:"target"`
	ReplicasPerCell int                         `json:"replicas_per_cell"`
	CreatedAt       time.Time                   `json:"created_at"`
	UpdatedAt       time.Time                   `json:"updated_at"`
	DeletedAt       *time.Time                  `json:"deleted_at"`
	Actor           Actor                       `json:"actor"`
	Branch          PostgresBouncerBranch       `json:"branch"`
	Parameters      []*PostgresBouncerParameter `json:"parameters"`
	// Deprecated response fields retained for compatibility with older API payloads.
	BouncerSize string `json:"bouncer_size,omitempty"`
}

type postgresBouncersResponse struct {
	Bouncers []*PostgresBouncer `json:"data"`
}

// ListPostgresBouncersRequest encapsulates listing PgBouncers for a branch.
type ListPostgresBouncersRequest struct {
	Organization string
	Database     string
	Branch       string
}

// GetPostgresBouncerRequest encapsulates getting a PgBouncer by name.
type GetPostgresBouncerRequest struct {
	Organization string
	Database     string
	Branch       string
	Bouncer      string
}

// CreatePostgresBouncerRequest encapsulates creating a dedicated PgBouncer.
type CreatePostgresBouncerRequest struct {
	Organization    string `json:"-"`
	Database        string `json:"-"`
	Branch          string `json:"-"`
	Name            string `json:"name,omitempty"`
	Target          string `json:"target"`
	BouncerSize     string `json:"bouncer_size,omitempty"`
	ReplicasPerCell *int   `json:"replicas_per_cell,omitempty"`
}

// DeletePostgresBouncerRequest encapsulates deleting a PgBouncer by name.
type DeletePostgresBouncerRequest struct {
	Organization string
	Database     string
	Branch       string
	Bouncer      string
}

// PostgresBouncersService is an interface for the PlanetScale PgBouncer API.
type PostgresBouncersService interface {
	List(context.Context, *ListPostgresBouncersRequest, ...ListOption) ([]*PostgresBouncer, error)
	Get(context.Context, *GetPostgresBouncerRequest) (*PostgresBouncer, error)
	Create(context.Context, *CreatePostgresBouncerRequest) (*PostgresBouncer, error)
	Delete(context.Context, *DeletePostgresBouncerRequest) error
}

type postgresBouncersService struct {
	client *Client
}

var _ PostgresBouncersService = &postgresBouncersService{}

func (s *postgresBouncersService) List(ctx context.Context, listReq *ListPostgresBouncersRequest, opts ...ListOption) ([]*PostgresBouncer, error) {
	listOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}

	req, err := s.client.newRequest(http.MethodGet, postgresBouncersAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list postgres bouncers: %w", err)
	}

	resp := &postgresBouncersResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Bouncers, nil
}

func (s *postgresBouncersService) Get(ctx context.Context, getReq *GetPostgresBouncerRequest) (*PostgresBouncer, error) {
	req, err := s.client.newRequest(http.MethodGet, postgresBouncerAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.Bouncer), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get postgres bouncer: %w", err)
	}

	bouncer := &PostgresBouncer{}
	if err := s.client.do(ctx, req, &bouncer); err != nil {
		return nil, err
	}

	return bouncer, nil
}

func (s *postgresBouncersService) Create(ctx context.Context, createReq *CreatePostgresBouncerRequest) (*PostgresBouncer, error) {
	req, err := s.client.newRequest(http.MethodPost, postgresBouncersAPIPath(createReq.Organization, createReq.Database, createReq.Branch), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for create postgres bouncer: %w", err)
	}

	bouncer := &PostgresBouncer{}
	if err := s.client.do(ctx, req, &bouncer); err != nil {
		return nil, err
	}

	return bouncer, nil
}

func (s *postgresBouncersService) Delete(ctx context.Context, deleteReq *DeletePostgresBouncerRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, postgresBouncerAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.Branch, deleteReq.Bouncer), nil)
	if err != nil {
		return fmt.Errorf("error creating request for delete postgres bouncer: %w", err)
	}

	return s.client.do(ctx, req, nil)
}

func postgresBouncersAPIPath(org, db, branch string) string {
	return path.Join("v1/organizations", org, "databases", db, "branches", branch, "bouncers")
}

func postgresBouncerAPIPath(org, db, branch, bouncer string) string {
	return path.Join(postgresBouncersAPIPath(org, db, branch), bouncer)
}
