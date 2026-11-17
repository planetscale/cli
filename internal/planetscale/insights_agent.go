package planetscale

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

var _ InsightsAgentService = &insightsAgentService{}

// WithSchemaRecommendation sets the schema_recommendation query parameter.
func WithSchemaRecommendation(number int) ListOption {
	return func(opt *ListOptions) error {
		if number > 0 {
			opt.URLValues.Set("schema_recommendation", strconv.Itoa(number))
		}
		return nil
	}
}

// InsightsAgentService is an interface for communicating with the PlanetScale
// Insights Agent API.
type InsightsAgentService interface {
	ListRuns(context.Context, *ListInsightsAgentRunsRequest, ...ListOption) ([]*InsightsAgentRun, error)
	GetRun(context.Context, *GetInsightsAgentRunRequest) (*InsightsAgentRun, error)
	CreateRun(context.Context, *CreateInsightsAgentRunRequest) (*InsightsAgentRun, error)
	CancelRun(context.Context, *CancelInsightsAgentRunRequest) (*InsightsAgentRun, error)
	ListSchedules(context.Context, *ListInsightsAgentSchedulesRequest) ([]*InsightsAgentSchedule, error)
	UpdateSchedule(context.Context, *UpdateInsightsAgentScheduleRequest) (*InsightsAgentSchedule, error)
}

// InsightsAgentRun is a run of the Insights Agent on a branch.
type InsightsAgentRun struct {
	ID                    string                                  `json:"id"`
	Number                int                                     `json:"number"`
	RunType               string                                  `json:"run_type"`
	State                 string                                  `json:"state"`
	Outcome               string                                  `json:"outcome,omitempty"`
	Title                 string                                  `json:"title,omitempty"`
	Summary               string                                  `json:"summary,omitempty"`
	FailureMessage        string                                  `json:"failure_message,omitempty"`
	AgentModel            string                                  `json:"agent_model,omitempty"`
	Cost                  string                                  `json:"cost,omitempty"`
	HtmlURL               string                                  `json:"html_url,omitempty"`
	Actor                 *Actor                                  `json:"actor,omitempty"`
	Schedule              *InsightsAgentSchedule                  `json:"schedule,omitempty"`
	Repository            *InsightsAgentRepository                `json:"repository,omitempty"`
	QueryPatterns         []*InsightsAgentQueryPattern            `json:"query_patterns,omitempty"`
	SchemaRecommendations []*InsightsAgentRunSchemaRecommendation `json:"schema_recommendations,omitempty"`
	PullRequest           *InsightsAgentPullRequest               `json:"pull_request,omitempty"`
	Attempt               *int                                    `json:"attempt,omitempty"`
	Stages                []*InsightsAgentRunStage                `json:"stages,omitempty"`
	SubmittedAt           *time.Time                              `json:"submitted_at,omitempty"`
	StartedAt             *time.Time                              `json:"started_at,omitempty"`
	FinishedAt            *time.Time                              `json:"finished_at,omitempty"`
	CreatedAt             time.Time                               `json:"created_at"`
	UpdatedAt             time.Time                               `json:"updated_at"`
}

// InsightsAgentRepository is the repository and commit a run analyzed.
type InsightsAgentRepository struct {
	Provider      string `json:"provider"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch"`
	Commit        string `json:"commit"`
}

// InsightsAgentQueryPattern is a slow query pattern a run reviews.
type InsightsAgentQueryPattern struct {
	Fingerprint   string `json:"fingerprint"`
	Keyspace      string `json:"keyspace"`
	NormalizedSQL string `json:"normalized_sql,omitempty"`
	HtmlURL       string `json:"html_url,omitempty"`
}

// InsightsAgentRunSchemaRecommendation is a schema recommendation a run implements.
type InsightsAgentRunSchemaRecommendation struct {
	ID      string `json:"id"`
	Number  int    `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`
	HtmlURL string `json:"html_url"`
}

// InsightsAgentPullRequest is the pull request a run opened.
type InsightsAgentPullRequest struct {
	Number        int                         `json:"number"`
	URL           string                      `json:"url"`
	State         string                      `json:"state"`
	MergedAt      *time.Time                  `json:"merged_at,omitempty"`
	ClosedAt      *time.Time                  `json:"closed_at,omitempty"`
	DeployRequest *InsightsAgentDeployRequest `json:"deploy_request,omitempty"`
}

// InsightsAgentDeployRequest is the deploy request opened for a pull request's migration.
type InsightsAgentDeployRequest struct {
	ID      string `json:"id"`
	Number  int    `json:"number"`
	HtmlURL string `json:"html_url"`
}

