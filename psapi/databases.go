package psapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

var databasesAPIPath = "databases"

// CreateDatabaseRequest encapsulates the request for creating a new database.
type CreateDatabaseRequest struct {
	Database *Database `json:"demo_api"`
}

// DatabaseService is an interface for communicating with the PlanetScale
// Databases API endpoint.
type DatabasesService interface {
	Create(context.Context, *CreateDatabaseRequest) (*Database, error)
	List(context.Context) ([]*Database, error)
	Delete(context.Context, int64) (bool, error)
}

// Database represents a PlanetScale Database
type Database struct {
	ID   int64  `json:"id,omitempty" header:"id"`
	Name string `json:"name" header:"name"`
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
	req, err := ds.client.NewRequest(http.MethodGet, databasesAPIPath, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	listRes := &ListDatabasesResponse{}
	_, err = ds.client.Do(ctx, req, listRes)
	if err != nil {
		return nil, errors.Wrap(err, "error communicating with API")
	}

	return listRes.Databases, nil
}

// CreateDatabaseResponse encapsulates the JSON returned after successfully
// creating a database.
type CreateDatabaseResponse struct {
	Database *Database `json:"database"`
	ID       int64     `json:"id"`
	Name     string    `json:"name"`
}

func (ds *databasesService) Create(ctx context.Context, createReq *CreateDatabaseRequest) (*Database, error) {
	req, err := ds.client.NewRequest(http.MethodPost, databasesAPIPath, createReq)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for create database")
	}

	createRes := &CreateDatabaseResponse{}
	_, err = ds.client.Do(ctx, req, createRes)
	if err != nil {
		return nil, errors.Wrap(err, "error communicating with API")
	}

	return createRes.Database, nil
}

func (ds *databasesService) Delete(ctx context.Context, id int64) (bool, error) {
	path := fmt.Sprintf("%s/%d", databasesAPIPath, id)
	req, err := ds.client.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return false, errors.Wrap(err, "error creating request for delete database")
	}

	res, err := ds.client.Do(ctx, req, nil)
	if err != nil {
		return false, errors.Wrap(err, "error deleting database")
	}

	if res.StatusCode == http.StatusNotFound {
		return false, errors.New("database not found")
	}

	return true, nil
}

func (ds *databasesService) getListDatabasesEndpoint() string {
	return ds.client.GetAPIEndpoint("databases")

}
