package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

const (
	PostgresBouncerResizeStatePending   = "pending"
	PostgresBouncerResizeStateResizing  = "resizing"
	PostgresBouncerResizeStateCanceled  = "canceled"
	PostgresBouncerResizeStateCompleted = "completed"
)

// PostgresBouncerResizeRequest is an asynchronous dedicated-PgBouncer change.
type PostgresBouncerResizeRequest struct {
	ID                      string                `json:"id"`
	State                   string                `json:"state"`
	ReplicasPerCell         int                   `json:"replicas_per_cell"`
	Target                  string                `json:"target"`
	Parameters              map[string]any        `json:"parameters"`
	PreviousReplicasPerCell int                   `json:"previous_replicas_per_cell"`
	PreviousTarget          string                `json:"previous_target"`
	PreviousParameters      map[string]any        `json:"previous_parameters"`
	StartedAt               *time.Time            `json:"started_at"`
	CompletedAt             *time.Time            `json:"completed_at"`
	CreatedAt               time.Time             `json:"created_at"`
	UpdatedAt               time.Time             `json:"updated_at"`
	Actor                   Actor                 `json:"actor"`
	Bouncer                 PostgresBouncerBranch `json:"bouncer"`
	SKU                     *PostgresBouncerSKU   `json:"sku"`
	PreviousSKU             *PostgresBouncerSKU   `json:"previous_sku"`
}

// Finished reports whether the resize request is in a terminal state.
func (r *PostgresBouncerResizeRequest) Finished() bool {
	return r.State == PostgresBouncerResizeStateCompleted || r.State == PostgresBouncerResizeStateCanceled
}

type postgresBouncerResizesResponse struct {
	Resizes []*PostgresBouncerResizeRequest `json:"data"`
}

// ListPostgresBouncerResizesRequest lists resize requests for one bouncer.
type ListPostgresBouncerResizesRequest struct {
	Organization string
	Database     string
	Branch       string
	Bouncer      string
}

// ResizePostgresBouncerRequest upserts a resize request for a bouncer.
type ResizePostgresBouncerRequest struct {
	Organization    string                       `json:"-"`
	Database        string                       `json:"-"`
	Branch          string                       `json:"-"`
	Bouncer         string                       `json:"-"`
	BouncerSize     string                       `json:"bouncer_size,omitempty"`
	ReplicasPerCell *int                         `json:"replicas_per_cell,omitempty"`
	Target          string                       `json:"target,omitempty"`
	Parameters      map[string]map[string]string `json:"parameters,omitempty"`
}

// CancelPostgresBouncerResizesRequest cancels unfinished resize requests.
type CancelPostgresBouncerResizesRequest struct {
	Organization string
	Database     string
	Branch       string
	Bouncer      string
}

// ListResizes lists resize requests for a dedicated PgBouncer (newest first).
func (s *postgresBouncersService) ListResizes(ctx context.Context, listReq *ListPostgresBouncerResizesRequest, opts ...ListOption) ([]*PostgresBouncerResizeRequest, error) {
	listOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}

	req, err := s.client.newRequest(http.MethodGet, postgresBouncerResizesAPIPath(listReq.Organization, listReq.Database, listReq.Branch, listReq.Bouncer), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list postgres bouncer resizes: %w", err)
	}

	resp := &postgresBouncerResizesResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Resizes, nil
}

// Resize upserts a resize request for a dedicated PgBouncer.
func (s *postgresBouncersService) Resize(ctx context.Context, resizeReq *ResizePostgresBouncerRequest) (*PostgresBouncerResizeRequest, error) {
	req, err := s.client.newRequest(http.MethodPatch, postgresBouncerResizesAPIPath(resizeReq.Organization, resizeReq.Database, resizeReq.Branch, resizeReq.Bouncer), resizeReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for resize postgres bouncer: %w", err)
	}

	out := &PostgresBouncerResizeRequest{}
	if err := s.client.do(ctx, req, &out); err != nil {
		return nil, err
	}

	// An empty body (for example a 204 when the bouncer already matches the
	// requested configuration) leaves the ID empty. Surface that as a nil
	// resize request so callers can detect the no-op.
	if out.ID == "" {
		return nil, nil
	}

	return out, nil
}

// CancelResizes cancels unfinished resize requests for a dedicated PgBouncer.
func (s *postgresBouncersService) CancelResizes(ctx context.Context, cancelReq *CancelPostgresBouncerResizesRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, postgresBouncerResizesAPIPath(cancelReq.Organization, cancelReq.Database, cancelReq.Branch, cancelReq.Bouncer), nil)
	if err != nil {
		return fmt.Errorf("error creating request for cancel postgres bouncer resizes: %w", err)
	}

	return s.client.do(ctx, req, nil)
}

func postgresBouncerResizesAPIPath(org, db, branch, bouncer string) string {
	return path.Join(postgresBouncerAPIPath(org, db, branch, bouncer), "resizes")
}
