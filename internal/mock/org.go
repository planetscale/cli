package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type OrganizationsService struct {
	GetFn        func(context.Context, *ps.GetOrganizationRequest) (*ps.Organization, error)
	GetFnInvoked bool

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

	ListTeamsFn        func(context.Context, *ps.ListOrganizationTeamsRequest, ...ps.ListOption) ([]*ps.OrganizationTeam, error)
	ListTeamsFnInvoked bool

	GetTeamFn        func(context.Context, *ps.GetOrganizationTeamRequest) (*ps.OrganizationTeam, error)
	GetTeamFnInvoked bool

	CreateTeamFn        func(context.Context, *ps.CreateOrganizationTeamRequest) (*ps.OrganizationTeam, error)
	CreateTeamFnInvoked bool

	UpdateTeamFn        func(context.Context, *ps.UpdateOrganizationTeamRequest) (*ps.OrganizationTeam, error)
	UpdateTeamFnInvoked bool

	DeleteTeamFn        func(context.Context, *ps.DeleteOrganizationTeamRequest) error
	DeleteTeamFnInvoked bool

	ListTeamMembersFn        func(context.Context, *ps.ListOrganizationTeamMembersRequest, ...ps.ListOption) ([]*ps.OrganizationTeamMembership, error)
	ListTeamMembersFnInvoked bool

	AddTeamMemberFn        func(context.Context, *ps.AddOrganizationTeamMemberRequest) (*ps.OrganizationTeamMembership, error)
	AddTeamMemberFnInvoked bool

	RemoveTeamMemberFn        func(context.Context, *ps.RemoveOrganizationTeamMemberRequest) error
	RemoveTeamMemberFnInvoked bool
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

func (o *OrganizationsService) ListTeams(ctx context.Context, req *ps.ListOrganizationTeamsRequest, opts ...ps.ListOption) ([]*ps.OrganizationTeam, error) {
	o.ListTeamsFnInvoked = true
	return o.ListTeamsFn(ctx, req, opts...)
}

func (o *OrganizationsService) GetTeam(ctx context.Context, req *ps.GetOrganizationTeamRequest) (*ps.OrganizationTeam, error) {
	o.GetTeamFnInvoked = true
	return o.GetTeamFn(ctx, req)
}

func (o *OrganizationsService) CreateTeam(ctx context.Context, req *ps.CreateOrganizationTeamRequest) (*ps.OrganizationTeam, error) {
	o.CreateTeamFnInvoked = true
	return o.CreateTeamFn(ctx, req)
}

func (o *OrganizationsService) UpdateTeam(ctx context.Context, req *ps.UpdateOrganizationTeamRequest) (*ps.OrganizationTeam, error) {
	o.UpdateTeamFnInvoked = true
	return o.UpdateTeamFn(ctx, req)
}

func (o *OrganizationsService) DeleteTeam(ctx context.Context, req *ps.DeleteOrganizationTeamRequest) error {
	o.DeleteTeamFnInvoked = true
	return o.DeleteTeamFn(ctx, req)
}

func (o *OrganizationsService) ListTeamMembers(ctx context.Context, req *ps.ListOrganizationTeamMembersRequest, opts ...ps.ListOption) ([]*ps.OrganizationTeamMembership, error) {
	o.ListTeamMembersFnInvoked = true
	return o.ListTeamMembersFn(ctx, req, opts...)
}

func (o *OrganizationsService) AddTeamMember(ctx context.Context, req *ps.AddOrganizationTeamMemberRequest) (*ps.OrganizationTeamMembership, error) {
	o.AddTeamMemberFnInvoked = true
	return o.AddTeamMemberFn(ctx, req)
}

func (o *OrganizationsService) RemoveTeamMember(ctx context.Context, req *ps.RemoveOrganizationTeamMemberRequest) error {
	o.RemoveTeamMemberFnInvoked = true
	return o.RemoveTeamMemberFn(ctx, req)
}
