package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

// VtctldService is an interface for interacting with the general vtctld endpoints of the
// PlanetScale API.
type VtctldService interface {
	ListWorkflows(context.Context, *VtctldListWorkflowsRequest) (json.RawMessage, error)
	ListKeyspaces(context.Context, *VtctldListKeyspacesRequest) (json.RawMessage, error)
	GetRoutingRules(context.Context, *VtctldGetRoutingRulesRequest) (json.RawMessage, error)
	GetShard(context.Context, *VtctldGetShardRequest) (json.RawMessage, error)
	SetShardTabletControl(context.Context, *VtctldSetShardTabletControlRequest) (json.RawMessage, error)
	RefreshStateByShard(context.Context, *VtctldRefreshStateByShardRequest) (json.RawMessage, error)
	ListTablets(context.Context, *ListBranchTabletsRequest) ([]*TabletGroup, error)
	StartWorkflow(context.Context, *VtctldStartWorkflowRequest) (json.RawMessage, error)
	StopWorkflow(context.Context, *VtctldStopWorkflowRequest) (json.RawMessage, error)
	GetThrottlerStatus(context.Context, *VtctldGetThrottlerStatusRequest) (json.RawMessage, error)
	CheckThrottler(context.Context, *VtctldCheckThrottlerRequest) (json.RawMessage, error)
	UpdateThrottlerConfig(context.Context, *VtctldUpdateThrottlerConfigRequest) (json.RawMessage, error)
	GetOperation(context.Context, *GetVtctldOperationRequest) (*VtctldOperation, error)
}

type VtctldListWorkflowsRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Keyspace     string `json:"-"`
	Workflow     string `json:"-"`
	IncludeLogs  *bool  `json:"-"`
}

type VtctldListKeyspacesRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Name         string `json:"-"`
}

// VtctldGetRoutingRulesRequest is a request for reading live routing rules
// from the cluster via vtctld.
type VtctldGetRoutingRulesRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

// VtctldGetShardRequest is a request for reading a shard record from the
// cluster via vtctld.
type VtctldGetShardRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Keyspace     string `json:"-"`
	Shard        string `json:"-"`
}

// VtctldSetShardTabletControlRequest is a request for updating shard tablet
// controls via vtctld.
type VtctldSetShardTabletControlRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`

	Keyspace            string   `json:"keyspace"`
	Shard               string   `json:"shard"`
	TabletType          string   `json:"tablet_type"`
	Cells               []string `json:"cells,omitempty"`
	DeniedTables        []string `json:"denied_tables,omitempty"`
	Remove              *bool    `json:"remove,omitempty"`
	DisableQueryService *bool    `json:"disable_query_service,omitempty"`
}

// VtctldRefreshStateByShardRequest reloads tablet records for all tablets in
// a shard via vtctld.
type VtctldRefreshStateByShardRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`

	Keyspace string   `json:"keyspace"`
	Shard    string   `json:"shard"`
	Cells    []string `json:"cells,omitempty"`
}

// VtctldStartWorkflowRequest is a request for starting a workflow.
type VtctldStartWorkflowRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Workflow     string `json:"-"`
	Keyspace     string `json:"keyspace"`
}

// VtctldStopWorkflowRequest is a request for stopping a workflow.
type VtctldStopWorkflowRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Workflow     string `json:"-"`
	Keyspace     string `json:"keyspace"`
}

// VtctldGetThrottlerStatusRequest is a request for reading the tablet throttler
// status from a single tablet. The tablet is identified by its alias, as
// discovered via ListTablets.
type VtctldGetThrottlerStatusRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	TabletAlias  string `json:"-"`
}

// VtctldCheckThrottlerRequest is a request for issuing a throttler check against
// a single tablet. The tablet is identified by its alias, as discovered via
// ListTablets.
type VtctldCheckThrottlerRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`

	TabletAlias string `json:"tablet_alias"`
	// AppName is the app to issue the check on behalf of (e.g. "online-ddl"). If
	// empty, the throttler's default app is used.
	AppName string `json:"app_name,omitempty"`
	// Scope is the scope of the check, either "shard" or "self". If empty, the
	// throttler's default scope is used.
	Scope string `json:"scope,omitempty"`
	// SkipRequestHeartbeats instructs the throttler not to renew its heartbeat
	// lease while serving this check.
	SkipRequestHeartbeats *bool `json:"skip_request_heartbeats,omitempty"`
	// OkIfNotExists instructs the throttler to return OK even if the requested
	// metric does not exist.
	OkIfNotExists *bool `json:"ok_if_not_exists,omitempty"`
}

// VtctldThrottledAppConfig configures a single throttled app rule for
// UpdateThrottlerConfig.
type VtctldThrottledAppConfig struct {
	// Name is the app to throttle (e.g. "online-ddl", "vreplication", or an
	// Online DDL migration UUID).
	Name string `json:"name"`
	// Ratio is the fraction of operations to throttle for the app, 0.00-1.00 in
	// increments of 0.01.
	Ratio *float64 `json:"ratio,omitempty"`
	// ExpireAt is an optional RFC3339 expiration time for the app rule. It must
	// be in the future. Omit to throttle until changed.
	ExpireAt string `json:"expire_at,omitempty"`
}

// VtctldUpdateThrottlerConfigRequest is a request for updating the tablet
// throttler configuration for a keyspace.
type VtctldUpdateThrottlerConfigRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`

	Keyspace string `json:"keyspace"`
	// Enabled controls whether the tablet throttler is enabled for the keyspace.
	// Omit to leave the enable state unchanged.
	Enabled *bool `json:"enabled,omitempty"`
	// Threshold is the threshold for the default throttler check (replication
	// lag in seconds). It must be >= 0. Omit to leave the existing threshold
	// unchanged.
	Threshold *float64 `json:"threshold,omitempty"`
	// Apps configures zero or more per-app throttling rules.
	Apps []VtctldThrottledAppConfig `json:"apps,omitempty"`
	// UnthrottleApps names apps whose throttled-app rules should be removed.
	UnthrottleApps []string `json:"unthrottle_apps,omitempty"`
	// AppCheckedMetrics assigns metrics to check per app name
	// (e.g. {"rowstreamer": ["lag", "loadavg"]}).
	AppCheckedMetrics map[string][]string `json:"app_checked_metrics,omitempty"`
}

