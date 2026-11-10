package planetscale

import (
	"context"
	"net/http"
	"path"
)

var _ OIDCTrustsService = &oidcTrustsService{}

// OIDCTrustsService is an interface for communicating with the PlanetScale
// OIDC trusts API.
type OIDCTrustsService interface {
	ListProviders(context.Context, *ListOIDCTrustProvidersRequest) ([]*OIDCTrustProvider, error)
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
