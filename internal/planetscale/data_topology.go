package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"time"
)

type DataTopology struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	DataTopology json.RawMessage `json:"data_topology"`
	SyncedAt     *time.Time      `json:"synced_at"`
}

type BranchDataTopologyRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

type UpdateBranchDataTopologyRequest struct {
	Organization string          `json:"-"`
	Database     string          `json:"-"`
	Branch       string          `json:"-"`
	DataTopology json.RawMessage `json:"data_topology"`
}

func (d *databaseBranchesService) DataTopology(ctx context.Context, dataTopologyReq *BranchDataTopologyRequest) (*DataTopology, error) {
	p := path.Join(databaseBranchAPIPath(dataTopologyReq.Organization, dataTopologyReq.Database, dataTopologyReq.Branch), "data-topology")

	req, err := d.client.newRequest(http.MethodGet, p, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	dataTopology := &DataTopology{}
	if err := d.client.do(ctx, req, dataTopology); err != nil {
		return nil, err
	}

	return dataTopology, nil
}

func (d *databaseBranchesService) UpdateDataTopology(ctx context.Context, updateReq *UpdateBranchDataTopologyRequest) (*DataTopology, error) {
	p := path.Join(databaseBranchAPIPath(updateReq.Organization, updateReq.Database, updateReq.Branch), "data-topology")

	req, err := d.client.newRequest(http.MethodPut, p, updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	dataTopology := &DataTopology{}
	if err := d.client.do(ctx, req, dataTopology); err != nil {
		return nil, err
	}

	return dataTopology, nil
}
