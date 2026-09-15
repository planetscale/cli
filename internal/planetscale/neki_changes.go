package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"time"
)

// NekiChange is a branch-wide change request. JSON keeps the API object so
// resource-specific fields (size, parameters, flags) are not dropped.
type NekiChange struct {
	Type        string     `json:"type,omitempty"`
	ID          string     `json:"id"`
	State       string     `json:"state"`
	CanDelete   bool       `json:"can_delete"`
	TargetID    string     `json:"target_id,omitempty"`
	TargetType  string     `json:"target_type"`
	TargetName  string     `json:"target_name"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Actor       *Actor     `json:"actor,omitempty"`
	raw         json.RawMessage
}

func (c *NekiChange) UnmarshalJSON(data []byte) error {
	type fields NekiChange
	var parsed fields
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	*c = NekiChange(parsed)
	c.raw = append(json.RawMessage(nil), data...)
	return nil
}

func (c *NekiChange) MarshalJSON() ([]byte, error) {
	if len(c.raw) > 0 {
		return c.raw, nil
	}
	type fields NekiChange
	return json.Marshal(fields(*c))
}

type ListNekiChangesRequest struct {
	Organization string   `json:"-"`
	Database     string   `json:"-"`
	Branch       string   `json:"-"`
	States       []string `json:"-"`
	TargetTypes  []string `json:"-"`
	TargetID     string   `json:"-"`
	Period       string   `json:"-"`
	CompletedAt  string   `json:"-"`
	Page         int      `json:"-"`
	PerPage      int      `json:"-"`
}

type GetNekiChangeRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	Branch       string `json:"-"`
	Change       string `json:"-"`
}

type CancelNekiChangeRequest = GetNekiChangeRequest

type nekiChangesResponse struct {
	Changes []*NekiChange `json:"data"`
}

// NekiChangesService lists, shows, and cancels change requests across a Neki branch.
type NekiChangesService interface {
	List(context.Context, *ListNekiChangesRequest) ([]*NekiChange, error)
	Get(context.Context, *GetNekiChangeRequest) (*NekiChange, error)
	Cancel(context.Context, *CancelNekiChangeRequest) error
}

type nekiChangesService struct{ client *Client }

var _ NekiChangesService = &nekiChangesService{}

func (s *nekiChangesService) List(ctx context.Context, r *ListNekiChangesRequest) ([]*NekiChange, error) {
	values := defaultListOptions(WithPeriod(r.Period), WithPage(r.Page), WithPerPage(r.PerPage))
	if r.CompletedAt != "" {
		values.URLValues.Set("completed_at", r.CompletedAt)
	}
	if r.TargetID != "" {
		values.URLValues.Set("target_id", r.TargetID)
	}
	addQueryValues(*values.URLValues, "state[]", r.States)
	addQueryValues(*values.URLValues, "target_types[]", r.TargetTypes)
	request, err := s.client.newRequest(http.MethodGet, nekiChangesAPIPath(r.Organization, r.Database, r.Branch), nil, WithQueryParams(*values.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request to list Neki changes: %w", err)
	}
	response := &nekiChangesResponse{}
	if err := s.client.do(ctx, request, response); err != nil {
		return nil, err
	}
	return response.Changes, nil
}

func (s *nekiChangesService) Get(ctx context.Context, r *GetNekiChangeRequest) (*NekiChange, error) {
	request, err := s.client.newRequest(http.MethodGet, nekiChangeAPIPath(r.Organization, r.Database, r.Branch, r.Change), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to get Neki change: %w", err)
	}
	change := &NekiChange{}
	if err := s.client.do(ctx, request, change); err != nil {
		return nil, err
	}
	return change, nil
}

func (s *nekiChangesService) Cancel(ctx context.Context, r *CancelNekiChangeRequest) error {
	request, err := s.client.newRequest(http.MethodDelete, nekiChangeAPIPath(r.Organization, r.Database, r.Branch, r.Change), nil)
	if err != nil {
		return fmt.Errorf("error creating request to cancel Neki change: %w", err)
	}
	return s.client.do(ctx, request, nil)
}

func nekiChangesAPIPath(org, database, branch string) string {
	return path.Join(databaseBranchAPIPath(org, database, branch), "neki-changes")
}

func nekiChangeAPIPath(org, database, branch, change string) string {
	return path.Join(nekiChangesAPIPath(org, database, branch), change)
}
