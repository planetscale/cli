package psapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

var databasesAPIPath = "databases"

// CreateDatabaseRequest encapsulates the request for creating a new database.
type CreateDatabaseRequest struct {
	Database *Database `json:"database"`
}

// DatabaseService is an interface for communicating with the PlanetScale
// Databases API endpoint.
type DatabasesService interface {
	Create(context.Context, *CreateDatabaseRequest) (*Database, error)
	Get(context.Context, int64) (*Database, error)
	Status(context.Context, int64) (*DatabaseStatus, error)
	List(context.Context) ([]*Database, error)
	Delete(context.Context, int64) (bool, error)
}

// Database represents a PlanetScale database
type Database struct {
	ID        int64      `json:"id,omitempty" header:"id"`
	Name      string     `json:"name" header:"name"`
	Notes     string     `json:"notes" header:"notes"`
	CreatedAt *time.Time `json:"created_at" header:"created_at,unixtime_human"`
	UpdatedAt *time.Time `json:"updated_at" header:"updated_at,unixtime_human"`
}

// DatabaseStatus represents the status of a PlanetScale database.
type DatabaseStatus struct {
	DatabaseID    int64  `json:"database_id" header:"database_id"`
	DeployPhase   string `json:"deploy_phase" header:"status"`
	GatewayHost   string `json:"mysql_gateway_host" header:"gateway_host"`
	GatewayPort   int    `json:"mysql_gateway_port" header:"gateway_port"`
	MySQLUser     string `json:"mysql_gateway_user" header:"user"`
	MySQLPassword string `json:"mysql_gateway_pass" header:"password"`
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
		return nil, err
	}

	return listRes.Databases, nil
}

// DatabaseResponse encapsulates the JSON returned after successfully creating
// or fetching a database.
type DatabaseResponse struct {
	Database *Database `json:"database"`
}

func (ds *databasesService) Create(ctx context.Context, createReq *CreateDatabaseRequest) (*Database, error) {
	req, err := ds.client.NewRequest(http.MethodPost, databasesAPIPath, createReq)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for create database")
	}

	createRes := &DatabaseResponse{}
	_, err = ds.client.Do(ctx, req, createRes)
	if err != nil {
		return nil, err
	}

	return createRes.Database, nil
}

func (ds *databasesService) Get(ctx context.Context, id int64) (*Database, error) {
	path := fmt.Sprintf("%s/%d", databasesAPIPath, id)
	req, err := ds.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for get database")
	}

	_, err = ds.client.Do(ctx, req, nil)
	if err != nil {
		return nil, err
	}

	dbRes := &DatabaseResponse{}
	_, err = ds.client.Do(ctx, req, dbRes)
	if err != nil {
		return nil, err
	}

	return dbRes.Database, nil
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

// StatusResponse returns a response for the status of a database
type StatusResponse struct {
	Status *DatabaseStatus `json:"status"`
}

func (ds *databasesService) Status(ctx context.Context, id int64) (*DatabaseStatus, error) {
	path := fmt.Sprintf("%s/%d/status", databasesAPIPath, id)
	req, err := ds.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request for database status")
	}

	status := &StatusResponse{}
	_, err = ds.client.Do(ctx, req, status)
	if err != nil {
		return nil, errors.Wrap(err, "error getting database status")
	}

	return status.Status, nil
}
