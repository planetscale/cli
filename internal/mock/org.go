package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type OrganizationsService struct {
	GetFn        func(context.Context, *ps.GetOrganizationRequest) (*ps.Organization, error)
	GetFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdateOrganizationRequest) (*ps.Organization, error)
	UpdateFnInvoked bool

	ListFn        func(context.Context) ([]*ps.Organization, error)
	ListFnInvoked bool

	ListRegionsFn        func(context.Context, *ps.ListOrganizationRegionsRequest) ([]*ps.Region, error)
	ListRegionsFnInvoked bool

	ListClusterSKUsFn        func(context.Context, *ps.ListOrganizationClusterSKUsRequest, ...ps.ListOption) ([]*ps.ClusterSKU, error)
	ListClusterSKUsFnInvoked bool

	ListMembersFn        func(context.Context, *ps.ListOrganizationMembersRequest, ...ps.ListOption) ([]*ps.OrganizationMembership, error)
	ListMembersFnInvoked bool

	GetMemberFn        func(context.Context, *ps.GetOrganizationMemberRequest) (*ps.OrganizationMembership, error)
	GetMemberFnInvoked bool

	UpdateMemberFn        func(context.Context, *ps.UpdateOrganizationMemberRequest) (*ps.OrganizationMembership, error)
	UpdateMemberFnInvoked bool

	RemoveMemberFn        func(context.Context, *ps.RemoveOrganizationMemberRequest) error
	RemoveMemberFnInvoked bool
}

func (o *OrganizationsService) Get(ctx context.Context, req *ps.GetOrganizationRequest) (*ps.Organization, error) {
	o.GetFnInvoked = true
	return o.GetFn(ctx, req)
}

func (o *OrganizationsService) Update(ctx context.Context, req *ps.UpdateOrganizationRequest) (*ps.Organization, error) {
	o.UpdateFnInvoked = true
	return o.UpdateFn(ctx, req)
}

func (o *OrganizationsService) List(ctx context.Context) ([]*ps.Organization, error) {
	o.ListFnInvoked = true
	return o.ListFn(ctx)
}

func (o *OrganizationsService) ListRegions(ctx context.Context, req *ps.ListOrganizationRegionsRequest) ([]*ps.Region, error) {
	o.ListRegionsFnInvoked = true
	return o.ListRegionsFn(ctx, req)
}

func (o *OrganizationsService) ListClusterSKUs(ctx context.Context, req *ps.ListOrganizationClusterSKUsRequest, opts ...ps.ListOption) ([]*ps.ClusterSKU, error) {
	o.ListClusterSKUsFnInvoked = true
	return o.ListClusterSKUsFn(ctx, req, opts...)
}

func (o *OrganizationsService) ListMembers(ctx context.Context, req *ps.ListOrganizationMembersRequest, opts ...ps.ListOption) ([]*ps.OrganizationMembership, error) {
	o.ListMembersFnInvoked = true
	return o.ListMembersFn(ctx, req, opts...)
}

func (o *OrganizationsService) GetMember(ctx context.Context, req *ps.GetOrganizationMemberRequest) (*ps.OrganizationMembership, error) {
	o.GetMemberFnInvoked = true
	return o.GetMemberFn(ctx, req)
}

func (o *OrganizationsService) UpdateMember(ctx context.Context, req *ps.UpdateOrganizationMemberRequest) (*ps.OrganizationMembership, error) {
	o.UpdateMemberFnInvoked = true
	return o.UpdateMemberFn(ctx, req)
}

func (o *OrganizationsService) RemoveMember(ctx context.Context, req *ps.RemoveOrganizationMemberRequest) error {
	o.RemoveMemberFnInvoked = true
	return o.RemoveMemberFn(ctx, req)
}
