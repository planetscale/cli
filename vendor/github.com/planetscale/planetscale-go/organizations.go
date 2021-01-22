package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/google/jsonapi"
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
	Name      string    `jsonapi:"attr,name" json:"name"`
	CreatedAt time.Time `jsonapi:"attr,created_at,iso8601" json:"created_at"`
	UpdatedAt time.Time `jsonapi:"attr,updated_at,iso8601" json:"updated_at"`
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
	err = jsonapi.UnmarshalPayload(res.Body, organization)
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

	organizations, err := jsonapi.UnmarshalManyPayload(res.Body, reflect.TypeOf(new(Organization)))
	if err != nil {
		return nil, err
	}

	orgs := make([]*Organization, 0)

	for _, organization := range organizations {
		org, ok := organization.(*Organization)
		if ok {
			orgs = append(orgs, org)
		}
	}

	return orgs, nil
}
