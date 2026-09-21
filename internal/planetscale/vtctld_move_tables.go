package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
)

// MoveTablesService is an interface for interacting with the MoveTables endpoints of the
// PlanetScale API.
type MoveTablesService interface {
	List(context.Context, *MoveTablesListRequest) (json.RawMessage, error)
	Create(context.Context, *MoveTablesCreateRequest) (*VtctldOperationReference, error)
	Show(context.Context, *MoveTablesShowRequest) (json.RawMessage, error)
	Status(context.Context, *MoveTablesStatusRequest) (json.RawMessage, error)
	Start(context.Context, *MoveTablesStartRequest) (json.RawMessage, error)
	Stop(context.Context, *MoveTablesStopRequest) (json.RawMessage, error)
	SwitchTraffic(context.Context, *MoveTablesSwitchTrafficRequest) (*VtctldOperationReference, error)
	ReverseTraffic(context.Context, *MoveTablesReverseTrafficRequest) (*VtctldOperationReference, error)
	Cancel(context.Context, *MoveTablesCancelRequest) (*VtctldOperationReference, error)
	Complete(context.Context, *MoveTablesCompleteRequest) (*VtctldOperationReference, error)
}

// MoveTablesListRequest is a request for listing MoveTables workflows.
type MoveTablesListRequest struct {
	Organization   string `json:"-"`
	Database       string `json:"-"`
	Branch         string `json:"-"`
	TargetKeyspace string `json:"-"`
}

