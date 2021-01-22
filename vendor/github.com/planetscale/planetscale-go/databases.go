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
	Name      string    `jsonapi:"attr,name" json:"name"`
	Notes     string    `jsonapi:"attr,notes" json:"notes"`
	CreatedAt time.Time `jsonapi:"attr,created_at,iso8601" json:"created_at"`
	UpdatedAt time.Time `jsonapi:"attr,updated_at,iso8601" json:"updated_at"`
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

	databases, err := jsonapi.UnmarshalManyPayload(res.Body, reflect.TypeOf(new(Database)))
	if err != nil {
		return nil, err
	}

	dbs := make([]*Database, 0)
	for _, database := range databases {
		db, ok := database.(*Database)
		if ok {
			dbs = append(dbs, db)
		}
	}

	return dbs, nil
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
	err = jsonapi.UnmarshalPayload(res.Body, db)
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
	err = jsonapi.UnmarshalPayload(res.Body, db)
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
