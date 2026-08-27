package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type OrganizationSSOService struct {
	GetFn        func(context.Context, *ps.GetOrganizationSSORequest) (*ps.OrganizationSSO, error)
	GetFnInvoked bool

	EnableFn        func(context.Context, *ps.EnableOrganizationSSORequest) (*ps.OrganizationSSO, error)
	EnableFnInvoked bool

	DisableFn        func(context.Context, *ps.DisableOrganizationSSORequest) (*ps.OrganizationSSO, error)
	DisableFnInvoked bool

	ConfigureFn        func(context.Context, *ps.ConfigureOrganizationSSORequest) (*ps.SSOPortal, error)
	ConfigureFnInvoked bool

	EnableDirectoryFn        func(context.Context, *ps.EnableOrganizationSSODirectoryRequest) (*ps.SSOPortal, error)
	EnableDirectoryFnInvoked bool

	DisableDirectoryFn        func(context.Context, *ps.DisableOrganizationSSODirectoryRequest) (*ps.OrganizationSSO, error)
	DisableDirectoryFnInvoked bool

	ListDomainsFn        func(context.Context, *ps.ListOrganizationSSODomainsRequest) ([]*ps.OrganizationDomain, error)
	ListDomainsFnInvoked bool

	VerifyDomainFn        func(context.Context, *ps.VerifyOrganizationSSODomainRequest) (*ps.SSOPortal, error)
	VerifyDomainFnInvoked bool

	DeleteDomainFn        func(context.Context, *ps.DeleteOrganizationSSODomainRequest) error
	DeleteDomainFnInvoked bool
}

func (s *OrganizationSSOService) Get(ctx context.Context, req *ps.GetOrganizationSSORequest) (*ps.OrganizationSSO, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *OrganizationSSOService) Enable(ctx context.Context, req *ps.EnableOrganizationSSORequest) (*ps.OrganizationSSO, error) {
	s.EnableFnInvoked = true
	return s.EnableFn(ctx, req)
}

func (s *OrganizationSSOService) Disable(ctx context.Context, req *ps.DisableOrganizationSSORequest) (*ps.OrganizationSSO, error) {
	s.DisableFnInvoked = true
	return s.DisableFn(ctx, req)
}

func (s *OrganizationSSOService) Configure(ctx context.Context, req *ps.ConfigureOrganizationSSORequest) (*ps.SSOPortal, error) {
	s.ConfigureFnInvoked = true
	return s.ConfigureFn(ctx, req)
}

func (s *OrganizationSSOService) EnableDirectory(ctx context.Context, req *ps.EnableOrganizationSSODirectoryRequest) (*ps.SSOPortal, error) {
	s.EnableDirectoryFnInvoked = true
	return s.EnableDirectoryFn(ctx, req)
}

func (s *OrganizationSSOService) DisableDirectory(ctx context.Context, req *ps.DisableOrganizationSSODirectoryRequest) (*ps.OrganizationSSO, error) {
	s.DisableDirectoryFnInvoked = true
	return s.DisableDirectoryFn(ctx, req)
}

func (s *OrganizationSSOService) ListDomains(ctx context.Context, req *ps.ListOrganizationSSODomainsRequest) ([]*ps.OrganizationDomain, error) {
	s.ListDomainsFnInvoked = true
	return s.ListDomainsFn(ctx, req)
}

func (s *OrganizationSSOService) VerifyDomain(ctx context.Context, req *ps.VerifyOrganizationSSODomainRequest) (*ps.SSOPortal, error) {
	s.VerifyDomainFnInvoked = true
	return s.VerifyDomainFn(ctx, req)
}

func (s *OrganizationSSOService) DeleteDomain(ctx context.Context, req *ps.DeleteOrganizationSSODomainRequest) error {
	s.DeleteDomainFnInvoked = true
	return s.DeleteDomainFn(ctx, req)
}
