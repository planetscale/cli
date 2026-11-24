package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// OrganizationCustomRole is an organization-defined role. Members and
// invitations reference it by Slug in the shared `role` field.
type OrganizationCustomRole struct {
	ID          string                             `json:"id"`
	Name        string                             `json:"name"`
	Slug        string                             `json:"slug"`
	Description *string                            `json:"description"`
	Permissions []OrganizationCustomRolePermission `json:"permissions"`
	CreatedAt   time.Time                          `json:"created_at"`
	UpdatedAt   time.Time                          `json:"updated_at"`
}

type OrganizationCustomRolePermission struct {
	Action string `json:"action"`
}

type organizationCustomRolesResponse struct {
	Data []*OrganizationCustomRole `json:"data"`
}

type ListOrganizationCustomRolesRequest struct {
	Organization string
}

func organizationCustomRolesAPIPath(org string) string {
	return path.Join(organizationsAPIPath, org, "custom-roles")
}

func (o *organizationsService) ListCustomRoles(ctx context.Context, listReq *ListOrganizationCustomRolesRequest, opts ...ListOption) ([]*OrganizationCustomRole, error) {
	listOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}

	req, err := o.client.newRequest(http.MethodGet, organizationCustomRolesAPIPath(listReq.Organization), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list organization custom roles: %w", err)
	}

	resp := &organizationCustomRolesResponse{}
	if err := o.client.do(ctx, req, resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}
