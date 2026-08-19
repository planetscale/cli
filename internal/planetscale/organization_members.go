package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

// OrganizationMembership is a user's membership in an organization.
// ID is the membership public id. PATCH/DELETE/GET member routes take the
// nested user's public id (User.ID), not this ID.
type OrganizationMembership struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"`
	User      OrganizationMemberUser `json:"user"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// OrganizationMemberUser is the user nested on an organization membership.
type OrganizationMemberUser struct {
	ID                      string    `json:"id"`
	DisplayName             string    `json:"display_name"`
	Name                    string    `json:"name"`
	Email                   string    `json:"email"`
	AvatarURL               string    `json:"avatar_url"`
	TwoFactorAuthConfigured bool      `json:"two_factor_auth_configured"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type organizationMembersResponse struct {
	Data []*OrganizationMembership `json:"data"`
}

type ListOrganizationMembersRequest struct {
	Organization string
	Query        string
}

type GetOrganizationMemberRequest struct {
	Organization string
	UserID       string
}

type UpdateOrganizationMemberRequest struct {
	Organization string `json:"-"`
	UserID       string `json:"-"`
	Role         string `json:"role"`
}

type RemoveOrganizationMemberRequest struct {
	Organization        string
	UserID              string
	DeletePasswords     bool
	DeleteServiceTokens bool
}

func organizationMembersAPIPath(org string) string {
	return path.Join(organizationsAPIPath, org, "members")
}

func organizationMemberAPIPath(org, userID string) string {
	return path.Join(organizationMembersAPIPath(org), userID)
}

func (o *organizationsService) ListMembers(ctx context.Context, listReq *ListOrganizationMembersRequest, opts ...ListOption) ([]*OrganizationMembership, error) {
	defaultOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(defaultOpts); err != nil {
			return nil, err
		}
	}
	if listReq.Query != "" {
		defaultOpts.URLValues.Set("q", listReq.Query)
	}

	req, err := o.client.newRequest(http.MethodGet, organizationMembersAPIPath(listReq.Organization), nil, WithQueryParams(*defaultOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list organization members: %w", err)
	}

	resp := &organizationMembersResponse{}
	if err := o.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (o *organizationsService) GetMember(ctx context.Context, getReq *GetOrganizationMemberRequest) (*OrganizationMembership, error) {
	req, err := o.client.newRequest(http.MethodGet, organizationMemberAPIPath(getReq.Organization, getReq.UserID), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get organization member: %w", err)
	}

	membership := &OrganizationMembership{}
	if err := o.client.do(ctx, req, &membership); err != nil {
		return nil, err
	}

	return membership, nil
}

func (o *organizationsService) UpdateMember(ctx context.Context, updateReq *UpdateOrganizationMemberRequest) (*OrganizationMembership, error) {
	body := struct {
		Role string `json:"role"`
	}{Role: updateReq.Role}

	req, err := o.client.newRequest(http.MethodPatch, organizationMemberAPIPath(updateReq.Organization, updateReq.UserID), body)
	if err != nil {
		return nil, fmt.Errorf("error creating request for update organization member: %w", err)
	}

	membership := &OrganizationMembership{}
	if err := o.client.do(ctx, req, &membership); err != nil {
		return nil, err
	}

	return membership, nil
}

func (o *organizationsService) RemoveMember(ctx context.Context, removeReq *RemoveOrganizationMemberRequest) error {
	v := url.Values{}
	if removeReq.DeletePasswords {
		v.Set("delete_passwords", "true")
	}
	if removeReq.DeleteServiceTokens {
		v.Set("delete_service_tokens", "true")
	}

	req, err := o.client.newRequest(http.MethodDelete, organizationMemberAPIPath(removeReq.Organization, removeReq.UserID), nil, WithQueryParams(v))
	if err != nil {
		return fmt.Errorf("error creating request for remove organization member: %w", err)
	}

	return o.client.do(ctx, req, nil)
}
