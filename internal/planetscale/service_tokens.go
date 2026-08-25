package planetscale

import (
	"context"
	"net/http"
	"path"
	"time"
)

var _ ServiceTokenService = &serviceTokenService{}

// ServiceTokenService is an interface for communicating with the PlanetScale
// Service Token API.
type ServiceTokenService interface {
	Create(context.Context, *CreateServiceTokenRequest) (*ServiceToken, error)
	Get(context.Context, *GetServiceTokenRequest) (*ServiceToken, error)
	List(context.Context, *ListServiceTokensRequest) ([]*ServiceToken, error)
	ListGrants(context.Context, *ListServiceTokenGrantsRequest) ([]*ServiceTokenGrant, error)
	Delete(context.Context, *DeleteServiceTokenRequest) error
	GetAccess(context.Context, *GetServiceTokenAccessRequest) ([]*ServiceTokenAccess, error)
	AddAccess(context.Context, *AddServiceTokenAccessRequest) ([]*ServiceTokenAccess, error)
	DeleteAccess(context.Context, *DeleteServiceTokenAccessRequest) error
}

type serviceTokenService struct {
	client *Client
}

func (s *serviceTokenService) Create(ctx context.Context, createReq *CreateServiceTokenRequest) (*ServiceToken, error) {
	req, err := s.client.newRequest(http.MethodPost, serviceTokensAPIPath(createReq.Organization), createReq)
	if err != nil {
		return nil, err
	}

	st := &ServiceToken{}
	if err := s.client.do(ctx, req, &st); err != nil {
		return nil, err
	}

	return st, nil
}

func (s *serviceTokenService) Get(ctx context.Context, getReq *GetServiceTokenRequest) (*ServiceToken, error) {
	req, err := s.client.newRequest(http.MethodGet, serviceTokenAPIPath(getReq.Organization, getReq.ID), nil)
	if err != nil {
		return nil, err
	}

	st := &ServiceToken{}
	if err := s.client.do(ctx, req, st); err != nil {
		return nil, err
	}

	return st, nil
}

func (s *serviceTokenService) List(ctx context.Context, listReq *ListServiceTokensRequest) ([]*ServiceToken, error) {
	req, err := s.client.newRequest(http.MethodGet, serviceTokensAPIPath(listReq.Organization), nil)
	if err != nil {
		return nil, err
	}

	tokenListResponse := serviceTokensResponse{}
	if err := s.client.do(ctx, req, &tokenListResponse); err != nil {
		return nil, err
	}

	return tokenListResponse.ServiceTokens, nil
}

func (s *serviceTokenService) Delete(ctx context.Context, delReq *DeleteServiceTokenRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, serviceTokenAPIPath(delReq.Organization, delReq.ID), nil)
	if err != nil {
		return err
	}

	err = s.client.do(ctx, req, nil)
	return err
}

func (s *serviceTokenService) GetAccess(ctx context.Context, accessReq *GetServiceTokenAccessRequest) ([]*ServiceTokenAccess, error) {
	req, err := s.client.newRequest(http.MethodGet, serviceTokenAccessAPIPath(accessReq.Organization, accessReq.ID), nil)
	if err != nil {
		return nil, err
	}

	tokenAccess := serviceTokenAccessResponse{}
	if err := s.client.do(ctx, req, &tokenAccess); err != nil {
		return nil, err
	}
	return tokenAccess.ServiceTokenAccesses, nil
}

func (s *serviceTokenService) ListGrants(ctx context.Context, listReq *ListServiceTokenGrantsRequest) ([]*ServiceTokenGrant, error) {
	req, err := s.client.newRequest(http.MethodGet, serviceTokenGrantsAPIPath(listReq.Organization, listReq.ID), nil)
	if err != nil {
		return nil, err
	}

	tokenGrants := serviceTokenGrantsResponse{}
	if err := s.client.do(ctx, req, &tokenGrants); err != nil {
		return nil, err
	}
	return tokenGrants.ServiceTokenGrants, nil
}

func (s *serviceTokenService) AddAccess(ctx context.Context, addReq *AddServiceTokenAccessRequest) ([]*ServiceTokenAccess, error) {
	req, err := s.client.newRequest(http.MethodPost, serviceTokenAccessAPIPath(addReq.Organization, addReq.ID), addReq)
	if err != nil {
		return nil, err
	}

	tokenAccess := serviceTokenAccessResponse{}
	if err := s.client.do(ctx, req, &tokenAccess); err != nil {
		return nil, err
	}
	return tokenAccess.ServiceTokenAccesses, nil
}

func (s *serviceTokenService) DeleteAccess(ctx context.Context, delReq *DeleteServiceTokenAccessRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, serviceTokenAccessAPIPath(delReq.Organization, delReq.ID), delReq)
	if err != nil {
		return err
	}

	err = s.client.do(ctx, req, nil)
	return err
}

type CreateServiceTokenRequest struct {
	Organization string  `json:"-"`
	Name         *string `json:"name,omitempty"`
	TTL          *int    `json:"ttl,omitempty"`
}

type GetServiceTokenRequest struct {
	Organization string `json:"-"`
	ID           string `json:"-"`
}

type ListServiceTokenGrantsRequest struct {
	Organization string `json:"-"`
	ID           string `json:"-"`
}

type DeleteServiceTokenRequest struct {
	Organization string `json:"-"`
	ID           string `json:"-"`
}

type ListServiceTokensRequest struct {
	Organization string `json:"-"`
}

type GetServiceTokenAccessRequest struct {
	Organization string `json:"-"`
	ID           string `json:"-"`
}

type AddServiceTokenAccessRequest struct {
	Organization string   `json:"-"`
	ID           string   `json:"-"`
	Database     string   `json:"database"`
	Accesses     []string `json:"access"`
}

type DeleteServiceTokenAccessRequest struct {
	Organization string   `json:"-"`
	ID           string   `json:"-"`
	Database     string   `json:"database"`
	Accesses     []string `json:"access"`
}

type ServiceToken struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Token      string     `json:"token"`
	Name       *string    `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

type ServiceTokenGrant struct {
	ID           string                     `json:"id"`
	ResourceName string                     `json:"resource_name"`
	ResourceType string                     `json:"resource_type"`
	ResourceID   string                     `json:"resource_id"`
	Accesses     []*ServiceTokenGrantAccess `json:"accesses"`
}

type ServiceTokenGrantAccess struct {
	Access      string `json:"access"`
	Description string `json:"description"`
}

type serviceTokensResponse struct {
	ServiceTokens []*ServiceToken `json:"data"`
}

type ServiceTokenAccess struct {
	ID       string               `json:"id"`
	Access   string               `json:"access"`
	Type     string               `json:"type"`
	Resource ServiceTokenResource `json:"resource"`
}

type ServiceTokenResource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type serviceTokenAccessResponse struct {
	ServiceTokenAccesses []*ServiceTokenAccess `json:"data"`
}

type serviceTokenGrantsResponse struct {
	ServiceTokenGrants []*ServiceTokenGrant `json:"data"`
}

func serviceTokenAccessAPIPath(org, id string) string {
	return path.Join(serviceTokensAPIPath(org), id, "access")
}

func serviceTokenGrantsAPIPath(org, id string) string {
	return path.Join(serviceTokensAPIPath(org), id, "grants")
}

func serviceTokensAPIPath(org string) string {
	return path.Join("v1/organizations", org, "service-tokens")
}

func serviceTokenAPIPath(org, id string) string {
	return path.Join(serviceTokensAPIPath(org), id)
}
