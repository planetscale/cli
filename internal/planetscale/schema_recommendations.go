package planetscale

import (
	"context"
	"net/http"
	"path"
	"time"
)

var _ SchemaRecommendationService = &schemaRecommendationService{}

// SchemaRecommendationService is an interface for communicating with the PlanetScale
// Schema recommendation API.
type SchemaRecommendationService interface {
	List(context.Context, *ListSchemaRecommendationsRequest, ...ListOption) ([]*SchemaRecommendation, error)
	Get(context.Context, *GetSchemaRecommendationRequest) (*SchemaRecommendation, error)
	Dismiss(context.Context, *DismissSchemaRecommendationRequest) (*SchemaRecommendation, error)
}

type schemaRecommendationsResponse struct {
	SchemaRecommendations []*SchemaRecommendation `json:"data"`
}

// SchemaRecommendation represents a PlanetScale schema recommendation.
type SchemaRecommendation struct {
	ID                    string                 `json:"id"`
	HtmlURL               string                 `json:"html_url"`
	Title                 string                 `json:"title"`
	Table                 string                 `json:"table_name"`
	Keyspace              string                 `json:"keyspace"`
	DDLStatement          string                 `json:"ddl_statement"`
	Number                int                    `json:"number"`
	State                 string                 `json:"state"`
	RecommendationType    string                 `json:"recommendation_type"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
	AppliedAt             *time.Time             `json:"applied_at"`
	DismissedAt           *time.Time             `json:"dismissed_at"`
	ClosedByDeployRequest *ClosedByDeployRequest `json:"closed_by_deploy_request"`
	DismissedBy           *Actor                 `json:"dismissed_by"`
}

type ClosedByDeployRequest struct {
	ID       string `json:"id"`
	BranchID string `json:"branch_id"`
	Number   int    `json:"number"`
}

// ListSchemaRecommendationsRequest is the request for listing schema recommendations.
type ListSchemaRecommendationsRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
}

// GetSchemaRecommendationRequest is the request for getting a schema recommendation.
type GetSchemaRecommendationRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	ID           string `json:"-"`
}

// DismissSchemaRecommendationRequest is the request for dismissing a schema recommendation.
// ID is the recommendation sequence number from list/show (path param `number`).
type DismissSchemaRecommendationRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	ID           string `json:"-"`
	Reason       string `json:"reason,omitempty"`
}

type schemaRecommendationService struct {
	client *Client
}

func (s *schemaRecommendationService) List(ctx context.Context, request *ListSchemaRecommendationsRequest, opts ...ListOption) ([]*SchemaRecommendation, error) {
	listOpts := defaultListOptions(opts...)

	req, err := s.client.newRequest(http.MethodGet, schemaRecommendationsAPIPath(request.Organization, request.Database), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	resp := &schemaRecommendationsResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.SchemaRecommendations, nil
}

func (s *schemaRecommendationService) Get(ctx context.Context, request *GetSchemaRecommendationRequest) (*SchemaRecommendation, error) {
	req, err := s.client.newRequest(http.MethodGet, schemaRecommendationAPIPath(request.Organization, request.Database, request.ID), nil)
	if err != nil {
		return nil, err
	}

	schemaRecommendation := &SchemaRecommendation{}
	if err := s.client.do(ctx, req, &schemaRecommendation); err != nil {
		return nil, err
	}

	return schemaRecommendation, nil
}

func (s *schemaRecommendationService) Dismiss(ctx context.Context, request *DismissSchemaRecommendationRequest) (*SchemaRecommendation, error) {
	req, err := s.client.newRequest(http.MethodPost, dismissSchemaRecommendationAPIPath(request.Organization, request.Database, request.ID), request)
	if err != nil {
		return nil, err
	}

	schemaRecommendation := &SchemaRecommendation{}
	if err := s.client.do(ctx, req, &schemaRecommendation); err != nil {
		return nil, err
	}

	return schemaRecommendation, nil
}

func schemaRecommendationsAPIPath(org, db string) string {
	return path.Join("v1/organizations", org, "databases", db, "schema-recommendations")
}

func schemaRecommendationAPIPath(org, db, id string) string {
	return path.Join(schemaRecommendationsAPIPath(org, db), id)
}

func dismissSchemaRecommendationAPIPath(org, db, id string) string {
	return path.Join(schemaRecommendationAPIPath(org, db, id), "dismiss")
}
