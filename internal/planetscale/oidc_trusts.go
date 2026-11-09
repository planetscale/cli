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
	List(context.Context, *ListOIDCTrustsRequest, ...ListOption) ([]*OIDCTrust, error)
	Get(context.Context, *GetOIDCTrustRequest) (*OIDCTrust, error)
	ListProviders(context.Context, *ListOIDCTrustProvidersRequest) ([]*OIDCTrustProvider, error)
}

// OIDCTrustCondition is a claim a token must match to use a trust.
type OIDCTrustCondition struct {
	ID    string `json:"id"`
	Claim string `json:"claim"`
	Value string `json:"value"`
}

// OIDCTrustAccess is an access granted by a trust.
type OIDCTrustAccess struct {
	ID           string `json:"id"`
	Access       string `json:"access"`
	ResourceName string `json:"resource_name"`
	ResourceID   string `json:"resource_id"`
	ResourceType string `json:"resource_type"`
}

// OIDCTrust represents a PlanetScale OIDC trust.
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

// OIDCTrustProvider is an OIDC provider that trusts can be created for.
type OIDCTrustProvider struct {
	Name           string   `json:"name"`
	DisplayName    string   `json:"display_name"`
	Issuer         string   `json:"issuer"`
	RequiredClaims []string `json:"required_claims"`
}

// ListOIDCTrustsRequest is the request for listing OIDC trusts.
type ListOIDCTrustsRequest struct {
	Organization string `json:"-"`
}

// GetOIDCTrustRequest is the request for getting an OIDC trust.
type GetOIDCTrustRequest struct {
	Organization string `json:"-"`
	ID           string `json:"-"`
}

// ListOIDCTrustProvidersRequest is the request for listing OIDC providers.
type ListOIDCTrustProvidersRequest struct {
	Organization string `json:"-"`
}

type oidcTrustsResponse struct {
	Trusts []*OIDCTrust `json:"data"`
}

type oidcTrustProvidersResponse struct {
	Providers []*OIDCTrustProvider `json:"data"`
}

type oidcTrustsService struct {
	client *Client
}

func (o *oidcTrustsService) List(ctx context.Context, listReq *ListOIDCTrustsRequest, opts ...ListOption) ([]*OIDCTrust, error) {
	listOpts := defaultListOptions(opts...)

	req, err := o.client.newRequest(http.MethodGet, oidcTrustsAPIPath(listReq.Organization), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	resp := &oidcTrustsResponse{}
	if err := o.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Trusts, nil
}

func (o *oidcTrustsService) Get(ctx context.Context, getReq *GetOIDCTrustRequest) (*OIDCTrust, error) {
	req, err := o.client.newRequest(http.MethodGet, oidcTrustAPIPath(getReq.Organization, getReq.ID), nil)
	if err != nil {
		return nil, err
	}

	trust := &OIDCTrust{}
	if err := o.client.do(ctx, req, &trust); err != nil {
		return nil, err
	}

	return trust, nil
}

func (o *oidcTrustsService) ListProviders(ctx context.Context, listReq *ListOIDCTrustProvidersRequest) ([]*OIDCTrustProvider, error) {
	req, err := o.client.newRequest(http.MethodGet, path.Join(oidcTrustsAPIPath(listReq.Organization), "providers"), nil)
	if err != nil {
		return nil, err
	}

	resp := &oidcTrustProvidersResponse{}
	if err := o.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Providers, nil
}

func oidcTrustsAPIPath(org string) string {
	return path.Join("v1/organizations", org, "oidc-trusts")
}

func oidcTrustAPIPath(org, id string) string {
	return path.Join(oidcTrustsAPIPath(org), id)
}
