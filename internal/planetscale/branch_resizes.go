package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// BranchResizeRequest represents a Vitess branch VTGate resize request.
type BranchResizeRequest struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	State string `json:"state"`
	Actor *Actor `json:"actor"`

	VTGateSize                         string `json:"vtgate_size"`
	PreviousVTGateSize                 string `json:"previous_vtgate_size"`
	VTGateName                         string `json:"vtgate_name"`
	VTGateDisplayName                  string `json:"vtgate_display_name"`
	PreviousVTGateName                 string `json:"previous_vtgate_name"`
	PreviousVTGateDisplayName          string `json:"previous_vtgate_display_name"`
	VTGateCount                        int    `json:"vtgate_count"`
	PreviousVTGateCount                int    `json:"previous_vtgate_count"`
	VTGateMaxCount                     *int   `json:"vtgate_max_count"`
	PreviousVTGateMaxCount             *int   `json:"previous_vtgate_max_count"`
	VTGateAutoscaling                  bool   `json:"vtgate_autoscaling"`
	PreviousVTGateAutoscaling          bool   `json:"previous_vtgate_autoscaling"`
	VTGateTargetCPUUtilization         *int   `json:"vtgate_target_cpu_utilization"`
	PreviousVTGateTargetCPUUtilization *int   `json:"previous_vtgate_target_cpu_utilization"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// ResizeBranchRequest encapsulates a request to resize a branch's VTGates.
type ResizeBranchRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`

	VTGateSize                 string `json:"vtgate_size,omitempty"`
	VTGateCount                *int   `json:"vtgate_count,omitempty"`
	VTGateMaxCount             *int   `json:"vtgate_max_count,omitempty"`
	VTGateAutoscaling          *bool  `json:"vtgate_autoscaling,omitempty"`
	VTGateTargetCPUUtilization *int   `json:"vtgate_target_cpu_utilization,omitempty"`
}

// ListBranchResizesRequest encapsulates a request to list branch VTGate resize requests.
type ListBranchResizesRequest struct {
	Organization string
	Database     string
	Branch       string
}

// CancelBranchResizeRequest encapsulates a request to cancel a queued branch VTGate resize.
type CancelBranchResizeRequest struct {
	Organization string
	Database     string
	Branch       string
}

// BranchResizeStatusRequest encapsulates a request for the latest branch VTGate resize status.
type BranchResizeStatusRequest struct {
	Organization string
	Database     string
	Branch       string
}

type branchResizesResponse struct {
	Resizes []*BranchResizeRequest `json:"data"`
}

func branchResizesAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "resizes")
}

// Resize queues or updates a VTGate resize for a Vitess branch.
func (d *databaseBranchesService) Resize(ctx context.Context, resizeReq *ResizeBranchRequest) (*BranchResizeRequest, error) {
	req, err := d.client.newRequest(http.MethodPut, branchResizesAPIPath(resizeReq.Organization, resizeReq.Database, resizeReq.Branch), resizeReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resize := &BranchResizeRequest{}
	if err := d.client.do(ctx, req, resize); err != nil {
		return nil, err
	}

	return resize, nil
}

// ListResizes returns VTGate resize requests for a Vitess branch.
func (d *databaseBranchesService) ListResizes(ctx context.Context, listReq *ListBranchResizesRequest) ([]*BranchResizeRequest, error) {
	req, err := d.client.newRequest(http.MethodGet, branchResizesAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resizes := &branchResizesResponse{}
	if err := d.client.do(ctx, req, resizes); err != nil {
		return nil, err
	}

	return resizes.Resizes, nil
}

// CancelResize cancels a queued VTGate resize for a Vitess branch.
func (d *databaseBranchesService) CancelResize(ctx context.Context, cancelReq *CancelBranchResizeRequest) error {
	req, err := d.client.newRequest(http.MethodDelete, branchResizesAPIPath(cancelReq.Organization, cancelReq.Database, cancelReq.Branch), nil)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	return d.client.do(ctx, req, nil)
}

// ResizeStatus returns the most recent VTGate resize request for a Vitess branch.
func (d *databaseBranchesService) ResizeStatus(ctx context.Context, statusReq *BranchResizeStatusRequest) (*BranchResizeRequest, error) {
	resizes, err := d.ListResizes(ctx, &ListBranchResizesRequest{
		Organization: statusReq.Organization,
		Database:     statusReq.Database,
		Branch:       statusReq.Branch,
	})
	if err != nil {
		return nil, err
	}

	if len(resizes) == 0 {
		return nil, &Error{
			msg:  "Not Found",
			Code: ErrNotFound,
		}
	}

	return resizes[0], nil
}
