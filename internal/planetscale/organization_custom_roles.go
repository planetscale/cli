package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"time"
)

// OrganizationCustomRole is an organization-defined role. Members
// reference it by Slug in the shared `role` field.
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

type CreateOrganizationCustomRoleRequest struct {
	Organization string   `json:"-"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Permissions  []string `json:"permissions"`
}

type UpdateOrganizationCustomRoleRequest struct {
	Organization string
	Slug         string
	Name         *string
	Description  *string
	Permissions  *[]string
}

func (r *UpdateOrganizationCustomRoleRequest) MarshalJSON() ([]byte, error) {
	body := map[string]any{}
	if r.Name != nil {
		body["name"] = *r.Name
	}
	if r.Description != nil {
		body["description"] = *r.Description
	}
	if r.Permissions != nil {
		body["permissions"] = *r.Permissions
	}
	return json.Marshal(body)
}

type DeleteOrganizationCustomRoleRequest struct {
	Organization string
	Slug         string
}

func organizationCustomRolesAPIPath(org string) string {
	return path.Join(organizationsAPIPath, org, "custom-roles")
}

func organizationCustomRoleAPIPath(org, slug string) string {
	return path.Join(organizationCustomRolesAPIPath(org), slug)
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

func (o *organizationsService) CreateCustomRole(ctx context.Context, createReq *CreateOrganizationCustomRoleRequest) (*OrganizationCustomRole, error) {
	req, err := o.client.newRequest(http.MethodPost, organizationCustomRolesAPIPath(createReq.Organization), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for create organization custom role: %w", err)
	}

	role := &OrganizationCustomRole{}
	if err := o.client.do(ctx, req, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (o *organizationsService) UpdateCustomRole(ctx context.Context, updateReq *UpdateOrganizationCustomRoleRequest) (*OrganizationCustomRole, error) {
	req, err := o.client.newRequest(http.MethodPatch, organizationCustomRoleAPIPath(updateReq.Organization, updateReq.Slug), updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for update organization custom role: %w", err)
	}

	role := &OrganizationCustomRole{}
	if err := o.client.do(ctx, req, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (o *organizationsService) DeleteCustomRole(ctx context.Context, deleteReq *DeleteOrganizationCustomRoleRequest) error {
	req, err := o.client.newRequest(http.MethodDelete, organizationCustomRoleAPIPath(deleteReq.Organization, deleteReq.Slug), nil)
	if err != nil {
		return fmt.Errorf("error creating request for delete organization custom role: %w", err)
	}
	return o.client.do(ctx, req, nil)
}
