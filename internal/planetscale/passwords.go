package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

type DatabaseBranchPassword struct {
	PublicID  string         `json:"id"`
	Name      string         `json:"name"`
	Hostname  string         `json:"access_host_url"`
	Username  string         `json:"username"`
	Role      string         `json:"role"`
	Actor     *Actor         `json:"actor"`
	Branch    DatabaseBranch `json:"database_branch"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt time.Time      `json:"deleted_at"`
	ExpiresAt time.Time      `json:"expires_at"`
	PlainText string         `json:"plain_text"`
	TTL       int            `json:"ttl_seconds"`
	Renewable bool           `json:"renewable"`
	Replica   bool           `json:"replica"`
}

// DatabaseBranchPasswordRequest encapsulates the request for creating/getting/deleting a
// database branch password.
type DatabaseBranchPasswordRequest struct {
	Organization     string `json:"-"`
	Database         string `json:"-"`
	Branch           string `json:"-"`
	Role             string `json:"role,omitempty"`
	Name             string `json:"name"`
	TTL              int    `json:"ttl,omitempty"`
	Replica          bool   `json:"replica,omitempty"`
	ReadOnlyRegionID string `json:"read_only_region_id,omitempty"`
}

// ListDatabaseBranchPasswordRequest encapsulates the request for listing all passwords
// for a given database branch.
type ListDatabaseBranchPasswordRequest struct {
	Organization string
	Database     string
	Branch       string
}

// GetDatabaseBranchPasswordRequest encapsulates the request for listing all passwords
// for a given database branch.
type GetDatabaseBranchPasswordRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Name         string `json:"name"`
	PasswordId   string
}

// DeleteDatabaseBranchPasswordRequest encapsulates the request for deleting a password
// for a given database branch.
type DeleteDatabaseBranchPasswordRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Name         string `json:"name"`
	PasswordId   string
}

// RenewDatabaseBranchPasswordRequest encapsulates the request for renewing a password
// for a given database branch.
type RenewDatabaseBranchPasswordRequest struct {
	Organization string
	Database     string
	Branch       string
	PasswordId   string
}

// DatabaseBranchPasswordsService is an interface for communicating with the PlanetScale
// Database Branch Passwords API endpoint.
type PasswordsService interface {
	Create(context.Context, *DatabaseBranchPasswordRequest) (*DatabaseBranchPassword, error)
	// List returns passwords with optional pagination support via ListOption parameters
	List(context.Context, *ListDatabaseBranchPasswordRequest, ...ListOption) ([]*DatabaseBranchPassword, error)
	Get(context.Context, *GetDatabaseBranchPasswordRequest) (*DatabaseBranchPassword, error)
	Delete(context.Context, *DeleteDatabaseBranchPasswordRequest) error
	Renew(context.Context, *RenewDatabaseBranchPasswordRequest) (*DatabaseBranchPassword, error)
}

type passwordsService struct {
	client *Client
}

type passwordsResponse struct {
	Passwords []*DatabaseBranchPassword `json:"data"`
}

var _ PasswordsService = &passwordsService{}

func NewPasswordsService(client *Client) *passwordsService {
	return &passwordsService{
		client: client,
	}
}

// Creates a new password for a branch.
func (d *passwordsService) Create(ctx context.Context, createReq *DatabaseBranchPasswordRequest) (*DatabaseBranchPassword, error) {
	pathStr := passwordsBranchAPIPath(createReq.Organization, createReq.Database, createReq.Branch)
	req, err := d.client.newRequest(http.MethodPost, pathStr, createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	password := &DatabaseBranchPassword{}
	if err := d.client.do(ctx, req, &password); err != nil {
		return nil, err
	}

	return password, nil
}

// Delete an existing password for a branch.
func (d *passwordsService) Delete(ctx context.Context, deleteReq *DeleteDatabaseBranchPasswordRequest) error {
	pathStr := passwordBranchAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.Branch, deleteReq.PasswordId)
	req, err := d.client.newRequest(http.MethodDelete, pathStr, nil)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	err = d.client.do(ctx, req, nil)
	return err
}

// Get an existing password for a branch.
func (d *passwordsService) Get(ctx context.Context, getReq *GetDatabaseBranchPasswordRequest) (*DatabaseBranchPassword, error) {
	pathStr := passwordBranchAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.PasswordId)
	req, err := d.client.newRequest(http.MethodGet, pathStr, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	password := &DatabaseBranchPassword{}
	if err := d.client.do(ctx, req, &password); err != nil {
		return nil, err
	}

	return password, nil
}

// List all existing passwords. If req.Branch is set, all passwords for that
// branch will be listed.
func (d *passwordsService) List(ctx context.Context, listReq *ListDatabaseBranchPasswordRequest, opts ...ListOption) ([]*DatabaseBranchPassword, error) {
	pathStr := passwordsAPIPath(listReq.Organization, listReq.Database)
	if listReq.Branch != "" {
		pathStr = passwordBranchAPIPath(listReq.Organization, listReq.Database, listReq.Branch, "")
	}

	defaultOpts := defaultListOptions(WithPerPage(50))
	for _, opt := range opts {
		err := opt(defaultOpts)
		if err != nil {
			return nil, err
		}
	}

	req, err := d.client.newRequest(http.MethodGet, pathStr, nil, WithQueryParams(*defaultOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating http request to list passwords: %w", err)
	}

	passwordsResp := &passwordsResponse{}
	if err := d.client.do(ctx, req, &passwordsResp); err != nil {
		return nil, err
	}

	return passwordsResp.Passwords, nil
}

func (d *passwordsService) Renew(ctx context.Context, renewReq *RenewDatabaseBranchPasswordRequest) (*DatabaseBranchPassword, error) {
	pathStr := passwordRenewAPIPath(renewReq.Organization, renewReq.Database, renewReq.Branch, renewReq.PasswordId)
	req, err := d.client.newRequest(http.MethodPost, pathStr, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	password := &DatabaseBranchPassword{}
	if err := d.client.do(ctx, req, &password); err != nil {
		return nil, err
	}
	return password, nil
}

func passwordBranchAPIPath(org, db, branch, password string) string {
	return path.Join(passwordsBranchAPIPath(org, db, branch), password)
}

func passwordsBranchAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "passwords")
}

func passwordsAPIPath(org, db string) string {
	return path.Join(databasesAPIPath(org), db, "passwords")
}

func passwordRenewAPIPath(org, db, branch, password string) string {
	return path.Join(passwordBranchAPIPath(org, db, branch, password), "renew")
}
