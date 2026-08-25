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

type postgresSwitchoversResponse struct {
	Switchovers []*PostgresSwitchover `json:"data"`
}

type ListPostgresSwitchoversRequest struct {
	Organization string
	Database     string
	Branch       string
}

type GetPostgresSwitchoverRequest struct {
	Organization string
	Database     string
	Branch       string
	ID           string
}

// PostgresSwitchoversService is an interface for the PlanetScale Postgres
// switchover API.
type PostgresSwitchoversService interface {
	List(context.Context, *ListPostgresSwitchoversRequest, ...ListOption) ([]*PostgresSwitchover, error)
	Get(context.Context, *GetPostgresSwitchoverRequest) (*PostgresSwitchover, error)
	Create(context.Context, *CreatePostgresSwitchoverRequest) (*PostgresSwitchover, error)
}

type postgresSwitchoversService struct {
	client *Client
}

var _ PostgresSwitchoversService = &postgresSwitchoversService{}

func (s *postgresSwitchoversService) List(ctx context.Context, listReq *ListPostgresSwitchoversRequest, opts ...ListOption) ([]*PostgresSwitchover, error) {
	listOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}

	req, err := s.client.newRequest(http.MethodGet, postgresSwitchoversAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list postgres switchovers: %w", err)
	}

	resp := &postgresSwitchoversResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	return resp.Switchovers, nil
}

func (s *postgresSwitchoversService) Get(ctx context.Context, getReq *GetPostgresSwitchoverRequest) (*PostgresSwitchover, error) {
	req, err := s.client.newRequest(http.MethodGet, postgresSwitchoverAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.ID), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get postgres switchover: %w", err)
	}

	switchover := &PostgresSwitchover{}
	if err := s.client.do(ctx, req, &switchover); err != nil {
		return nil, err
	}
	return switchover, nil
}

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

func postgresSwitchoverAPIPath(org, db, branch, id string) string {
	return path.Join(postgresSwitchoversAPIPath(org, db, branch), id)
}
