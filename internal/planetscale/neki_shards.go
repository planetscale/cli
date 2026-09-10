package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

// NekiShard represents a shard managed by a Neki branch.
type NekiShard struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	DisplayName          *string   `json:"display_name"`
	ConfigurationProfile string    `json:"configuration_profile"`
	Ready                bool      `json:"ready"`
	Authoritative        bool      `json:"authoritative"`
	CreatedAt            time.Time `json:"created_at"`
}

type nekiShardsResponse struct {
	Shards []*NekiShard `json:"data"`
}

// NekiShardAssignment is the result of assigning one shard to a configuration profile.
type NekiShardAssignment struct {
	ID     string                 `json:"id"`
	Status string                 `json:"status"`
	Error  map[string]interface{} `json:"error,omitempty"`
}

type ListNekiShardsRequest struct {
	Organization                string `json:"-"`
	Database                    string `json:"-"`
	Branch                      string `json:"-"`
	ConfigurationProfile        string `json:"-"`
	ExcludeConfigurationProfile string `json:"-"`
	Query                       string `json:"-"`
	Page                        int    `json:"-"`
	PerPage                     int    `json:"-"`
}

type CreateNekiShardsRequest struct {
	Organization         string `json:"-"`
	Database             string `json:"-"`
	Branch               string `json:"-"`
	ConfigurationProfile string `json:"-"`
	Count                int    `json:"count"`
}

type CreateNekiShardsResponse struct {
	Created  int      `json:"created"`
	ShardIDs []string `json:"shard_ids"`
}

type AssignNekiShardsRequest struct {
	Organization         string   `json:"-"`
	Database             string   `json:"-"`
	Branch               string   `json:"-"`
	ConfigurationProfile string   `json:"-"`
	ShardIDs             []string `json:"shard_ids"`
}

type UpdateNekiShardRequest struct {
	Organization string  `json:"-"`
	Database     string  `json:"-"`
	Branch       string  `json:"-"`
	Shard        string  `json:"-"`
	DisplayName  *string `json:"display_name"`
}

type GetNekiShardRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Shard        string `json:"-"`
}

type DeleteNekiShardRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Shard        string `json:"-"`
}

// NekiShardsService manages shards belonging to Neki branches.
type NekiShardsService interface {
	List(context.Context, *ListNekiShardsRequest) ([]*NekiShard, error)
	Get(context.Context, *GetNekiShardRequest) (*NekiShard, error)
	Create(context.Context, *CreateNekiShardsRequest) (*CreateNekiShardsResponse, error)
	Assign(context.Context, *AssignNekiShardsRequest) ([]*NekiShardAssignment, error)
	Update(context.Context, *UpdateNekiShardRequest) (*NekiShard, error)
	Delete(context.Context, *DeleteNekiShardRequest) error
}

type nekiShardsService struct {
	client *Client
}

var _ NekiShardsService = &nekiShardsService{}

func NewNekiShardsService(client *Client) *nekiShardsService {
	return &nekiShardsService{client: client}
}

func (s *nekiShardsService) List(ctx context.Context, listReq *ListNekiShardsRequest) ([]*NekiShard, error) {
	pathStr := nekiShardsAPIPath(listReq.Organization, listReq.Database, listReq.Branch)
	if listReq.ConfigurationProfile != "" {
		pathStr = nekiShardConfigurationProfileShardsAPIPath(listReq.Organization, listReq.Database, listReq.Branch, listReq.ConfigurationProfile)
	}

	query := defaultListOptions(
		WithSearch(listReq.Query),
		WithPage(listReq.Page),
		WithPerPage(listReq.PerPage),
	)
	if listReq.ExcludeConfigurationProfile != "" {
		query.URLValues.Set("exclude_configuration_profile", listReq.ExcludeConfigurationProfile)
	}

	req, err := s.client.newRequest(http.MethodGet, pathStr, nil, WithQueryParams(*query.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating http request to list shards: %w", err)
	}

	response := &nekiShardsResponse{}
	if err := s.client.do(ctx, req, response); err != nil {
		return nil, err
	}

	return response.Shards, nil
}

func (s *nekiShardsService) Get(ctx context.Context, getReq *GetNekiShardRequest) (*NekiShard, error) {
	req, err := s.client.newRequest(http.MethodGet, nekiShardAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.Shard), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request to get shard: %w", err)
	}

	shard := &NekiShard{}
	if err := s.client.do(ctx, req, shard); err != nil {
		return nil, err
	}

	return shard, nil
}

func (s *nekiShardsService) Create(ctx context.Context, createReq *CreateNekiShardsRequest) (*CreateNekiShardsResponse, error) {
	pathStr := path.Join(nekiShardConfigurationProfileShardsAPIPath(createReq.Organization, createReq.Database, createReq.Branch, createReq.ConfigurationProfile), "bulk")
	req, err := s.client.newRequest(http.MethodPost, pathStr, createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request to create shards: %w", err)
	}

	response := &CreateNekiShardsResponse{}
	if err := s.client.do(ctx, req, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *nekiShardsService) Assign(ctx context.Context, assignReq *AssignNekiShardsRequest) ([]*NekiShardAssignment, error) {
	pathStr := nekiShardConfigurationProfileShardsAPIPath(assignReq.Organization, assignReq.Database, assignReq.Branch, assignReq.ConfigurationProfile)
	req, err := s.client.newRequest(http.MethodPatch, pathStr, assignReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request to assign shards: %w", err)
	}

	assignments := []*NekiShardAssignment{}
	if err := s.client.do(ctx, req, &assignments); err != nil {
		return nil, err
	}

	return assignments, nil
}

func (s *nekiShardsService) Update(ctx context.Context, updateReq *UpdateNekiShardRequest) (*NekiShard, error) {
	pathStr := nekiShardAPIPath(updateReq.Organization, updateReq.Database, updateReq.Branch, updateReq.Shard)
	req, err := s.client.newRequest(http.MethodPatch, pathStr, updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request to update shard: %w", err)
	}

	shard := &NekiShard{}
	if err := s.client.do(ctx, req, shard); err != nil {
		return nil, err
	}

	return shard, nil
}

func (s *nekiShardsService) Delete(ctx context.Context, deleteReq *DeleteNekiShardRequest) error {
	pathStr := nekiShardAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.Branch, deleteReq.Shard)
	req, err := s.client.newRequest(http.MethodDelete, pathStr, nil)
	if err != nil {
		return fmt.Errorf("error creating http request to delete shard: %w", err)
	}

	return s.client.do(ctx, req, nil)
}

func nekiShardsAPIPath(org, database, branch string) string {
	return path.Join(databaseBranchAPIPath(org, database, branch), "shards")
}

func nekiShardAPIPath(org, database, branch, shard string) string {
	return path.Join(nekiShardsAPIPath(org, database, branch), shard)
}

func nekiShardConfigurationProfileShardsAPIPath(org, database, branch, configurationProfile string) string {
	return path.Join(databaseBranchAPIPath(org, database, branch), "configuration-profiles", configurationProfile, "shards")
}