// InsightsAgentRunStage is one stage of a run's latest attempt.
type InsightsAgentRunStage struct {
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	Message    string     `json:"message,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// InsightsAgentSchedule is a branch's schedule for one run type.
type InsightsAgentSchedule struct {
	ID            string     `json:"id"`
	RunType       string     `json:"run_type"`
	Enabled       bool       `json:"enabled"`
	FrequencyUnit string     `json:"frequency_unit"`
	ScheduleTime  string     `json:"schedule_time"`
	TimeZone      string     `json:"time_zone"`
	DayOfWeek     *int       `json:"day_of_week"`
	DayOfMonth    *int       `json:"day_of_month"`
	NextRunAt     *time.Time `json:"next_run_at"`
	Actor         *Actor     `json:"actor,omitempty"`
}

// ListInsightsAgentRunsRequest lists a branch's runs, newest first. Filter with
// WithFingerprint, WithKeyspace, and WithSchemaRecommendation.
type ListInsightsAgentRunsRequest struct {
	Organization string
	Database     string
	Branch       string
}

// GetInsightsAgentRunRequest gets a run by its number.
type GetInsightsAgentRunRequest struct {
	Organization string
	Database     string
	Branch       string
	Number       int
}

// CreateInsightsAgentRunRequest starts a run on a branch.
type CreateInsightsAgentRunRequest struct {
	Organization          string                       `json:"-"`
	Database              string                       `json:"-"`
	Branch                string                       `json:"-"`
	QueryPatterns         []*InsightsAgentQueryPattern `json:"query_patterns,omitempty"`
	SchemaRecommendations []int                        `json:"schema_recommendations,omitempty"`
}

// CancelInsightsAgentRunRequest cancels a pending or running run.
type CancelInsightsAgentRunRequest struct {
	Organization string
	Database     string
	Branch       string
	Number       int
}

// ListInsightsAgentSchedulesRequest lists a branch's schedules.
type ListInsightsAgentSchedulesRequest struct {
	Organization string
	Database     string
	Branch       string
}

// UpdateInsightsAgentScheduleRequest creates or changes the schedule for a run type.
// Unset fields keep their current value.
type UpdateInsightsAgentScheduleRequest struct {
	Organization  string  `json:"-"`
	Database      string  `json:"-"`
	Branch        string  `json:"-"`
	RunType       string  `json:"-"`
	Enabled       *bool   `json:"enabled,omitempty"`
	FrequencyUnit *string `json:"frequency_unit,omitempty"`
	ScheduleTime  *string `json:"schedule_time,omitempty"`
	TimeZone      *string `json:"time_zone,omitempty"`
	DayOfWeek     *int    `json:"day_of_week,omitempty"`
	DayOfMonth    *int    `json:"day_of_month,omitempty"`
}

type insightsAgentRunsResponse struct {
	Runs []*InsightsAgentRun `json:"data"`
}

type insightsAgentSchedulesResponse struct {
	Schedules []*InsightsAgentSchedule `json:"data"`
}

type insightsAgentService struct {
	client *Client
}

func (s *insightsAgentService) ListRuns(ctx context.Context, request *ListInsightsAgentRunsRequest, opts ...ListOption) ([]*InsightsAgentRun, error) {
	listOpts := defaultListOptions(opts...)
	req, err := s.client.newRequest(http.MethodGet, insightsAgentRunsAPIPath(request.Organization, request.Database, request.Branch), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	resp := &insightsAgentRunsResponse{}
	if err := s.client.do(ctx, req, resp); err != nil {
		return nil, err
	}
	return resp.Runs, nil
}

func (s *insightsAgentService) GetRun(ctx context.Context, request *GetInsightsAgentRunRequest) (*InsightsAgentRun, error) {
	return s.runRequest(ctx, http.MethodGet, insightsAgentRunAPIPath(request.Organization, request.Database, request.Branch, request.Number), nil)
}

func (s *insightsAgentService) CreateRun(ctx context.Context, request *CreateInsightsAgentRunRequest) (*InsightsAgentRun, error) {
	return s.runRequest(ctx, http.MethodPost, insightsAgentRunsAPIPath(request.Organization, request.Database, request.Branch), request)
}

func (s *insightsAgentService) CancelRun(ctx context.Context, request *CancelInsightsAgentRunRequest) (*InsightsAgentRun, error) {
	runPath := insightsAgentRunAPIPath(request.Organization, request.Database, request.Branch, request.Number)
	return s.runRequest(ctx, http.MethodPost, path.Join(runPath, "cancel"), nil)
}

func (s *insightsAgentService) ListSchedules(ctx context.Context, request *ListInsightsAgentSchedulesRequest) ([]*InsightsAgentSchedule, error) {
	req, err := s.client.newRequest(http.MethodGet, insightsAgentSchedulesAPIPath(request.Organization, request.Database, request.Branch), nil)
	if err != nil {
		return nil, err
	}

	resp := &insightsAgentSchedulesResponse{}
	if err := s.client.do(ctx, req, resp); err != nil {
		return nil, err
	}
	return resp.Schedules, nil
}

func (s *insightsAgentService) UpdateSchedule(ctx context.Context, request *UpdateInsightsAgentScheduleRequest) (*InsightsAgentSchedule, error) {
	schedulePath := path.Join(insightsAgentSchedulesAPIPath(request.Organization, request.Database, request.Branch), url.PathEscape(request.RunType))
	req, err := s.client.newRequest(http.MethodPatch, schedulePath, request)
	if err != nil {
		return nil, err
	}

	schedule := &InsightsAgentSchedule{}
	if err := s.client.do(ctx, req, schedule); err != nil {
		return nil, err
	}
	return schedule, nil
}

func (s *insightsAgentService) runRequest(ctx context.Context, method, apiPath string, body any) (*InsightsAgentRun, error) {
	req, err := s.client.newRequest(method, apiPath, body)
	if err != nil {
		return nil, err
	}

	run := &InsightsAgentRun{}
	if err := s.client.do(ctx, req, run); err != nil {
		return nil, err
	}
	return run, nil
}

func insightsAgentAPIPath(org, db, branch string) string {
	return path.Join(insightsAPIPath(org, db, branch), "agent")
}

func insightsAgentRunsAPIPath(org, db, branch string) string {
	return path.Join(insightsAgentAPIPath(org, db, branch), "runs")
}

func insightsAgentRunAPIPath(org, db, branch string, number int) string {
	return path.Join(insightsAgentRunsAPIPath(org, db, branch), strconv.Itoa(number))
}

func insightsAgentSchedulesAPIPath(org, db, branch string) string {
	return path.Join(insightsAgentAPIPath(org, db, branch), "schedules")
}
