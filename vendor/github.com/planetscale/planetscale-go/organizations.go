package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const organizationsAPIPath = "v1/organizations"

// OrganizationsService is an interface for communicating with the PlanetScale
// Organizations API endpoints.
type OrganizationsService interface {
	Get(context.Context, string) (*Organization, error)
	List(context.Context) ([]*Organization, error)
}

// Organization represents a PlanetScale organization.
type Organization struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OrganizationsResponse struct {
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
func (o *organizationsService) Get(ctx context.Context, org string) (*Organization, error) {
	req, err := o.client.newRequest(http.MethodGet, fmt.Sprintf("%s/%s", organizationsAPIPath, org), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for get organization")
	}

	res, err := o.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	organization := &Organization{}
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&organization)

	if err != nil {
		return nil, err
	}

	return organization, nil
}

// List returns all the organizations for a user.
func (o *organizationsService) List(ctx context.Context) ([]*Organization, error) {
	req, err := o.client.newRequest(http.MethodGet, organizationsAPIPath, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for list organization")
	}

	res, err := o.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	orgResponse := &OrganizationsResponse{}
	err = json.NewDecoder(res.Body).Decode(&orgResponse)

	if err != nil {
		return nil, err
	}

	return orgResponse.Organizations, nil
}
