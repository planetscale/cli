package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

type NekiStorage struct {
	MinimumStorageBytes   *int64 `json:"minimum_storage_bytes,omitempty"`
	MaximumStorageBytes   *int64 `json:"maximum_storage_bytes,omitempty"`
	StorageAutoscaling    *bool  `json:"storage_autoscaling,omitempty"`
	StorageIOPS           *int64 `json:"storage_iops,omitempty"`
	StorageThroughputMiBs *int64 `json:"storage_throughput_mibs,omitempty"`
}

type NekiShardConfigurationProfile struct {
	Type                 string       `json:"type,omitempty"`
	Name                 string       `json:"name"`
	Architecture         string       `json:"architecture"`
	ClusterSize          string       `json:"cluster_size"`
	ClusterDisplayName   string       `json:"cluster_display_name"`
	Default              bool         `json:"default"`
	Metal                bool         `json:"metal"`
	Replicas             int          `json:"replicas"`
	PostgresMajorVersion int          `json:"postgres_major_version"`
	PostgresMinorVersion int          `json:"postgres_minor_version"`
	Shards               int          `json:"shards"`
	State                string       `json:"state"`
	Storage              *NekiStorage `json:"storage,omitempty"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

type ListNekiShardConfigurationProfilesRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

type GetNekiShardConfigurationProfileRequest struct {
	Organization         string `json:"-"`
	Database             string `json:"-"`
	Branch               string `json:"-"`
	ConfigurationProfile string `json:"-"`
}

type CreateNekiShardConfigurationProfileRequest struct {
	Organization         string       `json:"-"`
	Database             string       `json:"-"`
	Branch               string       `json:"-"`
	Name                 string       `json:"name"`
	ClusterSize          *string      `json:"cluster_size,omitempty"`
	Replicas             *int         `json:"replicas,omitempty"`
	PostgresMajorVersion *string      `json:"postgres_major_version,omitempty"`
	PostgresMinorVersion *string      `json:"postgres_minor_version,omitempty"`
	Storage              *NekiStorage `json:"storage,omitempty"`
}

type UpdateNekiShardConfigurationProfileRequest struct {
	Organization         string                       `json:"-"`
	Database             string                       `json:"-"`
	Branch               string                       `json:"-"`
	ConfigurationProfile string                       `json:"-"`
	Name                 *string                      `json:"name,omitempty"`
	ClusterSize          *string                      `json:"cluster_size,omitempty"`
	Replicas             *int                         `json:"replicas,omitempty"`
	Parameters           map[string]map[string]string `json:"parameters,omitempty"`
	PostgresMajorVersion *string                      `json:"postgres_major_version,omitempty"`
	PostgresMinorVersion *string                      `json:"postgres_minor_version,omitempty"`
	Storage              *NekiStorage                 `json:"storage,omitempty"`
}

type DeleteNekiShardConfigurationProfileRequest struct {
	Organization         string `json:"-"`
	Database             string `json:"-"`
	Branch               string `json:"-"`
	ConfigurationProfile string `json:"-"`
}

type GetDefaultNekiShardConfigurationProfileRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
}

type SetDefaultNekiShardConfigurationProfileRequest struct {
	Organization         string `json:"-"`
	Database             string `json:"-"`
	Branch               string `json:"-"`
	ConfigurationProfile string `json:"configuration_profile"`
}

type NekiParameter struct {
	Type          string      `json:"type,omitempty"`
	Name          string      `json:"name"`
	DisplayName   string      `json:"display_name"`
	Namespace     string      `json:"namespace"`
	Advanced      bool        `json:"advanced"`
	Category      *string     `json:"category"`
	Description   string      `json:"description"`
	ParameterType string      `json:"parameter_type"`
	DefaultValue  interface{} `json:"default_value"`
	Value         interface{} `json:"value"`
	Required      bool        `json:"required"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     *time.Time  `json:"updated_at"`
	Restart       bool        `json:"restart"`
	// Min, Max, and Step are numbers for plain numeric parameters but strings
	// for byte- and time-typed ones (for example, "128kB" or "10min").
	Max     any      `json:"max,omitempty"`
	Min     any      `json:"min,omitempty"`
	Step    any      `json:"step,omitempty"`
	Options []string `json:"options,omitempty"`
	Units   []string `json:"units,omitempty"`
	URL     string   `json:"url"`
	Actor   *Actor   `json:"actor,omitempty"`
}

