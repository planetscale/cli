package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// PostgresSwitchover represents a switchover of the primary of a Postgres
// branch.
type PostgresSwitchover struct {
	ID          string     `json:"id"`
	State       string     `json:"state"`
	Method      string     `json:"method,omitempty"`
	Error       string     `json:"error,omitempty"`
	Actor       Actor      `json:"actor"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreatePostgresSwitchoverRequest encapsulates creating a switchover for a
// Postgres branch.
type CreatePostgresSwitchoverRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Candidate    string `json:"candidate,omitempty"`
}

// PostgresSwitchoversService is an interface for the PlanetScale Postgres
// switchover API.
type PostgresSwitchoversService interface {
	Create(context.Context, *CreatePostgresSwitchoverRequest) (*PostgresSwitchover, error)
}

type postgresSwitchoversService struct {
	client *Client
}

var _ PostgresSwitchoversService = &postgresSwitchoversService{}

func (s *postgresSwitchoversService) Create(ctx context.Context, createReq *CreatePostgresSwitchoverRequest) (*PostgresSwitchover, error) {
	req, err := s.client.newRequest(http.MethodPost, postgresSwitchoversAPIPath(createReq.Organization, createReq.Database, createReq.Branch), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for create postgres switchover: %w", err)
	}

	switchover := &PostgresSwitchover{}
	if err := s.client.do(ctx, req, &switchover); err != nil {
		return nil, err
	}

	return switchover, nil
}

func postgresSwitchoversAPIPath(org, db, branch string) string {
	return path.Join("v1/organizations", org, "databases", db, "branches", branch, "switchovers")
}
