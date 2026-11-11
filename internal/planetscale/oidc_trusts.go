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
	List(context.Context, *ListOIDCTrustsRequest, ...ListOption) ([]*OIDCTrust, error)
	Get(context.Context, *GetOIDCTrustRequest) (*OIDCTrust, error)
	Create(context.Context, *CreateOIDCTrustRequest) (*OIDCTrust, error)
	Update(context.Context, *UpdateOIDCTrustRequest) (*OIDCTrust, error)
	Delete(context.Context, *DeleteOIDCTrustRequest) error
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

// ListOIDCTrustsRequest is the request for listing OIDC trusts.
type ListOIDCTrustsRequest struct {
	Organization string `json:"-"`
}

type oidcTrustsResponse struct {
	Trusts []*OIDCTrust `json:"data"`
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

func (s *oidcTrustsService) List(ctx context.Context, listReq *ListOIDCTrustsRequest, opts ...ListOption) ([]*OIDCTrust, error) {
	listOpts := defaultListOptions(opts...)

	req, err := s.client.newRequest(http.MethodGet, oidcTrustsAPIPath(listReq.Organization), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	resp := &oidcTrustsResponse{}
	if err := s.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Trusts, nil
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

// OIDCTrustConditionInput is a claim condition to set on a trust.
type OIDCTrustConditionInput struct {
	Claim string `json:"claim"`
	Value string `json:"value"`
}

// OIDCTrustAccessInput is an access to grant. An empty Database grants the
// access on the organization.
type OIDCTrustAccessInput struct {
	Access   string `json:"access"`
	Database string `json:"database,omitempty"`
}

// CreateOIDCTrustRequest is the request for creating an OIDC trust.
type CreateOIDCTrustRequest struct {
	Organization         string                    `json:"-"`
	DisplayName          string                    `json:"display_name"`
	Provider             string                    `json:"provider"`
	MaxTTLSeconds        int                       `json:"max_ttl_seconds"`
	Conditions           []OIDCTrustConditionInput `json:"conditions"`
	Accesses             []OIDCTrustAccessInput    `json:"accesses"`
	BulkDatabaseAccesses []string                  `json:"bulk_database_accesses,omitempty"`
}

func (s *oidcTrustsService) Create(ctx context.Context, createReq *CreateOIDCTrustRequest) (*OIDCTrust, error) {
	body := *createReq
	if body.Conditions == nil {
		body.Conditions = []OIDCTrustConditionInput{}
	}
	if body.Accesses == nil {
		body.Accesses = []OIDCTrustAccessInput{}
	}

	req, err := s.client.newRequest(http.MethodPost, oidcTrustsAPIPath(createReq.Organization), &body)
	if err != nil {
		return nil, err
	}

	trust := &OIDCTrust{}
	if err := s.client.do(ctx, req, &trust); err != nil {
		return nil, err
	}

	return trust, nil
}

// UpdateOIDCTrustRequest is the request for updating an OIDC trust. Pointer
// fields that are nil are not sent; a sent list replaces the existing list.
type UpdateOIDCTrustRequest struct {
	Organization         string                     `json:"-"`
	ID                   string                     `json:"-"`
	DisplayName          *string                    `json:"display_name,omitempty"`
	MaxTTLSeconds        *int                       `json:"max_ttl_seconds,omitempty"`
	Conditions           *[]OIDCTrustConditionInput `json:"conditions,omitempty"`
	Accesses             *[]OIDCTrustAccessInput    `json:"accesses,omitempty"`
	BulkDatabaseAccesses *[]string                  `json:"bulk_database_accesses,omitempty"`
}

// DeleteOIDCTrustRequest is the request for deleting an OIDC trust.
type DeleteOIDCTrustRequest struct {
	Organization string `json:"-"`
	ID           string `json:"-"`
}

func (s *oidcTrustsService) Update(ctx context.Context, updateReq *UpdateOIDCTrustRequest) (*OIDCTrust, error) {
	req, err := s.client.newRequest(http.MethodPatch, oidcTrustAPIPath(updateReq.Organization, updateReq.ID), updateReq)
	if err != nil {
		return nil, err
	}

	trust := &OIDCTrust{}
	if err := s.client.do(ctx, req, &trust); err != nil {
		return nil, err
	}

	return trust, nil
}

func (s *oidcTrustsService) Delete(ctx context.Context, deleteReq *DeleteOIDCTrustRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, oidcTrustAPIPath(deleteReq.Organization, deleteReq.ID), nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
