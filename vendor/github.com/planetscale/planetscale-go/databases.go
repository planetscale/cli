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
	Status(context.Context, string, string) (*DatabaseStatus, error)
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

// DatabaseStatus represents the status of a PlanetScale database.
type DatabaseStatus struct {
	DatabaseID    int64  `json:"database_id" jsonapi:"database_id"`
	DeployPhase   string `json:"deploy_phase" jsonapi:"deploy_phase"`
	GatewayHost   string `json:"mysql_gateway_host" jsonapi:"gateway_host"`
	GatewayPort   int    `json:"mysql_gateway_port" jsonapi:"gateway_port"`
	MySQLUser     string `json:"mysql_gateway_user" jsonapi:"my_sql_user"`
	MySQLPassword string `json:"mysql_gateway_pass" jsonapi:"my_sql_password"`
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

	dbs := make([]*Database, 0, len(databases))
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

	createRes := &Database{}
	err = jsonapi.UnmarshalPayload(res.Body, createRes)
	if err != nil {
		return nil, err
	}

	return createRes, nil
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

// StatusResponse returns a response for the status of a database
type StatusResponse struct {
	Status *DatabaseStatus `json:"status"`
}

func databasesAPIPath(org string) string {
	return fmt.Sprintf("organizations/%s/databases", org)
}

func (ds *databasesService) Status(ctx context.Context, org string, name string) (*DatabaseStatus, error) {
	path := fmt.Sprintf("%s/%s/status", databasesAPIPath(org), name)
	req, err := ds.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for database status")
	}

	res, err := ds.client.Do(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "error getting database status")
	}
	defer res.Body.Close()

	status := &StatusResponse{}
	err = jsonapi.UnmarshalPayload(res.Body, status)
	if err != nil {
		return nil, err
	}

	return status.Status, nil
}
