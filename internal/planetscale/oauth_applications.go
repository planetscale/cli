package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"path"
	"time"
)

var _ OAuthApplicationsService = &oauthApplicationsService{}

// OAuthApplicationsService is an interface for communicating with the PlanetScale
// OAuth applications API.
type OAuthApplicationsService interface {
	List(context.Context, *ListOAuthApplicationsRequest, ...ListOption) ([]*OAuthApplication, error)
	Get(context.Context, *GetOAuthApplicationRequest) (*OAuthApplication, error)
	ListTokens(context.Context, *ListOAuthTokensRequest, ...ListOption) ([]*OAuthToken, error)
	GetToken(context.Context, *GetOAuthTokenRequest) (*OAuthToken, error)
	DeleteToken(context.Context, *DeleteOAuthTokenRequest) error
	CreateToken(context.Context, *CreateOAuthTokenRequest) (*OAuthToken, error)
}

type oauthApplicationsService struct {
	client *Client
}

type oauthApplicationsResponse struct {
	OAuthApplications []*OAuthApplication `json:"data"`
}

type oauthTokensResponse struct {
	OAuthTokens []*OAuthToken `json:"data"`
}

// OAuthApplication represents a PlanetScale OAuth application.
type OAuthApplication struct {
	ID                     string          `json:"id"`
	Name                   string          `json:"name"`
	RedirectURI            string          `json:"redirect_uri"`
	Domain                 string          `json:"domain"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
	Scopes                 string          `json:"scopes"`
	Avatar                 string          `json:"avatar"`
	ClientID               string          `json:"client_id"`
	Tokens                 int             `json:"tokens"`
	DCR                    bool            `json:"dcr"`
	SingleOrgAuthorization bool            `json:"single_org_authorization"`
	RequiresOrgScope       bool            `json:"requires_org_scope"`
	ScopesByResource       json.RawMessage `json:"scopes_by_resource"`
	AllScopesByResource    json.RawMessage `json:"all_scopes_by_resource"`
	MCPToolGroups          json.RawMessage `json:"mcp_tool_groups"`
}

// OAuthToken represents an OAuth application token.
type OAuthToken struct {
	ID                      string          `json:"id"`
	Name                    *string         `json:"name"`
	DisplayName             string          `json:"display_name"`
	Token                   *string         `json:"token"`
	PlainTextRefreshToken   *string         `json:"plain_text_refresh_token"`
	AvatarURL               string          `json:"avatar_url"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at"`
	ExpiresAt               *time.Time      `json:"expires_at"`
	LastUsedAt              *time.Time      `json:"last_used_at"`
	ActorID                 *string         `json:"actor_id"`
	ActorDisplayName        *string         `json:"actor_display_name"`
	ActorType               *string         `json:"actor_type"`
	ServiceTokenAccesses    json.RawMessage `json:"service_token_accesses"`
	OAuthAccessesByResource json.RawMessage `json:"oauth_accesses_by_resource"`
}

// ListOAuthApplicationsRequest is the request for listing OAuth applications.
type ListOAuthApplicationsRequest struct {
	Organization string `json:"-"`
}

// GetOAuthApplicationRequest is the request for getting an OAuth application.
type GetOAuthApplicationRequest struct {
	Organization string `json:"-"`
	ID           string `json:"-"`
}

// ListOAuthTokensRequest is the request for listing OAuth application tokens.
type ListOAuthTokensRequest struct {
	Organization  string `json:"-"`
	ApplicationID string `json:"-"`
}

// GetOAuthTokenRequest is the request for getting an OAuth application token.
type GetOAuthTokenRequest struct {
	Organization  string `json:"-"`
	ApplicationID string `json:"-"`
	TokenID       string `json:"-"`
}

// DeleteOAuthTokenRequest is the request for deleting an OAuth application token.
type DeleteOAuthTokenRequest struct {
	Organization  string `json:"-"`
	ApplicationID string `json:"-"`
	TokenID       string `json:"-"`
}

