package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

const (
	PostgresBouncerRestartStatePending   = "pending"
	PostgresBouncerRestartStateRunning   = "running"
	PostgresBouncerRestartStateSucceeded = "succeeded"
	PostgresBouncerRestartStateFailed    = "failed"
	PostgresBouncerRestartStateCanceled  = "canceled"
)

// PostgresBouncerLocalName addresses the built-in PgBouncer of the primary
// instance in the restart API. Dedicated PgBouncers cannot use this name.
const PostgresBouncerLocalName = "primary"

// PostgresBouncerRestart is an asynchronous in-place restart of a PgBouncer.
type PostgresBouncerRestart struct {
	ID          string     `json:"id"`
	State       string     `json:"state"`
	Target      string     `json:"target"`
	Bouncer     string     `json:"bouncer"`
	Error       string     `json:"error,omitempty"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Actor       Actor      `json:"actor"`
}

// Finished reports whether the restart is in a terminal state.
func (r *PostgresBouncerRestart) Finished() bool {
	switch r.State {
	case PostgresBouncerRestartStateSucceeded, PostgresBouncerRestartStateFailed, PostgresBouncerRestartStateCanceled:
		return true
	default:
		return false
	}
}

type postgresBouncerRestartsResponse struct {
	Restarts []*PostgresBouncerRestart `json:"data"`
}

// CreatePostgresBouncerRestartRequest restarts a dedicated PgBouncer, or the
// primary's built-in PgBouncer when Bouncer is PostgresBouncerLocalName.
type CreatePostgresBouncerRestartRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Bouncer      string `json:"-"`
}

// ListPostgresBouncerRestartsRequest lists restarts for one bouncer.
type ListPostgresBouncerRestartsRequest struct {
	Organization string
	Database     string
	Branch       string
	Bouncer      string
}

// GetPostgresBouncerRestartRequest fetches one restart by ID.
type GetPostgresBouncerRestartRequest struct {
	Organization string
	Database     string
	Branch       string
	Bouncer      string
	ID           string
}

// CreateRestart requests an in-place restart of a PgBouncer.
func (s *postgresBouncersService) CreateRestart(ctx context.Context, createReq *CreatePostgresBouncerRestartRequest) (*PostgresBouncerRestart, error) {
	req, err := s.client.newRequest(http.MethodPost, postgresBouncerRestartsAPIPath(createReq.Organization, createReq.Database, createReq.Branch, createReq.Bouncer), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for create postgres bouncer restart: %w", err)
	}

	out := &PostgresBouncerRestart{}
	if err := s.client.do(ctx, req, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// ListRestarts lists restarts for a PgBouncer (newest first).
func (s *postgresBouncersService) ListRestarts(ctx context.Context, listReq *ListPostgresBouncerRestartsRequest, opts ...ListOption) ([]*PostgresBouncerRestart, error) {
	listOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}

	req, err := s.client.newRequest(http.MethodGet, postgresBouncerRestartsAPIPath(listReq.Organization, listReq.Database, listReq.Branch, listReq.Bouncer), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list postgres bouncer restarts: %w", err)
	}

	resp := &postgresBouncerRestartsResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Restarts, nil
}

// GetRestart fetches one restart of a PgBouncer.
func (s *postgresBouncersService) GetRestart(ctx context.Context, getReq *GetPostgresBouncerRestartRequest) (*PostgresBouncerRestart, error) {
	req, err := s.client.newRequest(http.MethodGet, path.Join(postgresBouncerRestartsAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.Bouncer), getReq.ID), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get postgres bouncer restart: %w", err)
	}

	out := &PostgresBouncerRestart{}
	if err := s.client.do(ctx, req, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func postgresBouncerRestartsAPIPath(org, db, branch, bouncer string) string {
	return path.Join(postgresBouncerAPIPath(org, db, branch, bouncer), "restarts")
}
