package planetscale

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"time"
)

var _ TrafficBudgetsService = &trafficBudgetsService{}

// TrafficBudgetsService communicates with the PlanetScale traffic budgets API.
type TrafficBudgetsService interface {
	List(context.Context, *ListTrafficBudgetsRequest) ([]*TrafficBudget, error)
	Get(context.Context, *GetTrafficBudgetRequest) (*TrafficBudget, error)
	Create(context.Context, *CreateTrafficBudgetRequest) (*TrafficBudget, error)
	Update(context.Context, *UpdateTrafficBudgetRequest) (*TrafficBudget, error)
	Delete(context.Context, *DeleteTrafficBudgetRequest) error
}

// TrafficRuleTag is a tag attached to a traffic rule.
type TrafficRuleTag struct {
	Type   string `json:"type"`
	KeyID  string `json:"key_id"`
	Key    string `json:"key"`
	Value  string `json:"value"`
	Source string `json:"source"`
}

// TrafficRule represents a rule for a traffic budget.
type TrafficRule struct {
	ID                   string           `json:"id"`
	Type                 string           `json:"type"`
	Kind                 string           `json:"kind"`
	Fingerprint          *string          `json:"fingerprint"`
	Keyspace             *string          `json:"keyspace"`
	Tags                 []TrafficRuleTag `json:"tags"`
	Actor                Actor            `json:"actor"`
	SyntaxHighlightedSQL string           `json:"syntax_highlighted_sql"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
}

// TrafficBudget represents a traffic budget on a branch.
type TrafficBudget struct {
	ID   string `json:"id"`
	Type string `json:"type"`

	Name string `json:"name"`
	Mode string `json:"mode"`

	Capacity         *int `json:"capacity"`
	Rate             *int `json:"rate"`
	Burst            *int `json:"burst"`
	Concurrency      *int `json:"concurrency"`
	WarningThreshold *int `json:"warning_threshold"`

	Rules []TrafficRule `json:"rules"`

	Actor     Actor     `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type trafficBudgetsResponse struct {
	Data []*TrafficBudget `json:"data"`
}

// ListTrafficBudgetsRequest is the request for listing traffic budgets.
type ListTrafficBudgetsRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	// Fingerprint, when set, returns only budgets with a rule for that query
	// fingerprint.
	Fingerprint string `json:"-"`
}

// GetTrafficBudgetRequest is the request for getting a traffic budget.
type GetTrafficBudgetRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	BudgetID     string `json:"-"`
}

// CreateTrafficBudgetRuleRequest describes a rule when creating or updating a budget.
type CreateTrafficBudgetRuleRequest struct {
	Kind        string            `json:"kind"`
	Fingerprint string            `json:"fingerprint,omitempty"`
	Keyspace    string            `json:"keyspace,omitempty"`
	Tags        *[]TrafficRuleTag `json:"tags,omitempty"`
}

// CreateTrafficBudgetRequest is the request for creating a traffic budget.
type CreateTrafficBudgetRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`

	Name string `json:"name"`
	Mode string `json:"mode"`

	Capacity         *int `json:"capacity,omitempty"`
	Rate             *int `json:"rate,omitempty"`
	Burst            *int `json:"burst,omitempty"`
	Concurrency      *int `json:"concurrency,omitempty"`
	WarningThreshold *int `json:"warning_threshold,omitempty"`

	Rules *[]CreateTrafficBudgetRuleRequest `json:"rules,omitempty"`
}

// UpdateTrafficBudgetRequest is the request for updating a traffic budget.
type UpdateTrafficBudgetRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	BudgetID     string `json:"-"`

	Name *string `json:"name,omitempty"`
	Mode *string `json:"mode,omitempty"`

	Capacity         *int `json:"capacity,omitempty"`
	Rate             *int `json:"rate,omitempty"`
	Burst            *int `json:"burst,omitempty"`
	Concurrency      *int `json:"concurrency,omitempty"`
	WarningThreshold *int `json:"warning_threshold,omitempty"`

	Rules *[]CreateTrafficBudgetRuleRequest `json:"rules,omitempty"`
}

// DeleteTrafficBudgetRequest is the request for deleting a traffic budget.
type DeleteTrafficBudgetRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	BudgetID     string `json:"-"`
}

type trafficBudgetsService struct {
	client *Client
}

// NewTrafficBudgetsService returns a TrafficBudgetsService backed by client.
func NewTrafficBudgetsService(client *Client) *trafficBudgetsService {
	return &trafficBudgetsService{client: client}
}

func (s *trafficBudgetsService) List(ctx context.Context, listReq *ListTrafficBudgetsRequest) ([]*TrafficBudget, error) {
	var opts []RequestOption
	if listReq.Fingerprint != "" {
		v := url.Values{}
		v.Add("fingerprint", listReq.Fingerprint)
		opts = append(opts, WithQueryParams(v))
	}

	req, err := s.client.newRequest(http.MethodGet, trafficBudgetsAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil, opts...)
	if err != nil {
		return nil, err
	}

	resp := &trafficBudgetsResponse{}
	if err := s.client.do(ctx, req, resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (s *trafficBudgetsService) Get(ctx context.Context, getReq *GetTrafficBudgetRequest) (*TrafficBudget, error) {
	req, err := s.client.newRequest(http.MethodGet, trafficBudgetAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.BudgetID), nil)
	if err != nil {
		return nil, err
	}

	budget := &TrafficBudget{}
	if err := s.client.do(ctx, req, budget); err != nil {
		return nil, err
	}

	return budget, nil
}

func (s *trafficBudgetsService) Create(ctx context.Context, createReq *CreateTrafficBudgetRequest) (*TrafficBudget, error) {
	req, err := s.client.newRequest(http.MethodPost, trafficBudgetsAPIPath(createReq.Organization, createReq.Database, createReq.Branch), createReq)
	if err != nil {
		return nil, err
	}

	budget := &TrafficBudget{}
	if err := s.client.do(ctx, req, budget); err != nil {
		return nil, err
	}

	return budget, nil
}

func (s *trafficBudgetsService) Update(ctx context.Context, updateReq *UpdateTrafficBudgetRequest) (*TrafficBudget, error) {
	req, err := s.client.newRequest(http.MethodPatch, trafficBudgetAPIPath(updateReq.Organization, updateReq.Database, updateReq.Branch, updateReq.BudgetID), updateReq)
	if err != nil {
		return nil, err
	}

	budget := &TrafficBudget{}
	if err := s.client.do(ctx, req, budget); err != nil {
		return nil, err
	}

	return budget, nil
}

func (s *trafficBudgetsService) Delete(ctx context.Context, deleteReq *DeleteTrafficBudgetRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, trafficBudgetAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.Branch, deleteReq.BudgetID), nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

func trafficBudgetsAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "traffic", "budgets")
}

func trafficBudgetAPIPath(org, db, branch, budgetID string) string {
	return path.Join(trafficBudgetsAPIPath(org, db, branch), budgetID)
}
