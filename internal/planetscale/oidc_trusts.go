package planetscale

import (
	"context"
	"net/http"
	"path"
	"time"
)

var _ OIDCTrustsService = &oidcTrustsService{}

// OIDCTrustsService is an interface for communicating with the PlanetScale
// OIDC trusts API.
type OIDCTrustsService interface {
	ListProviders(context.Context, *ListOIDCTrustProvidersRequest) ([]*OIDCTrustProvider, error)
	Get(context.Context, *GetOIDCTrustRequest) (*OIDCTrust, error)
}

// OIDCTrust represents a trust between an organization and an identity provider.
type OIDCTrust struct {
	ID                   string               `json:"id"`
	DisplayName          string               `json:"display_name"`
	Provider             string               `json:"provider"`
	Audience             string               `json:"audience"`
	MaxTTLSeconds        int                  `json:"max_ttl_seconds"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
	Conditions           []OIDCTrustCondition `json:"conditions"`
	Accesses             []OIDCTrustAccess    `json:"accesses"`
	BulkDatabaseAccesses []string             `json:"bulk_database_accesses"`
}

// OIDCTrustCondition is a claim that a token must match.
type OIDCTrustCondition struct {
	ID    string `json:"id"`
	Claim string `json:"claim"`
	Value string `json:"value"`
}

// OIDCTrustAccess is an access granted to tokens that match a trust.
type OIDCTrustAccess struct {
	ID           string `json:"id"`
	Access       string `json:"access"`
	ResourceName string `json:"resource_name"`
	ResourceID   string `json:"resource_id"`
	ResourceType string `json:"resource_type"`
}

// GetOIDCTrustRequest is the request for getting an OIDC trust.
type GetOIDCTrustRequest struct {
	Organization string `json:"-"`
	ID           string `json:"-"`
}

// OIDCTrustProvider represents an identity provider that can be trusted.
type OIDCTrustProvider struct {
	Name           string   `json:"name"`
	DisplayName    string   `json:"display_name"`
	Issuer         string   `json:"issuer"`
	RequiredClaims []string `json:"required_claims"`
}

// ListOIDCTrustProvidersRequest is the request for listing OIDC trust providers.
type ListOIDCTrustProvidersRequest struct {
	Organization string `json:"-"`
}

type oidcTrustProvidersResponse struct {
	Providers []*OIDCTrustProvider `json:"data"`
}

type oidcTrustsService struct {
	client *Client
}

func (s *oidcTrustsService) ListProviders(ctx context.Context, listReq *ListOIDCTrustProvidersRequest) ([]*OIDCTrustProvider, error) {
	req, err := s.client.newRequest(http.MethodGet, oidcTrustProvidersAPIPath(listReq.Organization), nil)
	if err != nil {
		return nil, err
	}

	resp := &oidcTrustProvidersResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Providers, nil
}

func oidcTrustProvidersAPIPath(org string) string {
	return path.Join("v1/organizations", org, "oidc-trust-providers")
}

func (s *oidcTrustsService) Get(ctx context.Context, getReq *GetOIDCTrustRequest) (*OIDCTrust, error) {
	req, err := s.client.newRequest(http.MethodGet, oidcTrustAPIPath(getReq.Organization, getReq.ID), nil)
	if err != nil {
		return nil, err
	}

	trust := &OIDCTrust{}
	if err := s.client.do(ctx, req, &trust); err != nil {
		return nil, err
	}

	return trust, nil
}

func oidcTrustsAPIPath(org string) string {
	return path.Join("v1/organizations", org, "oidc-trusts")
}

func oidcTrustAPIPath(org, id string) string {
	return path.Join(oidcTrustsAPIPath(org), id)
}
