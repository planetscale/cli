package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// OrganizationSSOService is an interface for communicating with the PlanetScale
// organization SSO API endpoints.
type OrganizationSSOService interface {
	Get(context.Context, *OrganizationSSORequest) (*OrganizationSSO, error)
	Enable(context.Context, *OrganizationSSORequest) (*OrganizationSSO, error)
	Disable(context.Context, *OrganizationSSORequest) (*OrganizationSSO, error)
	Configure(context.Context, *OrganizationSSORequest) (*SSOPortal, error)
	EnableDirectory(context.Context, *OrganizationSSORequest) (*SSOPortal, error)
	DisableDirectory(context.Context, *OrganizationSSORequest) (*OrganizationSSO, error)
	ListDomains(context.Context, *OrganizationSSORequest) ([]*OrganizationDomain, error)
	VerifyDomain(context.Context, *OrganizationSSORequest) (*SSOPortal, error)
	DeleteDomain(context.Context, *DeleteOrganizationSSODomainRequest) error
}

// OrganizationSSO is the SSO status for an organization.
type OrganizationSSO struct {
	ID                    string                `json:"id"`
	Type                  string                `json:"type"`
	Enabled               bool                  `json:"enabled"`
	Configured            bool                  `json:"configured"`
	Directory             bool                  `json:"directory"`
	HasVerifiedDomain     bool                  `json:"has_verified_domain"`
	Domains               []*OrganizationDomain `json:"domains"`
	DomainVerificationURL *string               `json:"domain_verification_url"`
}

// OrganizationDomain is an email domain registered for organization SSO.
type OrganizationDomain struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`
	Domain        string     `json:"domain"`
	State         string     `json:"state"`
	VerifiedAt    *time.Time `json:"verified_at"`
	FailureReason *string    `json:"failure_reason"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// SSOPortal is a URL for verifying a domain or configuring SSO.
type SSOPortal struct {
	PortalURL string `json:"portal_url"`
}

type OrganizationSSORequest struct {
	Organization string
}

type DeleteOrganizationSSODomainRequest struct {
	Organization string
	DomainID     string
}

type organizationSSOService struct {
	client *Client
}

var _ OrganizationSSOService = &organizationSSOService{}

func organizationSSOAPIPath(org string) string {
	return path.Join(organizationsAPIPath, org, "sso")
}

func (s *organizationSSOService) Get(ctx context.Context, getReq *OrganizationSSORequest) (*OrganizationSSO, error) {
	return s.doSSO(ctx, http.MethodGet, organizationSSOAPIPath(getReq.Organization), "get organization sso")
}

func (s *organizationSSOService) Enable(ctx context.Context, enableReq *OrganizationSSORequest) (*OrganizationSSO, error) {
	return s.doSSO(ctx, http.MethodPost, organizationSSOAPIPath(enableReq.Organization), "enable organization sso")
}

func (s *organizationSSOService) Disable(ctx context.Context, disableReq *OrganizationSSORequest) (*OrganizationSSO, error) {
	return s.doSSO(ctx, http.MethodDelete, organizationSSOAPIPath(disableReq.Organization), "disable organization sso")
}

func (s *organizationSSOService) Configure(ctx context.Context, configureReq *OrganizationSSORequest) (*SSOPortal, error) {
	return s.postPortal(ctx, path.Join(organizationSSOAPIPath(configureReq.Organization), "configure"), "configure organization sso")
}

func (s *organizationSSOService) EnableDirectory(ctx context.Context, enableReq *OrganizationSSORequest) (*SSOPortal, error) {
	return s.postPortal(ctx, path.Join(organizationSSOAPIPath(enableReq.Organization), "directory"), "enable organization sso directory")
}

func (s *organizationSSOService) DisableDirectory(ctx context.Context, disableReq *OrganizationSSORequest) (*OrganizationSSO, error) {
	return s.doSSO(ctx, http.MethodDelete, path.Join(organizationSSOAPIPath(disableReq.Organization), "directory"), "disable organization sso directory")
}

func (s *organizationSSOService) ListDomains(ctx context.Context, listReq *OrganizationSSORequest) ([]*OrganizationDomain, error) {
	req, err := s.client.newRequest(http.MethodGet, path.Join(organizationSSOAPIPath(listReq.Organization), "domains"), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for list organization sso domains: %w", err)
	}

	var domains []*OrganizationDomain
	if err := s.client.do(ctx, req, &domains); err != nil {
		return nil, err
	}
	return domains, nil
}

func (s *organizationSSOService) VerifyDomain(ctx context.Context, verifyReq *OrganizationSSORequest) (*SSOPortal, error) {
	return s.postPortal(ctx, path.Join(organizationSSOAPIPath(verifyReq.Organization), "domains"), "verify organization sso domain")
}

func (s *organizationSSOService) DeleteDomain(ctx context.Context, deleteReq *DeleteOrganizationSSODomainRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, path.Join(organizationSSOAPIPath(deleteReq.Organization), "domains", deleteReq.DomainID), nil)
	if err != nil {
		return fmt.Errorf("error creating request for delete organization sso domain: %w", err)
	}
	return s.client.do(ctx, req, nil)
}

func (s *organizationSSOService) doSSO(ctx context.Context, method, apiPath, action string) (*OrganizationSSO, error) {
	req, err := s.client.newRequest(method, apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for %s: %w", action, err)
	}

	sso := &OrganizationSSO{}
	if err := s.client.do(ctx, req, sso); err != nil {
		return nil, err
	}
	return sso, nil
}

func (s *organizationSSOService) postPortal(ctx context.Context, apiPath, action string) (*SSOPortal, error) {
	req, err := s.client.newRequest(http.MethodPost, apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for %s: %w", action, err)
	}

	portal := &SSOPortal{}
	if err := s.client.do(ctx, req, portal); err != nil {
		return nil, err
	}
	return portal, nil
}
