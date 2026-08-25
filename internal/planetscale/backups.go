package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

type Backup struct {
	PublicID    string    `json:"id"`
	Name        string    `json:"name"`
	State       string    `json:"state"`
	Size        int64     `json:"size"`
	Actor       *Actor    `json:"actor"`
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

	Name           string `json:"name,omitempty"`
	RetentionUnit  string `json:"retention_unit,omitempty"`
	RetentionValue int    `json:"retention_value,omitempty"`
	Emergency      bool   `json:"emergency,omitempty"`
}

type ListBackupsRequest struct {
	Organization string
	Database     string
	Branch       string
	To           string
	State        string
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
	req, err := d.client.newRequest(http.MethodPost, path, createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	backup := &Backup{}
	if err := d.client.do(ctx, req, &backup); err != nil {
		return nil, err
	}

	return backup, nil
}

// Returns a single backup for a branch.
func (d *backupsService) Get(ctx context.Context, getReq *GetBackupRequest) (*Backup, error) {
	path := backupAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.Backup)
	req, err := d.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	backup := &Backup{}
	if err := d.client.do(ctx, req, &backup); err != nil {
		return nil, err
	}

	return backup, nil
}

// Returns all of the backups for a branch.
func (d *backupsService) List(ctx context.Context, listReq *ListBackupsRequest) ([]*Backup, error) {
	query := url.Values{}
	if listReq.To != "" {
		query.Set("to", listReq.To)
	}
	if listReq.State != "" {
		query.Set("state", listReq.State)
	}

	var opts []RequestOption
	if len(query) > 0 {
		opts = append(opts, WithQueryParams(query))
	}

	req, err := d.client.newRequest(http.MethodGet, backupsAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	backups := &backupsResponse{}
	if err := d.client.do(ctx, req, &backups); err != nil {
		return nil, err
	}

	return backups.Backups, nil
}

// Deletes a branch backup.
func (d *backupsService) Delete(ctx context.Context, deleteReq *DeleteBackupRequest) error {
	path := backupAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.Branch, deleteReq.Backup)
	req, err := d.client.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	err = d.client.do(ctx, req, nil)
	return err
}

func backupsAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "backups")
}

func backupAPIPath(org, db, branch, backup string) string {
	return path.Join(backupsAPIPath(org, db, branch), backup)
}
