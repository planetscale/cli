package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

type OrganizationTeamActor struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

type OrganizationTeamUser struct {
	ID                      string          `json:"id"`
	DisplayName             string          `json:"display_name"`
	Name                    string          `json:"name"`
	Email                   string          `json:"email"`
	AvatarURL               string          `json:"avatar_url"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at"`
	TwoFactorAuthConfigured bool            `json:"two_factor_auth_configured"`
	DefaultOrganization     json.RawMessage `json:"default_organization"`
	SSO                     *bool           `json:"sso"`
	Managed                 *bool           `json:"managed"`
	DirectoryManaged        *bool           `json:"directory_managed"`
	EmailVerified           *bool           `json:"email_verified"`
}

type OrganizationTeamDatabase struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	BranchesURL string `json:"branches_url"`
}

type OrganizationTeam struct {
	ID               string                     `json:"id"`
	DisplayName      string                     `json:"display_name"`
	Creator          OrganizationTeamActor      `json:"creator"`
	Members          []OrganizationTeamUser     `json:"members"`
	Databases        []OrganizationTeamDatabase `json:"databases"`
	AnalystDatabases []OrganizationTeamDatabase `json:"analyst_databases"`
	Name             string                     `json:"name"`
	Slug             string                     `json:"slug"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
	Description      *string                    `json:"description"`
	Managed          bool                       `json:"managed"`
}

type OrganizationTeamMembership struct {
	ID        string                `json:"id"`
	User      OrganizationTeamUser  `json:"user"`
	Actor     OrganizationTeamActor `json:"actor"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
	Passwords []json.RawMessage     `json:"passwords"`
}

type organizationTeamsResponse struct {
	Data []*OrganizationTeam `json:"data"`
}

type organizationTeamMembersResponse struct {
	Data []*OrganizationTeamMembership `json:"data"`
}

type ListOrganizationTeamsRequest struct {
	Organization string
	Query        string
}

type GetOrganizationTeamRequest struct {
	Organization string
	Team         string
}

type CreateOrganizationTeamRequest struct {
	Organization string `json:"-"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
}

type UpdateOrganizationTeamRequest struct {
	Organization string  `json:"-"`
	Team         string  `json:"-"`
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
}

type DeleteOrganizationTeamRequest struct {
	Organization string
	Team         string
}

type ListOrganizationTeamMembersRequest struct {
	Organization string
	Team         string
}

type AddOrganizationTeamMemberRequest struct {
	Organization string `json:"-"`
	Team         string `json:"-"`
	UserID       string `json:"user_id"`
}

type RemoveOrganizationTeamMemberRequest struct {
	Organization    string
	Team            string
	ID              string
	DeletePasswords bool
}

func organizationTeamsAPIPath(org string) string {
	return path.Join(organizationsAPIPath, org, "teams")
}

func organizationTeamAPIPath(org, team string) string {
	return path.Join(organizationTeamsAPIPath(org), team)
}

func organizationTeamMembersAPIPath(org, team string) string {
	return path.Join(organizationTeamAPIPath(org, team), "members")
}

func (o *organizationsService) ListTeams(ctx context.Context, listReq *ListOrganizationTeamsRequest, opts ...ListOption) ([]*OrganizationTeam, error) {
	listOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}
	if listReq.Query != "" {
		listOpts.URLValues.Set("q", listReq.Query)
	}

	req, err := o.client.newRequest(http.MethodGet, organizationTeamsAPIPath(listReq.Organization), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list organization teams: %w", err)
	}

	resp := &organizationTeamsResponse{}
	if err := o.client.do(ctx, req, resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (o *organizationsService) GetTeam(ctx context.Context, getReq *GetOrganizationTeamRequest) (*OrganizationTeam, error) {
	req, err := o.client.newRequest(http.MethodGet, organizationTeamAPIPath(getReq.Organization, getReq.Team), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get organization team: %w", err)
	}

	team := &OrganizationTeam{}
	if err := o.client.do(ctx, req, team); err != nil {
		return nil, err
	}
	return team, nil
}

func (o *organizationsService) CreateTeam(ctx context.Context, createReq *CreateOrganizationTeamRequest) (*OrganizationTeam, error) {
	req, err := o.client.newRequest(http.MethodPost, organizationTeamsAPIPath(createReq.Organization), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for create organization team: %w", err)
	}

	team := &OrganizationTeam{}
	if err := o.client.do(ctx, req, team); err != nil {
		return nil, err
	}
	return team, nil
}

func (o *organizationsService) UpdateTeam(ctx context.Context, updateReq *UpdateOrganizationTeamRequest) (*OrganizationTeam, error) {
	req, err := o.client.newRequest(http.MethodPatch, organizationTeamAPIPath(updateReq.Organization, updateReq.Team), updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for update organization team: %w", err)
	}

	team := &OrganizationTeam{}
	if err := o.client.do(ctx, req, team); err != nil {
		return nil, err
	}
	return team, nil
}

func (o *organizationsService) DeleteTeam(ctx context.Context, deleteReq *DeleteOrganizationTeamRequest) error {
	req, err := o.client.newRequest(http.MethodDelete, organizationTeamAPIPath(deleteReq.Organization, deleteReq.Team), nil)
	if err != nil {
		return fmt.Errorf("error creating request for delete organization team: %w", err)
	}
	return o.client.do(ctx, req, nil)
}

func (o *organizationsService) ListTeamMembers(ctx context.Context, listReq *ListOrganizationTeamMembersRequest, opts ...ListOption) ([]*OrganizationTeamMembership, error) {
	listOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}

	req, err := o.client.newRequest(http.MethodGet, organizationTeamMembersAPIPath(listReq.Organization, listReq.Team), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list organization team members: %w", err)
	}

	resp := &organizationTeamMembersResponse{}
	if err := o.client.do(ctx, req, resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (o *organizationsService) AddTeamMember(ctx context.Context, addReq *AddOrganizationTeamMemberRequest) (*OrganizationTeamMembership, error) {
	req, err := o.client.newRequest(http.MethodPost, organizationTeamMembersAPIPath(addReq.Organization, addReq.Team), addReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for add organization team member: %w", err)
	}

	membership := &OrganizationTeamMembership{}
	if err := o.client.do(ctx, req, membership); err != nil {
		return nil, err
	}
	return membership, nil
}

func (o *organizationsService) RemoveTeamMember(ctx context.Context, removeReq *RemoveOrganizationTeamMemberRequest) error {
	values := url.Values{}
	if removeReq.DeletePasswords {
		values.Set("delete_passwords", "true")
	}
	apiPath := path.Join(organizationTeamMembersAPIPath(removeReq.Organization, removeReq.Team), removeReq.ID)
	req, err := o.client.newRequest(http.MethodDelete, apiPath, nil, WithQueryParams(values))
	if err != nil {
		return fmt.Errorf("error creating request for remove organization team member: %w", err)
	}
	return o.client.do(ctx, req, nil)
}