// CreateOAuthTokenRequest is the request for creating or renewing an OAuth token.
type CreateOAuthTokenRequest struct {
	Organization string `json:"-"`
	ID           string `json:"-"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
	Code         string `json:"code,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func (o *oauthApplicationsService) List(ctx context.Context, listReq *ListOAuthApplicationsRequest, opts ...ListOption) ([]*OAuthApplication, error) {
	listOpts := defaultListOptions(opts...)
	req, err := o.client.newRequest(http.MethodGet, oauthApplicationsAPIPath(listReq.Organization), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	resp := &oauthApplicationsResponse{}
	if err := o.client.do(ctx, req, resp); err != nil {
		return nil, err
	}
	return resp.OAuthApplications, nil
}

func (o *oauthApplicationsService) Get(ctx context.Context, getReq *GetOAuthApplicationRequest) (*OAuthApplication, error) {
	req, err := o.client.newRequest(http.MethodGet, oauthApplicationAPIPath(getReq.Organization, getReq.ID), nil)
	if err != nil {
		return nil, err
	}

	application := &OAuthApplication{}
	if err := o.client.do(ctx, req, application); err != nil {
		return nil, err
	}
	return application, nil
}

func (o *oauthApplicationsService) ListTokens(ctx context.Context, listReq *ListOAuthTokensRequest, opts ...ListOption) ([]*OAuthToken, error) {
	listOpts := defaultListOptions(opts...)
	req, err := o.client.newRequest(http.MethodGet, oauthTokensAPIPath(listReq.Organization, listReq.ApplicationID), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	resp := &oauthTokensResponse{}
	if err := o.client.do(ctx, req, resp); err != nil {
		return nil, err
	}
	return resp.OAuthTokens, nil
}

func (o *oauthApplicationsService) GetToken(ctx context.Context, getReq *GetOAuthTokenRequest) (*OAuthToken, error) {
	req, err := o.client.newRequest(http.MethodGet, oauthTokenAPIPath(getReq.Organization, getReq.ApplicationID, getReq.TokenID), nil)
	if err != nil {
		return nil, err
	}

	token := &OAuthToken{}
	if err := o.client.do(ctx, req, token); err != nil {
		return nil, err
	}
	return token, nil
}

func (o *oauthApplicationsService) DeleteToken(ctx context.Context, deleteReq *DeleteOAuthTokenRequest) error {
	req, err := o.client.newRequest(http.MethodDelete, oauthTokenAPIPath(deleteReq.Organization, deleteReq.ApplicationID, deleteReq.TokenID), nil)
	if err != nil {
		return err
	}
	return o.client.do(ctx, req, nil)
}

func (o *oauthApplicationsService) CreateToken(ctx context.Context, createReq *CreateOAuthTokenRequest) (*OAuthToken, error) {
	req, err := o.client.newRequest(http.MethodPost, oauthTokenCreateAPIPath(createReq.Organization, createReq.ID), createReq)
	if err != nil {
		return nil, err
	}

	token := &OAuthToken{}
	if err := o.client.do(ctx, req, token); err != nil {
		return nil, err
	}
	return token, nil
}

func oauthApplicationsAPIPath(org string) string {
	return path.Join("v1/organizations", org, "oauth-applications")
}

func oauthApplicationAPIPath(org, id string) string {
	return path.Join(oauthApplicationsAPIPath(org), id)
}

func oauthTokensAPIPath(org, applicationID string) string {
	return path.Join(oauthApplicationAPIPath(org, applicationID), "tokens")
}

func oauthTokenAPIPath(org, applicationID, tokenID string) string {
	return path.Join(oauthTokensAPIPath(org, applicationID), tokenID)
}

func oauthTokenCreateAPIPath(org, applicationID string) string {
	return path.Join(oauthApplicationAPIPath(org, applicationID), "token")
}
