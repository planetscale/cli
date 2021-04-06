package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const organizationsAPIPath = "v1/organizations"

// GetOrganizationRequest encapsulates the request for getting a single
// organization.
type GetOrganizationRequest struct {
	Organization string
}

// OrganizationsService is an interface for communicating with the PlanetScale
// Organizations API endpoints.
type OrganizationsService interface {
	Get(context.Context, *GetOrganizationRequest) (*Organization, error)
	List(context.Context) ([]*Organization, error)
}

// Organization represents a PlanetScale organization.
type Organization struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type organizationsResponse struct {
	Organizations []*Organization `json:"data"`
}

type organizationsService struct {
	client *Client
}

var _ OrganizationsService = &organizationsService{}

func NewOrganizationsService(client *Client) *organizationsService {
	return &organizationsService{
		client: client,
	}
}

// Get fetches a single organization by name.
func (o *organizationsService) Get(ctx context.Context, getReq *GetOrganizationRequest) (*Organization, error) {
	req, err := o.client.newRequest(http.MethodGet, fmt.Sprintf("%s/%s", organizationsAPIPath, getReq.Organization), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for get organization")
	}

	org := &Organization{}
	if err := o.client.do(ctx, req, &org); err != nil {
		return nil, err
	}

	return org, nil
}

// List returns all the organizations for a user.
func (o *organizationsService) List(ctx context.Context) ([]*Organization, error) {
	req, err := o.client.newRequest(http.MethodGet, organizationsAPIPath, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for list organization")
	}

	orgResponse := &organizationsResponse{}
	if err := o.client.do(ctx, req, &orgResponse); err != nil {
		return nil, err
	}

	return orgResponse.Organizations, nil
}
