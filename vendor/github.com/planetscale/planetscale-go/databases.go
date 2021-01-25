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
	Database *Database `json:"database"`
}

// DatabaseService is an interface for communicating with the PlanetScale
// Databases API endpoint.
type DatabasesService interface {
	Create(context.Context, string, *CreateDatabaseRequest) (*Database, error)
	Get(context.Context, string, string) (*Database, error)
	List(context.Context, string) ([]*Database, error)
	Delete(context.Context, string, string) (bool, error)
}

// Database represents a PlanetScale database
type Database struct {
	Name      string    `json:"name"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Database represents a list of PlanetScale databases
type Databases struct {
	Data []*Database `json:"data"`
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

func (ds *databasesService) List(ctx context.Context, org string) ([]*Database, error) {
	req, err := ds.client.newRequest(http.MethodGet, databasesAPIPath(org), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	databases := Databases{}
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&databases)

	if err != nil {
		return nil, err
	}

	return databases.Data, nil
}

func (ds *databasesService) Create(ctx context.Context, org string, createReq *CreateDatabaseRequest) (*Database, error) {
	req, err := ds.client.newRequest(http.MethodPost, databasesAPIPath(org), createReq)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for create database")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	db := &Database{}
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&db)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (ds *databasesService) Get(ctx context.Context, org string, name string) (*Database, error) {
	path := fmt.Sprintf("%s/%s", databasesAPIPath(org), name)
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
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&db)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (ds *databasesService) Delete(ctx context.Context, org string, name string) (bool, error) {
	path := fmt.Sprintf("%s/%s", databasesAPIPath(org), name)
	req, err := ds.client.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return false, errors.Wrap(err, "error creating request for delete database")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return false, errors.Wrap(err, "error deleting database")
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return false, errors.New("database not found")
	}

	return true, nil
}

func databasesAPIPath(org string) string {
	return fmt.Sprintf("v1/organizations/%s/databases", org)
}
