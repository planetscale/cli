package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const deployRequestsAPIPath = "v1/deploy-requests"

// PerformDeployRequest is a request for approving and deploying a deploy request.
// NOTE: We deviate from naming convention here because we have a data model
// named DeployRequest already.
type PerformDeployRequest struct {
	ID string `json:"-"`
}

// GetDeployRequest encapsulates the request for getting a single deploy
// request.
type GetDeployRequestRequest struct {
	ID string `json:"-"`
}

// DeployRequest encapsulates a requested deploy of a schema snapshot.
type DeployRequest struct {
	ID string `json:"id"`

	Branch     string `json:"branch"`
	IntoBranch string `json:"into_branch"`

	Notes string `json:"notes"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`
}

// DeployRequestsService is an interface for communicating with the PlanetScale
// deploy requests API.
type DeployRequestsService interface {
	Get(context.Context, *GetDeployRequestRequest) (*DeployRequest, error)
	Deploy(context.Context, *PerformDeployRequest) (*DeployRequest, error)
}

type deployRequestsService struct {
	client *Client
}

var _ DeployRequestsService = &deployRequestsService{}

func NewDeployRequestsService(client *Client) *deployRequestsService {
	return &deployRequestsService{
		client: client,
	}
}

// Get fetches a single deploy request.
func (d *deployRequestsService) Get(ctx context.Context, getReq *GetDeployRequestRequest) (*DeployRequest, error) {
	req, err := d.client.newRequest(http.MethodGet, deployRequestAPIPath(getReq.ID), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	res, err := d.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	dr := &DeployRequest{}
	err = json.NewDecoder(res.Body).Decode(dr)
	if err != nil {
		return nil, err
	}

	return dr, nil
}

// Deploy approves and executes a specific deploy request.
func (d *deployRequestsService) Deploy(ctx context.Context, deployReq *PerformDeployRequest) (*DeployRequest, error) {
	path := fmt.Sprintf("%s/deploy", deployRequestAPIPath(deployReq.ID))
	req, err := d.client.newRequest(http.MethodPost, path, deployReq)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	res, err := d.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	dr := &DeployRequest{}
	err = json.NewDecoder(res.Body).Decode(dr)
	if err != nil {
		return nil, err
	}

	return dr, nil
}

// deployRequestAPIPath gets the base path for accessing a single deploy request
func deployRequestAPIPath(id string) string {
	return fmt.Sprintf("%s/%s", deployRequestsAPIPath, id)
}