// MoveTablesCreateRequest is a request for creating a MoveTables workflow.
type MoveTablesCreateRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`

	Workflow                     string   `json:"workflow"`
	TargetKeyspace               string   `json:"target_keyspace"`
	SourceKeyspace               string   `json:"source_keyspace"`
	Tables                       []string `json:"tables,omitempty"`
	AllTables                    *bool    `json:"all_tables,omitempty"`
	AutoStart                    *bool    `json:"auto_start,omitempty"`
	StopAfterCopy                *bool    `json:"stop_after_copy,omitempty"`
	DeferSecondaryKeys           *bool    `json:"defer_secondary_keys,omitempty"`
	OnDDL                        string   `json:"on_ddl,omitempty"`
	ShardedAutoIncrementHandling string   `json:"sharded_auto_increment_handling,omitempty"`
	GlobalKeyspace               string   `json:"global_keyspace,omitempty"`
	SourceTimeZone               string   `json:"source_time_zone,omitempty"`
	TenantID                     string   `json:"tenant_id,omitempty"`
	Cells                        []string `json:"cells,omitempty"`
	TabletTypes                  []string `json:"tablet_types,omitempty"`
	ExcludeTables                []string `json:"exclude_tables,omitempty"`
	AtomicCopy                   *bool    `json:"atomic_copy,omitempty"`
}

// MoveTablesShowRequest is a request for showing a MoveTables workflow.
type MoveTablesShowRequest struct {
	Organization   string `json:"-"`
	Database       string `json:"-"`
	Branch         string `json:"-"`
	Workflow       string `json:"-"`
	TargetKeyspace string `json:"-"`
}

// MoveTablesStatusRequest is a request for getting the status of a MoveTables workflow.
type MoveTablesStatusRequest struct {
	Organization   string `json:"-"`
	Database       string `json:"-"`
	Branch         string `json:"-"`
	Workflow       string `json:"-"`
	TargetKeyspace string `json:"-"`
}

// MoveTablesStartRequest is a request for starting a MoveTables workflow.
type MoveTablesStartRequest struct {
	Organization   string `json:"-"`
	Database       string `json:"-"`
	Branch         string `json:"-"`
	Workflow       string `json:"-"`
	TargetKeyspace string `json:"target_keyspace"`
}

// MoveTablesStopRequest is a request for stopping a MoveTables workflow.
type MoveTablesStopRequest struct {
	Organization   string `json:"-"`
	Database       string `json:"-"`
	Branch         string `json:"-"`
	Workflow       string `json:"-"`
	TargetKeyspace string `json:"target_keyspace"`
}

// MoveTablesSwitchTrafficRequest is a request for switching traffic for a MoveTables workflow.
type MoveTablesSwitchTrafficRequest struct {
	Organization              string   `json:"-"`
	Database                  string   `json:"-"`
	Branch                    string   `json:"-"`
	Workflow                  string   `json:"-"`
	TargetKeyspace            string   `json:"target_keyspace"`
	TabletTypes               []string `json:"tablet_types,omitempty"`
	MaxReplicationLagAllowed  *int64   `json:"max_replication_lag_allowed,omitempty"`
	DryRun                    *bool    `json:"dry_run,omitempty"`
	InitializeTargetSequences *bool    `json:"initialize_target_sequences,omitempty"`
}

// MoveTablesReverseTrafficRequest is a request for reversing traffic for a MoveTables workflow.
type MoveTablesReverseTrafficRequest struct {
	Organization             string   `json:"-"`
	Database                 string   `json:"-"`
	Branch                   string   `json:"-"`
	Workflow                 string   `json:"-"`
	TargetKeyspace           string   `json:"target_keyspace"`
	TabletTypes              []string `json:"tablet_types,omitempty"`
	MaxReplicationLagAllowed *int64   `json:"max_replication_lag_allowed,omitempty"`
	DryRun                   *bool    `json:"dry_run,omitempty"`
}

// MoveTablesCancelRequest is a request for canceling a MoveTables workflow.
type MoveTablesCancelRequest struct {
	Organization     string `json:"-"`
	Database         string `json:"-"`
	Branch           string `json:"-"`
	Workflow         string `json:"-"`
	TargetKeyspace   string `json:"target_keyspace"`
	KeepData         *bool  `json:"keep_data,omitempty"`
	KeepRoutingRules *bool  `json:"keep_routing_rules,omitempty"`
}

// MoveTablesCompleteRequest is a request for completing a MoveTables workflow.
type MoveTablesCompleteRequest struct {
	Organization     string `json:"-"`
	Database         string `json:"-"`
	Branch           string `json:"-"`
	Workflow         string `json:"-"`
	TargetKeyspace   string `json:"target_keyspace"`
	KeepData         *bool  `json:"keep_data,omitempty"`
	KeepRoutingRules *bool  `json:"keep_routing_rules,omitempty"`
	RenameTables     *bool  `json:"rename_tables,omitempty"`
	DryRun           *bool  `json:"dry_run,omitempty"`
}

type moveTablesService struct {
	client *Client
}

var _ MoveTablesService = &moveTablesService{}

func moveTablesWorkflowsAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "move-tables", "workflows")
}

func moveTablesWorkflowAPIPath(org, db, branch, workflow string) string {
	return path.Join(moveTablesWorkflowsAPIPath(org, db, branch), workflow)
}

func (s *moveTablesService) List(ctx context.Context, req *MoveTablesListRequest) (json.RawMessage, error) {
	p := moveTablesWorkflowsAPIPath(req.Organization, req.Database, req.Branch)
	v := url.Values{}
	if req.TargetKeyspace != "" {
		v.Set("target_keyspace", req.TargetKeyspace)
	}
	httpReq, err := s.client.newRequest(http.MethodGet, p, nil, WithQueryParams(v))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}
	var resp json.RawMessage
	if err := s.client.do(ctx, httpReq, &resp); err != nil {
		return nil, err
	}
	var envelope vtctldDataResponse
	if err := json.Unmarshal(resp, &envelope); err == nil && len(envelope.Data) > 0 {
		return envelope.Data, nil
	}
	return resp, nil
}

func (s *moveTablesService) Create(ctx context.Context, req *MoveTablesCreateRequest) (*VtctldOperationReference, error) {
	p := moveTablesWorkflowsAPIPath(req.Organization, req.Database, req.Branch)
	return s.enqueueOperation(ctx, p, req)
}

func (s *moveTablesService) Show(ctx context.Context, req *MoveTablesShowRequest) (json.RawMessage, error) {
	p := moveTablesWorkflowAPIPath(req.Organization, req.Database, req.Branch, req.Workflow)
	v := url.Values{}
	v.Set("target_keyspace", req.TargetKeyspace)
	httpReq, err := s.client.newRequest(http.MethodGet, p, nil, WithQueryParams(v))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}
	resp := &vtctldDataResponse{}
	if err := s.client.do(ctx, httpReq, resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (s *moveTablesService) Status(ctx context.Context, req *MoveTablesStatusRequest) (json.RawMessage, error) {
	p := path.Join(moveTablesWorkflowAPIPath(req.Organization, req.Database, req.Branch, req.Workflow), "status")
	v := url.Values{}
	v.Set("target_keyspace", req.TargetKeyspace)
	httpReq, err := s.client.newRequest(http.MethodGet, p, nil, WithQueryParams(v))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}
	resp := &vtctldDataResponse{}
	if err := s.client.do(ctx, httpReq, resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (s *moveTablesService) Start(ctx context.Context, req *MoveTablesStartRequest) (json.RawMessage, error) {
	p := path.Join(moveTablesWorkflowAPIPath(req.Organization, req.Database, req.Branch, req.Workflow), "start")
	httpReq, err := s.client.newRequest(http.MethodPost, p, req)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}
	resp := &vtctldDataResponse{}
	if err := s.client.do(ctx, httpReq, resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (s *moveTablesService) Stop(ctx context.Context, req *MoveTablesStopRequest) (json.RawMessage, error) {
	p := path.Join(moveTablesWorkflowAPIPath(req.Organization, req.Database, req.Branch, req.Workflow), "stop")
	httpReq, err := s.client.newRequest(http.MethodPost, p, req)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}
	resp := &vtctldDataResponse{}
	if err := s.client.do(ctx, httpReq, resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (s *moveTablesService) SwitchTraffic(ctx context.Context, req *MoveTablesSwitchTrafficRequest) (*VtctldOperationReference, error) {
	p := path.Join(moveTablesWorkflowAPIPath(req.Organization, req.Database, req.Branch, req.Workflow), "switch-traffic")
	return s.enqueueOperation(ctx, p, req)
}

func (s *moveTablesService) ReverseTraffic(ctx context.Context, req *MoveTablesReverseTrafficRequest) (*VtctldOperationReference, error) {
	p := path.Join(moveTablesWorkflowAPIPath(req.Organization, req.Database, req.Branch, req.Workflow), "reverse-traffic")
	return s.enqueueOperation(ctx, p, req)
}

func (s *moveTablesService) Cancel(ctx context.Context, req *MoveTablesCancelRequest) (*VtctldOperationReference, error) {
	p := path.Join(moveTablesWorkflowAPIPath(req.Organization, req.Database, req.Branch, req.Workflow), "cancel")
	return s.enqueueOperation(ctx, p, req)
}

func (s *moveTablesService) Complete(ctx context.Context, req *MoveTablesCompleteRequest) (*VtctldOperationReference, error) {
	p := path.Join(moveTablesWorkflowAPIPath(req.Organization, req.Database, req.Branch, req.Workflow), "complete")
	return s.enqueueOperation(ctx, p, req)
}

func (s *moveTablesService) enqueueOperation(ctx context.Context, p string, payload interface{}) (*VtctldOperationReference, error) {
	httpReq, err := s.client.newRequest(http.MethodPost, p, payload)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp := &VtctldOperationReference{}
	if err := s.client.do(ctx, httpReq, resp); err != nil {
		return nil, err
	}

	return resp, nil
}
