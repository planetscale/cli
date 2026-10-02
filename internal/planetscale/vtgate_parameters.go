package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
)

const vtgateChangeType = "vtgate"

// SubmitConfigChangesRequest applies draft config changes on a branch.
type SubmitConfigChangesRequest struct {
	Organization string   `json:"-"`
	Database     string   `json:"-"`
	Branch       string   `json:"-"`
	IDs          []string `json:"ids"`
}

type ListVTGateParametersRequest struct {
	Organization string
	Database     string
	Branch       string
}

// CreateVTGateConfigChangeRequest creates a draft change to a branch's VTGate
// parameters. A nil option value resets that parameter to its default.
type CreateVTGateConfigChangeRequest struct {
	Organization string             `json:"-"`
	Database     string             `json:"-"`
	Branch       string             `json:"-"`
	Options      map[string]*string `json:"options"`
}

type ListVTGateConfigChangesRequest struct {
	Organization string
	Database     string
	Branch       string
	Page         int
	PerPage      int
}

type GetVTGateConfigChangeRequest struct {
	Organization string
	Database     string
	Branch       string
	ID           string
}

type CancelVTGateConfigChangeRequest struct {
	Organization string
	Database     string
	Branch       string
	ID           string
}

type vtgateParametersResponse struct {
	VTGate []*VitessParameter `json:"vtgate"`
}

type branchConfigChangesResponse struct {
	ConfigChanges []*VitessConfigChange `json:"data"`
}

func (d *databaseBranchesService) SubmitConfigChanges(ctx context.Context, submitReq *SubmitConfigChangesRequest) error {
	req, err := d.client.newRequest(http.MethodPost, path.Join(branchConfigChangesAPIPath(submitReq.Organization, submitReq.Database, submitReq.Branch), "submit"), submitReq)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	return d.client.do(ctx, req, nil)
}

func (d *databaseBranchesService) ListVTGateParameters(ctx context.Context, listReq *ListVTGateParametersRequest) ([]*VitessParameter, error) {
	req, err := d.client.newRequest(http.MethodGet, path.Join(databaseBranchAPIPath(listReq.Organization, listReq.Database, listReq.Branch), "vitess-parameters"), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp := &vtgateParametersResponse{}
	if err := d.client.do(ctx, req, resp); err != nil {
		return nil, err
	}

	return resp.VTGate, nil
}

func (d *databaseBranchesService) CreateVTGateConfigChange(ctx context.Context, createReq *CreateVTGateConfigChangeRequest) (*VitessConfigChange, error) {
	body := struct {
		ChangeType string             `json:"change_type"`
		Options    map[string]*string `json:"options"`
	}{ChangeType: vtgateChangeType, Options: createReq.Options}

	req, err := d.client.newRequest(http.MethodPost, branchConfigChangesAPIPath(createReq.Organization, createReq.Database, createReq.Branch), body)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	change := &VitessConfigChange{}
	if err := d.client.do(ctx, req, change); err != nil {
		return nil, err
	}

	return change, nil
}

// ListVTGateConfigChanges returns the VTGate changes on a branch. The branch
// list also includes keyspace changes, so they are filtered out here.
func (d *databaseBranchesService) ListVTGateConfigChanges(ctx context.Context, listReq *ListVTGateConfigChangesRequest) ([]*VitessConfigChange, error) {
	values := defaultListOptions(WithPage(listReq.Page), WithPerPage(listReq.PerPage))
	values.URLValues.Set("change_type", vtgateChangeType)

	req, err := d.client.newRequest(http.MethodGet, branchConfigChangesAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil, WithQueryParams(*values.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp := &branchConfigChangesResponse{}
	if err := d.client.do(ctx, req, resp); err != nil {
		return nil, err
	}

	changes := make([]*VitessConfigChange, 0, len(resp.ConfigChanges))
	for _, change := range resp.ConfigChanges {
		if change.ChangeType == vtgateChangeType {
			changes = append(changes, change)
		}
	}
	return changes, nil
}

func (d *databaseBranchesService) GetVTGateConfigChange(ctx context.Context, getReq *GetVTGateConfigChangeRequest) (*VitessConfigChange, error) {
	req, err := d.client.newRequest(http.MethodGet, path.Join(branchConfigChangesAPIPath(getReq.Organization, getReq.Database, getReq.Branch), getReq.ID), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	change := &VitessConfigChange{}
	if err := d.client.do(ctx, req, change); err != nil {
		return nil, err
	}

	return change, nil
}

func (d *databaseBranchesService) CancelVTGateConfigChange(ctx context.Context, cancelReq *CancelVTGateConfigChangeRequest) error {
	req, err := d.client.newRequest(http.MethodDelete, path.Join(branchConfigChangesAPIPath(cancelReq.Organization, cancelReq.Database, cancelReq.Branch), cancelReq.ID), nil)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	return d.client.do(ctx, req, nil)
}

func branchConfigChangesAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "config-changes")
}
