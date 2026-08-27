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
	Get(context.Context, *GetOrganizationSSORequest) (*OrganizationSSO, error)
	Enable(context.Context, *EnableOrganizationSSORequest) (*OrganizationSSO, error)
	Disable(context.Context, *DisableOrganizationSSORequest) (*OrganizationSSO, error)
	Configure(context.Context, *ConfigureOrganizationSSORequest) (*SSOPortal, error)
	EnableDirectory(context.Context, *EnableOrganizationSSODirectoryRequest) (*SSOPortal, error)
	DisableDirectory(context.Context, *DisableOrganizationSSODirectoryRequest) (*OrganizationSSO, error)
	ListDomains(context.Context, *ListOrganizationSSODomainsRequest) ([]*OrganizationDomain, error)
	VerifyDomain(context.Context, *VerifyOrganizationSSODomainRequest) (*SSOPortal, error)
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

// SSOPortal is a one-time URL for verifying a domain or configuring SSO.
type SSOPortal struct {
	PortalURL string `json:"portal_url"`
}

type GetOrganizationSSORequest struct {
	Organization string
}

type EnableOrganizationSSORequest struct {
	Organization string
}

type DisableOrganizationSSORequest struct {
	Organization string
}

type ConfigureOrganizationSSORequest struct {
	Organization string
}

type EnableOrganizationSSODirectoryRequest struct {
	Organization string
}

type DisableOrganizationSSODirectoryRequest struct {
	Organization string
}

type ListOrganizationSSODomainsRequest struct {
	Organization string
}

type VerifyOrganizationSSODomainRequest struct {
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

func organizationSSOConfigureAPIPath(org string) string {
	return path.Join(organizationSSOAPIPath(org), "configure")
}

func organizationSSODirectoryAPIPath(org string) string {
	return path.Join(organizationSSOAPIPath(org), "directory")
}

func organizationSSODomainsAPIPath(org string) string {
	return path.Join(organizationSSOAPIPath(org), "domains")
}

func organizationSSODomainAPIPath(org, domainID string) string {
	return path.Join(organizationSSODomainsAPIPath(org), domainID)
}

func (s *organizationSSOService) Get(ctx context.Context, getReq *GetOrganizationSSORequest) (*OrganizationSSO, error) {
	req, err := s.client.newRequest(http.MethodGet, organizationSSOAPIPath(getReq.Organization), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get organization sso: %w", err)
	}

	sso := &OrganizationSSO{}
	if err := s.client.do(ctx, req, sso); err != nil {
		return nil, err
	}
	return sso, nil
}

func (s *organizationSSOService) Enable(ctx context.Context, enableReq *EnableOrganizationSSORequest) (*OrganizationSSO, error) {
	req, err := s.client.newRequest(http.MethodPost, organizationSSOAPIPath(enableReq.Organization), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for enable organization sso: %w", err)
	}

	sso := &OrganizationSSO{}
	if err := s.client.do(ctx, req, sso); err != nil {
		return nil, err
	}
	return sso, nil
}

func (s *organizationSSOService) Disable(ctx context.Context, disableReq *DisableOrganizationSSORequest) (*OrganizationSSO, error) {
	req, err := s.client.newRequest(http.MethodDelete, organizationSSOAPIPath(disableReq.Organization), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for disable organization sso: %w", err)
	}

	sso := &OrganizationSSO{}
	if err := s.client.do(ctx, req, sso); err != nil {
		return nil, err
	}
	return sso, nil
}

func (s *organizationSSOService) Configure(ctx context.Context, configureReq *ConfigureOrganizationSSORequest) (*SSOPortal, error) {
	return s.postPortal(ctx, organizationSSOConfigureAPIPath(configureReq.Organization), "configure organization sso")
}

func (s *organizationSSOService) EnableDirectory(ctx context.Context, enableReq *EnableOrganizationSSODirectoryRequest) (*SSOPortal, error) {
	return s.postPortal(ctx, organizationSSODirectoryAPIPath(enableReq.Organization), "enable organization sso directory")
}

func (s *organizationSSOService) DisableDirectory(ctx context.Context, disableReq *DisableOrganizationSSODirectoryRequest) (*OrganizationSSO, error) {
	req, err := s.client.newRequest(http.MethodDelete, organizationSSODirectoryAPIPath(disableReq.Organization), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for disable organization sso directory: %w", err)
	}

	sso := &OrganizationSSO{}
	if err := s.client.do(ctx, req, sso); err != nil {
		return nil, err
	}
	return sso, nil
}

func (s *organizationSSOService) ListDomains(ctx context.Context, listReq *ListOrganizationSSODomainsRequest) ([]*OrganizationDomain, error) {
	req, err := s.client.newRequest(http.MethodGet, organizationSSODomainsAPIPath(listReq.Organization), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for list organization sso domains: %w", err)
	}

	var domains []*OrganizationDomain
	if err := s.client.do(ctx, req, &domains); err != nil {
		return nil, err
	}
	return domains, nil
}

func (s *organizationSSOService) VerifyDomain(ctx context.Context, verifyReq *VerifyOrganizationSSODomainRequest) (*SSOPortal, error) {
	return s.postPortal(ctx, organizationSSODomainsAPIPath(verifyReq.Organization), "verify organization sso domain")
}

func (s *organizationSSOService) DeleteDomain(ctx context.Context, deleteReq *DeleteOrganizationSSODomainRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, organizationSSODomainAPIPath(deleteReq.Organization, deleteReq.DomainID), nil)
	if err != nil {
		return fmt.Errorf("error creating request for delete organization sso domain: %w", err)
	}
	return s.client.do(ctx, req, nil)
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
