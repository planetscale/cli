package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// CreateDatabaseRequest encapsulates the request for creating a new database.
type CreateDatabaseRequest struct {
	Organization string
	Database     *Database `json:"database"`
}

// DatabaseRequest encapsulates the request for getting a single database.
type GetDatabaseRequest struct {
	Organization string
	Database     string
}

// ListDatabasesRequest encapsulates the request for listing all databases in an
// organization.
type ListDatabasesRequest struct {
	Organization string
}

// DeleteDatabaseRequest encapsulates the request for deleting a database from
// an organization.
type DeleteDatabaseRequest struct {
	Organization string
	Database     string
}

// DatabaseService is an interface for communicating with the PlanetScale
// Databases API endpoint.
type DatabasesService interface {
	Create(context.Context, *CreateDatabaseRequest) (*Database, error)
	Get(context.Context, *GetDatabaseRequest) (*Database, error)
	List(context.Context, *ListDatabasesRequest) ([]*Database, error)
	Delete(context.Context, *DeleteDatabaseRequest) error
}

// Database represents a PlanetScale database
type Database struct {
	Name      string    `json:"name"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Database represents a list of PlanetScale databases
type databasesResponse struct {
	Databases []*Database `json:"data"`
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

func (ds *databasesService) List(ctx context.Context, listReq *ListDatabasesRequest) ([]*Database, error) {
	req, err := ds.client.newRequest(http.MethodGet, databasesAPIPath(listReq.Organization), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	dbResponse := databasesResponse{}
	err = json.NewDecoder(res.Body).Decode(&dbResponse)

	if err != nil {
		return nil, err
	}

	return dbResponse.Databases, nil
}

func (ds *databasesService) Create(ctx context.Context, createReq *CreateDatabaseRequest) (*Database, error) {
	req, err := ds.client.newRequest(http.MethodPost, databasesAPIPath(createReq.Organization), createReq)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for create database")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	db := &Database{}
	err = json.NewDecoder(res.Body).Decode(db)

	if err != nil {
		return nil, err
	}

	return db, nil
}

func (ds *databasesService) Get(ctx context.Context, getReq *GetDatabaseRequest) (*Database, error) {
	path := fmt.Sprintf("%s/%s", databasesAPIPath(getReq.Organization), getReq.Database)
	req, err := ds.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for get database")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	db := &Database{}
	err = json.NewDecoder(res.Body).Decode(&db)

	if err != nil {
		return nil, err
	}

	return db, nil
}

func (ds *databasesService) Delete(ctx context.Context, deleteReq *DeleteDatabaseRequest) error {
	path := fmt.Sprintf("%s/%s", databasesAPIPath(deleteReq.Organization), deleteReq.Database)
	req, err := ds.client.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return errors.Wrap(err, "error creating request for delete database")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

func databasesAPIPath(org string) string {
	return fmt.Sprintf("v1/organizations/%s/databases", org)
}
