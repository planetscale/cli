package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ServiceTokenService is an interface for communicating with the PlanetScale
// Service Token API.
type ServiceTokenService interface {
	Create(context.Context, *CreateServiceTokenRequest) (*ServiceToken, error)
	List(context.Context, *ListServiceTokensRequest) ([]*ServiceToken, error)
	Delete(context.Context, *DeleteServiceTokenRequest) error
	GetAccess(context.Context, *GetServiceTokenAccessRequest) ([]*ServiceTokenAccess, error)
	AddAccess(context.Context, *AddServiceTokenAccessRequest) ([]*ServiceTokenAccess, error)
	DeleteAccess(context.Context, *DeleteServiceTokenAccessRequest) error
}

type serviceTokenService struct {
	client *Client
}

func (s *serviceTokenService) Create(ctx context.Context, createReq *CreateServiceTokenRequest) (*ServiceToken, error) {
	req, err := s.client.newRequest(http.MethodPost, serviceTokensAPIPath(createReq.Organization), nil)
	if err != nil {
		return nil, err
	}

	res, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	st := &ServiceToken{}
	if err := json.NewDecoder(res.Body).Decode(st); err != nil {
		return nil, err
	}

	return st, nil
}

func (s *serviceTokenService) List(ctx context.Context, listReq *ListServiceTokensRequest) ([]*ServiceToken, error) {
	req, err := s.client.newRequest(http.MethodGet, serviceTokensAPIPath(listReq.Organization), nil)
	if err != nil {
		return nil, err
	}

	res, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	tokenListResponse := serviceTokensResponse{}
	if err := json.NewDecoder(res.Body).Decode(&tokenListResponse); err != nil {
		return nil, err
	}
	return tokenListResponse.ServiceTokens, nil
}

func (s *serviceTokenService) Delete(ctx context.Context, delReq *DeleteServiceTokenRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, serviceTokenAPIPath(delReq.Organization, delReq.ID), nil)
	if err != nil {
		return err
	}

	res, err := s.client.Do(ctx, req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

func (s *serviceTokenService) GetAccess(ctx context.Context, accessReq *GetServiceTokenAccessRequest) ([]*ServiceTokenAccess, error) {
	req, err := s.client.newRequest(http.MethodGet, serviceTokenAccessAPIPath(accessReq.Organization, accessReq.ID), nil)
	if err != nil {
		return nil, err
	}

	res, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	tokenAccess := serviceTokenAccessResponse{}
	if err := json.NewDecoder(res.Body).Decode(&tokenAccess); err != nil {
		return nil, err
	}
	return tokenAccess.ServiceTokenAccesses, nil
}

func (s *serviceTokenService) AddAccess(ctx context.Context, addReq *AddServiceTokenAccessRequest) ([]*ServiceTokenAccess, error) {
	req, err := s.client.newRequest(http.MethodPost, serviceTokenAccessAPIPath(addReq.Organization, addReq.ID), addReq)
	if err != nil {
		return nil, err
	}

	res, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	tokenAccess := serviceTokenAccessResponse{}
	if err := json.NewDecoder(res.Body).Decode(&tokenAccess); err != nil {
		return nil, err
	}
	return tokenAccess.ServiceTokenAccesses, nil
}

func (s *serviceTokenService) DeleteAccess(ctx context.Context, delReq *DeleteServiceTokenAccessRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, serviceTokenAccessAPIPath(delReq.Organization, delReq.ID), delReq)
	if err != nil {
		return err
	}

	res, err := s.client.Do(ctx, req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	return nil
}

var _ ServiceTokenService = &serviceTokenService{}

type CreateServiceTokenRequest struct {
	Organization string `json:"-"`
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
	ID    string `json:"id"`
	Type  string `json:"type"`
	Token string `json:"token"`
}

type serviceTokensResponse struct {
	ServiceTokens []*ServiceToken `json:"data"`
}

type ServiceTokenAccess struct {
	ID       int      `json:"id"`
	Access   string   `json:"access"`
	Type     string   `json:"type"`
	Resource Database `json:"resource"`
}

type serviceTokenAccessResponse struct {
	ServiceTokenAccesses []*ServiceTokenAccess `json:"data"`
}

func serviceTokenAccessAPIPath(org, id string) string {
	return fmt.Sprintf("%s/%s/access", serviceTokensAPIPath(org), id)
}

func serviceTokensAPIPath(org string) string {
	return fmt.Sprintf("v1/organizations/%s/service-tokens", org)
}

func serviceTokenAPIPath(org, id string) string {
	return fmt.Sprintf("%s/%s", serviceTokensAPIPath(org), id)
}
