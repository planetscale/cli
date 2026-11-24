package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// OrganizationInvitation is a pending invite to join an organization.
// Role is a built-in name or a custom role slug.
type OrganizationInvitation struct {
	ID        string                  `json:"id"`
	Email     string                  `json:"email"`
	Role      string                  `json:"role"`
	Sender    OrganizationMemberUser  `json:"sender"`
	Recipient *OrganizationMemberUser `json:"recipient"`
	Expired   bool                    `json:"expired"`
	ExpiresAt time.Time               `json:"expires_at"`
	CreatedAt time.Time               `json:"created_at"`
	AvatarURL string                  `json:"avatar_url"`
}

type organizationInvitationsResponse struct {
	Data []*OrganizationInvitation `json:"data"`
}

type ListOrganizationInvitationsRequest struct {
	Organization string
}

type CreateOrganizationInvitationRequest struct {
	Organization string `json:"-"`
	Email        string `json:"email"`
	Role         string `json:"role,omitempty"`
}

type CancelOrganizationInvitationRequest struct {
	Organization string
	ID           string
}

func organizationInvitationsAPIPath(org string) string {
	return path.Join(organizationsAPIPath, org, "invitations")
}

func organizationInvitationAPIPath(org, id string) string {
	return path.Join(organizationInvitationsAPIPath(org), id)
}

func (o *organizationsService) ListInvitations(ctx context.Context, listReq *ListOrganizationInvitationsRequest, opts ...ListOption) ([]*OrganizationInvitation, error) {
	listOpts := defaultListOptions(WithPerPage(100))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}

	req, err := o.client.newRequest(http.MethodGet, organizationInvitationsAPIPath(listReq.Organization), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list organization invitations: %w", err)
	}

	resp := &organizationInvitationsResponse{}
	if err := o.client.do(ctx, req, resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (o *organizationsService) CreateInvitation(ctx context.Context, createReq *CreateOrganizationInvitationRequest) (*OrganizationInvitation, error) {
	req, err := o.client.newRequest(http.MethodPost, organizationInvitationsAPIPath(createReq.Organization), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for create organization invitation: %w", err)
	}

	invitation := &OrganizationInvitation{}
	if err := o.client.do(ctx, req, invitation); err != nil {
		return nil, err
	}
	return invitation, nil
}

func (o *organizationsService) CancelInvitation(ctx context.Context, cancelReq *CancelOrganizationInvitationRequest) error {
	req, err := o.client.newRequest(http.MethodDelete, organizationInvitationAPIPath(cancelReq.Organization, cancelReq.ID), nil)
	if err != nil {
		return fmt.Errorf("error creating request for cancel organization invitation: %w", err)
	}
	return o.client.do(ctx, req, nil)
}
