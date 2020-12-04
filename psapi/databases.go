package psapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

// CreateDatabaseRequest encapsulates the request for creating a new database.
type CreateDatabaseRequest struct {
	Name string `json:"name"`
}

// DatabaseService is an interface for communicating with the PlanetScale
// Databases API endpoint.
type DatabasesService interface {
	Create(context.Context, *CreateDatabaseRequest) (*Database, error)
	List(context.Context) ([]*Database, error)
}

// Database represents a PlanetScale Database
type Database struct {
	Name string `json:"name"`
}

type databasesService struct {
	client *Client
}

var _ DatabasesService = &databasesService{}

func NewDatabasesService(client *Client) *databasesService {
	return &databasesService{
		client: client,
	}
}

type ListDatabasesResponse struct {
	Databases []*Database `json:"databases"`
}

func (ds *databasesService) List(ctx context.Context) ([]*Database, error) {
	apiEndpoint := ds.client.GetAPIEndpoint("databases")

	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	res, err := ds.client.Do(ctx, req, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error communicating with API")
	}
	defer res.Body.Close()

	listRes := &ListDatabasesResponse{}
	err = json.NewDecoder(res.Body).Decode(listRes)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding list databases response")
	}

	return listRes.Databases, nil
}

func (ds *databasesService) Create(ctx context.Context, req *CreateDatabaseRequest) (*Database, error) {
	return nil, fmt.Errorf("unimplemented")
}