type vtctldService struct {
	client *Client
}

var _ VtctldService = &vtctldService{}

func vtctldWorkflowsAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "vtctld", "workflows")
}

func vtctldKeyspacesAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "vtctld", "keyspaces")
}

func vtctldRoutingRulesAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "vtctld", "routing-rules")
}

func vtctldShardAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "vtctld", "shard")
}

func vtctldShardTabletControlAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "vtctld", "shard", "tablet-control")
}

func vtctldShardRefreshStateAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "vtctld", "shard", "refresh-state")
}

func (s *vtctldService) ListWorkflows(ctx context.Context, req *VtctldListWorkflowsRequest) (json.RawMessage, error) {
	p := vtctldWorkflowsAPIPath(req.Organization, req.Database, req.Branch)
	v := url.Values{}
	v.Set("keyspace", req.Keyspace)
	if req.Workflow != "" {
		v.Set("workflow", req.Workflow)
	}
	if req.IncludeLogs != nil {
		v.Set("include_logs", strconv.FormatBool(*req.IncludeLogs))
	}
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

func (s *vtctldService) ListKeyspaces(ctx context.Context, req *VtctldListKeyspacesRequest) (json.RawMessage, error) {
	p := vtctldKeyspacesAPIPath(req.Organization, req.Database, req.Branch)
	v := url.Values{}
	if req.Name != "" {
		v.Set("name", req.Name)
	}
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

func (s *vtctldService) GetRoutingRules(ctx context.Context, req *VtctldGetRoutingRulesRequest) (json.RawMessage, error) {
	p := vtctldRoutingRulesAPIPath(req.Organization, req.Database, req.Branch)
	httpReq, err := s.client.newRequest(http.MethodGet, p, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}
	resp := &vtctldDataResponse{}
	if err := s.client.do(ctx, httpReq, resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (s *vtctldService) GetShard(ctx context.Context, req *VtctldGetShardRequest) (json.RawMessage, error) {
	p := vtctldShardAPIPath(req.Organization, req.Database, req.Branch)
	v := url.Values{}
	v.Set("keyspace", req.Keyspace)
	v.Set("shard", req.Shard)
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

func vtctldWorkflowAPIPath(org, db, branch, workflow string) string {
	return path.Join(vtctldWorkflowsAPIPath(org, db, branch), workflow)
}

func (s *vtctldService) StartWorkflow(ctx context.Context, req *VtctldStartWorkflowRequest) (json.RawMessage, error) {
	p := path.Join(vtctldWorkflowAPIPath(req.Organization, req.Database, req.Branch, req.Workflow), "start")
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

func (s *vtctldService) StopWorkflow(ctx context.Context, req *VtctldStopWorkflowRequest) (json.RawMessage, error) {
	p := path.Join(vtctldWorkflowAPIPath(req.Organization, req.Database, req.Branch, req.Workflow), "stop")
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

func vtctldThrottlerAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "vtctld", "throttler")
}

// GetThrottlerStatus reads the tablet throttler status from a single tablet.
func (s *vtctldService) GetThrottlerStatus(ctx context.Context, req *VtctldGetThrottlerStatusRequest) (json.RawMessage, error) {
	p := path.Join(vtctldThrottlerAPIPath(req.Organization, req.Database, req.Branch), "status")
	v := url.Values{}
	v.Set("tablet_alias", req.TabletAlias)
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

// SetShardTabletControl updates tablet controls on a shard via vtctld.
func (s *vtctldService) SetShardTabletControl(ctx context.Context, req *VtctldSetShardTabletControlRequest) (json.RawMessage, error) {
	p := vtctldShardTabletControlAPIPath(req.Organization, req.Database, req.Branch)
	httpReq, err := s.client.newRequest(http.MethodPut, p, req)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}
	resp := &vtctldDataResponse{}
	if err := s.client.do(ctx, httpReq, resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// RefreshStateByShard reloads tablet records for all tablets in a shard via vtctld.
func (s *vtctldService) RefreshStateByShard(ctx context.Context, req *VtctldRefreshStateByShardRequest) (json.RawMessage, error) {
	p := vtctldShardRefreshStateAPIPath(req.Organization, req.Database, req.Branch)
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

// CheckThrottler issues a throttler check against a single tablet.
func (s *vtctldService) CheckThrottler(ctx context.Context, req *VtctldCheckThrottlerRequest) (json.RawMessage, error) {
	p := path.Join(vtctldThrottlerAPIPath(req.Organization, req.Database, req.Branch), "check")
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

// UpdateThrottlerConfig updates the tablet throttler configuration for a keyspace.
func (s *vtctldService) UpdateThrottlerConfig(ctx context.Context, req *VtctldUpdateThrottlerConfigRequest) (json.RawMessage, error) {
	p := path.Join(vtctldThrottlerAPIPath(req.Organization, req.Database, req.Branch), "config")
	httpReq, err := s.client.newRequest(http.MethodPut, p, req)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}
	resp := &vtctldDataResponse{}
	if err := s.client.do(ctx, httpReq, resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}
