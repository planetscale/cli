package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

type Backup struct {
	Name        string    `json:"name"`
	State       string    `json:"state"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	StartedAt   time.Time `json:"started_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	CompletedAt time.Time `json:"completed_at"`
}

type backupsResponse struct {
	Backups []*Backup `json:"data"`
}

type CreateBackupRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

type ListBackupsRequest struct {
	Organization string
	Database     string
	Branch       string
}

type GetBackupRequest struct {
	Organization string
	Database     string
	Branch       string
	Backup       string
}

type DeleteBackupRequest struct {
	Organization string
	Database     string
	Branch       string
	Backup       string
}

// BackupsService is an interface for communicating with the PlanetScale
// backup API endpoint.
type BackupsService interface {
	Create(context.Context, *CreateBackupRequest) (*Backup, error)
	List(context.Context, *ListBackupsRequest) ([]*Backup, error)
	Get(context.Context, *GetBackupRequest) (*Backup, error)
	Delete(context.Context, *DeleteBackupRequest) error
}

type backupsService struct {
	client *Client
}

var _ BackupsService = &backupsService{}

func NewBackupsService(client *Client) *backupsService {
	return &backupsService{
		client: client,
	}
}

// Creates a new backup for a branch.
func (d *backupsService) Create(ctx context.Context, createReq *CreateBackupRequest) (*Backup, error) {
	path := backupsAPIPath(createReq.Organization, createReq.Database, createReq.Branch)
	req, err := d.client.newRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}
	res, err := d.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	backup := &Backup{}
	err = json.NewDecoder(res.Body).Decode(&backup)

	if err != nil {
		return nil, err
	}

	return backup, nil
}

// Returns a single backup for a branch.
func (d *backupsService) Get(ctx context.Context, getReq *GetBackupRequest) (*Backup, error) {
	path := backupAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.Backup)
	req, err := d.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	res, err := d.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	backup := &Backup{}
	err = json.NewDecoder(res.Body).Decode(&backup)

	if err != nil {
		return nil, err
	}

	return backup, nil
}

// Returns all of the backups for a branch.
func (d *backupsService) List(ctx context.Context, listReq *ListBackupsRequest) ([]*Backup, error) {
	req, err := d.client.newRequest(http.MethodGet, backupsAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}

	res, err := d.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	backups := &backupsResponse{}
	err = json.NewDecoder(res.Body).Decode(&backups)

	if err != nil {
		return nil, err
	}

	return backups.Backups, nil
}

// Deletes a branch backup.
func (d *backupsService) Delete(ctx context.Context, deleteReq *DeleteBackupRequest) error {
	path := backupAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.Branch, deleteReq.Backup)
	req, err := d.client.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return errors.Wrap(err, "error creating http request")
	}

	res, err := d.client.Do(ctx, req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

func backupsAPIPath(org, db, branch string) string {
	return fmt.Sprintf("%s/backups", databaseBranchAPIPath(org, db, branch))
}

func backupAPIPath(org, db, branch, backup string) string {
	return fmt.Sprintf("%s/%s", backupsAPIPath(org, db, branch), backup)
}