type ListNekiShardConfigurationProfileParametersRequest struct {
	Organization         string `json:"-"`
	Database             string `json:"-"`
	Branch               string `json:"-"`
	ConfigurationProfile string `json:"-"`
	Extension            *bool  `json:"-"`
	Internal             *bool  `json:"-"`
}

type NekiExtension struct {
	Type        string           `json:"type,omitempty"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Enabled     bool             `json:"enabled"`
	Internal    bool             `json:"internal"`
	Loader      string           `json:"loader"`
	URL         string           `json:"url"`
	Parameters  []*NekiParameter `json:"parameters"`
}

type ListNekiShardConfigurationProfileExtensionsRequest struct {
	Organization         string `json:"-"`
	Database             string `json:"-"`
	Branch               string `json:"-"`
	ConfigurationProfile string `json:"-"`
}

type UpdateNekiShardConfigurationProfileExtensionRequest struct {
	Organization         string `json:"-"`
	Database             string `json:"-"`
	Branch               string `json:"-"`
	ConfigurationProfile string `json:"-"`
	Extension            string `json:"-"`
	Enabled              bool   `json:"enabled"`
}

type RunNekiShardConfigurationProfileMaintenanceRequest struct {
	Organization         string `json:"-"`
	Database             string `json:"-"`
	Branch               string `json:"-"`
	ConfigurationProfile string `json:"-"`
}

type RunNekiShardConfigurationProfilesMaintenanceRequest struct {
	Organization              string   `json:"-"`
	Database                  string   `json:"-"`
	Branch                    string   `json:"-"`
	ConfigurationProfileNames []string `json:"configuration_profile_names"`
}

type NekiShardConfigurationProfileChange struct {
	Type                       string                       `json:"type,omitempty"`
	ID                         string                       `json:"id"`
	State                      string                       `json:"state"`
	StartedAt                  *time.Time                   `json:"started_at"`
	CompletedAt                *time.Time                   `json:"completed_at"`
	CreatedAt                  time.Time                    `json:"created_at"`
	UpdatedAt                  time.Time                    `json:"updated_at"`
	Name                       string                       `json:"name"`
	PreviousName               string                       `json:"previous_name"`
	ClusterSize                string                       `json:"cluster_size"`
	ClusterDisplayName         string                       `json:"cluster_display_name"`
	Metal                      bool                         `json:"metal"`
	ClusterRank                int                          `json:"cluster_rank"`
	Replicas                   int                          `json:"replicas"`
	Parameters                 map[string]map[string]string `json:"parameters"`
	PreviousClusterSize        string                       `json:"previous_cluster_size"`
	PreviousClusterDisplayName string                       `json:"previous_cluster_display_name"`
	PreviousMetal              bool                         `json:"previous_metal"`
	PreviousClusterRank        int                          `json:"previous_cluster_rank"`
	PreviousReplicas           int                          `json:"previous_replicas"`
	PreviousParameters         map[string]map[string]string `json:"previous_parameters"`
	Storage                    *NekiStorage                 `json:"storage,omitempty"`
	PreviousStorage            *NekiStorage                 `json:"previous_storage,omitempty"`
	Actor                      *Actor                       `json:"actor,omitempty"`
}

type ListNekiShardConfigurationProfileChangesRequest struct {
	Organization         string `json:"-"`
	Database             string `json:"-"`
	Branch               string `json:"-"`
	ConfigurationProfile string `json:"-"`
	Period               string `json:"-"`
	CompletedAt          string `json:"-"`
	Page                 int    `json:"-"`
	PerPage              int    `json:"-"`
}

type GetNekiShardConfigurationProfileChangeRequest struct {
	Organization         string `json:"-"`
	Database             string `json:"-"`
	Branch               string `json:"-"`
	ConfigurationProfile string `json:"-"`
	Change               string `json:"-"`
}

type CancelNekiShardConfigurationProfileChangeRequest = GetNekiShardConfigurationProfileChangeRequest

type nekiShardConfigurationProfileChangesResponse struct {
	Changes []*NekiShardConfigurationProfileChange `json:"data"`
}

type NekiShardConfigurationProfilesService interface {
	List(context.Context, *ListNekiShardConfigurationProfilesRequest) ([]*NekiShardConfigurationProfile, error)
	Get(context.Context, *GetNekiShardConfigurationProfileRequest) (*NekiShardConfigurationProfile, error)
	Create(context.Context, *CreateNekiShardConfigurationProfileRequest) (*NekiShardConfigurationProfile, error)
	Update(context.Context, *UpdateNekiShardConfigurationProfileRequest) (*NekiShardConfigurationProfile, error)
	Delete(context.Context, *DeleteNekiShardConfigurationProfileRequest) error
	GetDefault(context.Context, *GetDefaultNekiShardConfigurationProfileRequest) (*NekiShardConfigurationProfile, error)
	SetDefault(context.Context, *SetDefaultNekiShardConfigurationProfileRequest) (*NekiShardConfigurationProfile, error)
	ListParameters(context.Context, *ListNekiShardConfigurationProfileParametersRequest) ([]*NekiParameter, error)
	ListExtensions(context.Context, *ListNekiShardConfigurationProfileExtensionsRequest) ([]*NekiExtension, error)
	UpdateExtension(context.Context, *UpdateNekiShardConfigurationProfileExtensionRequest) (*NekiExtension, error)
	RunMaintenance(context.Context, *RunNekiShardConfigurationProfileMaintenanceRequest) error
	RunBulkMaintenance(context.Context, *RunNekiShardConfigurationProfilesMaintenanceRequest) error
	ListChanges(context.Context, *ListNekiShardConfigurationProfileChangesRequest) ([]*NekiShardConfigurationProfileChange, error)
	GetChange(context.Context, *GetNekiShardConfigurationProfileChangeRequest) (*NekiShardConfigurationProfileChange, error)
	CancelChange(context.Context, *CancelNekiShardConfigurationProfileChangeRequest) error
}

type nekiShardConfigurationProfilesService struct{ client *Client }

var _ NekiShardConfigurationProfilesService = &nekiShardConfigurationProfilesService{}

func (s *nekiShardConfigurationProfilesService) List(ctx context.Context, r *ListNekiShardConfigurationProfilesRequest) ([]*NekiShardConfigurationProfile, error) {
	request, err := s.client.newRequest(http.MethodGet, nekiShardConfigurationProfilesAPIPath(r.Organization, r.Database, r.Branch), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to list shard configuration profiles: %w", err)
	}
	profiles := []*NekiShardConfigurationProfile{}
	if err := s.client.do(ctx, request, &profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (s *nekiShardConfigurationProfilesService) Get(ctx context.Context, r *GetNekiShardConfigurationProfileRequest) (*NekiShardConfigurationProfile, error) {
	return s.getProfile(ctx, http.MethodGet, nekiShardConfigurationProfileAPIPath(r.Organization, r.Database, r.Branch, r.ConfigurationProfile), nil)
}

func (s *nekiShardConfigurationProfilesService) Create(ctx context.Context, r *CreateNekiShardConfigurationProfileRequest) (*NekiShardConfigurationProfile, error) {
	return s.getProfile(ctx, http.MethodPost, nekiShardConfigurationProfilesAPIPath(r.Organization, r.Database, r.Branch), r)
}

func (s *nekiShardConfigurationProfilesService) Update(ctx context.Context, r *UpdateNekiShardConfigurationProfileRequest) (*NekiShardConfigurationProfile, error) {
	return s.getProfile(ctx, http.MethodPatch, nekiShardConfigurationProfileAPIPath(r.Organization, r.Database, r.Branch, r.ConfigurationProfile), r)
}

func (s *nekiShardConfigurationProfilesService) Delete(ctx context.Context, r *DeleteNekiShardConfigurationProfileRequest) error {
	request, err := s.client.newRequest(http.MethodDelete, nekiShardConfigurationProfileAPIPath(r.Organization, r.Database, r.Branch, r.ConfigurationProfile), nil)
	if err != nil {
		return fmt.Errorf("error creating request to delete shard configuration profile: %w", err)
	}
	return s.client.do(ctx, request, nil)
}

func (s *nekiShardConfigurationProfilesService) GetDefault(ctx context.Context, r *GetDefaultNekiShardConfigurationProfileRequest) (*NekiShardConfigurationProfile, error) {
	return s.getProfile(ctx, http.MethodGet, nekiDefaultShardConfigurationProfileAPIPath(r.Organization, r.Database, r.Branch), nil)
}

func (s *nekiShardConfigurationProfilesService) SetDefault(ctx context.Context, r *SetDefaultNekiShardConfigurationProfileRequest) (*NekiShardConfigurationProfile, error) {
	return s.getProfile(ctx, http.MethodPut, nekiDefaultShardConfigurationProfileAPIPath(r.Organization, r.Database, r.Branch), r)
}

func (s *nekiShardConfigurationProfilesService) ListParameters(ctx context.Context, r *ListNekiShardConfigurationProfileParametersRequest) ([]*NekiParameter, error) {
	values := defaultListOptions()
	if r.Extension != nil {
		values.URLValues.Set("extension", fmt.Sprintf("%t", *r.Extension))
	}
	if r.Internal != nil {
		values.URLValues.Set("internal", fmt.Sprintf("%t", *r.Internal))
	}
	request, err := s.client.newRequest(http.MethodGet, path.Join(nekiShardConfigurationProfileAPIPath(r.Organization, r.Database, r.Branch, r.ConfigurationProfile), "parameters"), nil, WithQueryParams(*values.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request to list shard configuration profile parameters: %w", err)
	}
	parameters := []*NekiParameter{}
	if err := s.client.do(ctx, request, &parameters); err != nil {
		return nil, err
	}
	return parameters, nil
}

func (s *nekiShardConfigurationProfilesService) ListExtensions(ctx context.Context, r *ListNekiShardConfigurationProfileExtensionsRequest) ([]*NekiExtension, error) {
	request, err := s.client.newRequest(http.MethodGet, path.Join(nekiShardConfigurationProfileAPIPath(r.Organization, r.Database, r.Branch, r.ConfigurationProfile), "extensions"), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to list shard configuration profile extensions: %w", err)
	}
	extensions := []*NekiExtension{}
	if err := s.client.do(ctx, request, &extensions); err != nil {
		return nil, err
	}
	return extensions, nil
}

func (s *nekiShardConfigurationProfilesService) UpdateExtension(ctx context.Context, r *UpdateNekiShardConfigurationProfileExtensionRequest) (*NekiExtension, error) {
	request, err := s.client.newRequest(http.MethodPatch, path.Join(nekiShardConfigurationProfileAPIPath(r.Organization, r.Database, r.Branch, r.ConfigurationProfile), "extensions", r.Extension), r)
	if err != nil {
		return nil, fmt.Errorf("error creating request to update shard configuration profile extension: %w", err)
	}
	extension := &NekiExtension{}
	if err := s.client.do(ctx, request, extension); err != nil {
		return nil, err
	}
	return extension, nil
}

func (s *nekiShardConfigurationProfilesService) RunMaintenance(ctx context.Context, r *RunNekiShardConfigurationProfileMaintenanceRequest) error {
	request, err := s.client.newRequest(http.MethodPost, path.Join(nekiShardConfigurationProfileAPIPath(r.Organization, r.Database, r.Branch, r.ConfigurationProfile), "maintenance"), nil)
	if err != nil {
		return fmt.Errorf("error creating request to run shard configuration profile maintenance: %w", err)
	}
	return s.client.do(ctx, request, nil)
}

func (s *nekiShardConfigurationProfilesService) RunBulkMaintenance(ctx context.Context, r *RunNekiShardConfigurationProfilesMaintenanceRequest) error {
	request, err := s.client.newRequest(http.MethodPost, path.Join(nekiShardConfigurationProfilesAPIPath(r.Organization, r.Database, r.Branch), "maintenance"), r)
	if err != nil {
		return fmt.Errorf("error creating request to run shard configuration profile maintenance: %w", err)
	}
	return s.client.do(ctx, request, nil)
}

func (s *nekiShardConfigurationProfilesService) ListChanges(ctx context.Context, r *ListNekiShardConfigurationProfileChangesRequest) ([]*NekiShardConfigurationProfileChange, error) {
	values := defaultListOptions(WithPeriod(r.Period), WithPage(r.Page), WithPerPage(r.PerPage))
	if r.CompletedAt != "" {
		values.URLValues.Set("completed_at", r.CompletedAt)
	}
	request, err := s.client.newRequest(http.MethodGet, nekiShardConfigurationProfileChangesAPIPath(r.Organization, r.Database, r.Branch, r.ConfigurationProfile), nil, WithQueryParams(*values.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request to list shard configuration profile changes: %w", err)
	}
	response := &nekiShardConfigurationProfileChangesResponse{}
	if err := s.client.do(ctx, request, response); err != nil {
		return nil, err
	}
	return response.Changes, nil
}

func (s *nekiShardConfigurationProfilesService) GetChange(ctx context.Context, r *GetNekiShardConfigurationProfileChangeRequest) (*NekiShardConfigurationProfileChange, error) {
	request, err := s.client.newRequest(http.MethodGet, nekiShardConfigurationProfileChangeAPIPath(r.Organization, r.Database, r.Branch, r.ConfigurationProfile, r.Change), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to get shard configuration profile change: %w", err)
	}
	change := &NekiShardConfigurationProfileChange{}
	if err := s.client.do(ctx, request, change); err != nil {
		return nil, err
	}
	return change, nil
}

func (s *nekiShardConfigurationProfilesService) CancelChange(ctx context.Context, r *CancelNekiShardConfigurationProfileChangeRequest) error {
	request, err := s.client.newRequest(http.MethodDelete, nekiShardConfigurationProfileChangeAPIPath(r.Organization, r.Database, r.Branch, r.ConfigurationProfile, r.Change), nil)
	if err != nil {
		return fmt.Errorf("error creating request to cancel shard configuration profile change: %w", err)
	}
	return s.client.do(ctx, request, nil)
}

func (s *nekiShardConfigurationProfilesService) getProfile(ctx context.Context, method, pathStr string, body interface{}) (*NekiShardConfigurationProfile, error) {
	request, err := s.client.newRequest(method, pathStr, body)
	if err != nil {
		return nil, fmt.Errorf("error creating shard configuration profile request: %w", err)
	}
	profile := &NekiShardConfigurationProfile{}
	if err := s.client.do(ctx, request, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func nekiShardConfigurationProfilesAPIPath(org, database, branch string) string {
	return path.Join(databaseBranchAPIPath(org, database, branch), "configuration-profiles")
}

func nekiShardConfigurationProfileAPIPath(org, database, branch, profile string) string {
	return path.Join(nekiShardConfigurationProfilesAPIPath(org, database, branch), profile)
}

func nekiDefaultShardConfigurationProfileAPIPath(org, database, branch string) string {
	return path.Join(databaseBranchAPIPath(org, database, branch), "default-configuration-profile")
}

func nekiShardConfigurationProfileChangesAPIPath(org, database, branch, profile string) string {
	return path.Join(nekiShardConfigurationProfileAPIPath(org, database, branch, profile), "changes")
}

func nekiShardConfigurationProfileChangeAPIPath(org, database, branch, profile, change string) string {
	return path.Join(nekiShardConfigurationProfileChangesAPIPath(org, database, branch, profile), change)
}
