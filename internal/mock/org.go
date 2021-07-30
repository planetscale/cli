package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type OrganizationsService struct {
	GetFn        func(context.Context, *ps.GetOrganizationRequest) (*ps.Organization, error)
	GetFnInvoked bool

	ListFn        func(context.Context) ([]*ps.Organization, error)
	ListFnInvoked bool

	ListRegionsFn        func(context.Context, *ps.ListOrganizationRegionsRequest) ([]*ps.Region, error)
	ListRegionsFnInvoked bool
}

func (o *OrganizationsService) Get(ctx context.Context, req *ps.GetOrganizationRequest) (*ps.Organization, error) {
	o.GetFnInvoked = true
	return o.GetFn(ctx, req)
}

func (o *OrganizationsService) List(ctx context.Context) ([]*ps.Organization, error) {
	o.ListFnInvoked = true
	return o.ListFn(ctx)
}

func (o *OrganizationsService) ListRegions(ctx context.Context, req *ps.ListOrganizationRegionsRequest) ([]*ps.Region, error) {
	o.ListRegionsFnInvoked = true
	return o.ListRegionsFn(ctx, req)
}
