package planetscale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

type AuthAttemptExportFilters struct {
	SourceIPs        []string `json:"source_ips,omitempty"`
	Branches         []string `json:"branches,omitempty"`
	Outcomes         []string `json:"outcomes,omitempty"`
	Usernames        []string `json:"usernames,omitempty"`
	StartupDatabases []string `json:"startup_databases,omitempty"`
	FailureReasons   []string `json:"failure_reasons,omitempty"`
	BackendRoutes    []string `json:"backend_routes,omitempty"`
}

type CreateAuthAttemptExportRequest struct {
	Organization string                   `json:"-"`
	StartAt      time.Time                `json:"start_at"`
	EndAt        time.Time                `json:"end_at"`
	Format       string                   `json:"format"`
	Filters      AuthAttemptExportFilters `json:"filters,omitempty"`
}

type GetAuthAttemptExportRequest struct {
	Organization string
	Export       string
}

type DownloadAuthAttemptExportRequest struct {
	Organization string
	Export       string
}

type AuthAttemptExport struct {
	PublicID                string                   `json:"id"`
	State                   string                   `json:"state"`
	Expired                 bool                     `json:"expired"`
	StartAt                 time.Time                `json:"start_at"`
	EndAt                   time.Time                `json:"end_at"`
	Filters                 AuthAttemptExportFilters `json:"filters"`
	Format                  string                   `json:"format"`
	ResolvedBranchPublicIDs []string                 `json:"resolved_branch_public_ids"`
	CreatedAt               time.Time                `json:"created_at"`
	StartedAt               *time.Time               `json:"started_at"`
	GeneratedAt             *time.Time               `json:"generated_at"`
	FinishedAt              *time.Time               `json:"finished_at"`
	ExpiresAt               *time.Time               `json:"expires_at"`
	FailureReason           string                   `json:"failure_reason"`
	FailureDetail           string                   `json:"failure_detail"`
	RecoveryHint            string                   `json:"recovery_hint"`
	RetryAfter              time.Duration            `json:"-"`
}

type AuthAttemptExportsService interface {
	CreateExport(context.Context, *CreateAuthAttemptExportRequest) (*AuthAttemptExport, error)
	GetExport(context.Context, *GetAuthAttemptExportRequest) (*AuthAttemptExport, error)
	DownloadExport(context.Context, *DownloadAuthAttemptExportRequest) (io.ReadCloser, error)
}

type authAttemptExportsService struct {
	client *Client
}

var _ AuthAttemptExportsService = &authAttemptExportsService{}

func (s *authAttemptExportsService) CreateExport(ctx context.Context, createReq *CreateAuthAttemptExportRequest) (*AuthAttemptExport, error) {
	req, err := s.client.newRequest(http.MethodPost, authAttemptExportsAPIPath(createReq.Organization), createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	export := &AuthAttemptExport{}
	headers, err := s.client.doWithHeaders(ctx, req, export)
	if err != nil {
		return nil, err
	}
	export.RetryAfter = parseRetryAfter(headers.Get("Retry-After"))
	return export, nil
}

func (s *authAttemptExportsService) GetExport(ctx context.Context, getReq *GetAuthAttemptExportRequest) (*AuthAttemptExport, error) {
	req, err := s.client.newRequest(http.MethodGet, authAttemptExportAPIPath(getReq.Organization, getReq.Export), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	export := &AuthAttemptExport{}
	headers, err := s.client.doWithHeaders(ctx, req, export)
	if err != nil {
		if expiredErr := authAttemptExportExpiredError(err); expiredErr != nil {
			return nil, expiredErr
		}
		return nil, err
	}
	export.RetryAfter = parseRetryAfter(headers.Get("Retry-After"))
	return export, nil
}

func (s *authAttemptExportsService) DownloadExport(ctx context.Context, downloadReq *DownloadAuthAttemptExportRequest) (io.ReadCloser, error) {
	reqPath := path.Join(authAttemptExportAPIPath(downloadReq.Organization, downloadReq.Export), "download")
	req, err := s.client.newRequest(http.MethodGet, reqPath, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}
	body, err := s.client.downloadSignedURL(ctx, req)
	if err != nil {
		if expiredErr := authAttemptExportExpiredError(err); expiredErr != nil {
			return nil, expiredErr
		}
		return nil, fmt.Errorf("downloading auth attempt export: %w", err)
	}
	return body, nil
}

func authAttemptExportExpiredError(err error) error {
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Meta["http_status"] != http.StatusText(http.StatusGone) {
		return nil
	}

	export := &AuthAttemptExport{}
	if json.Unmarshal([]byte(apiErr.Meta["body"]), export) != nil || !export.Expired || export.PublicID == "" || export.RecoveryHint == "" {
		return nil
	}
	return fmt.Errorf("auth attempt export %s expired: %s", export.PublicID, export.RecoveryHint)
}

func authAttemptExportsAPIPath(org string) string {
	return path.Join("v1/organizations", org, "auth-attempt-exports")
}

func authAttemptExportAPIPath(org, export string) string {
	return path.Join(authAttemptExportsAPIPath(org), export)
}

func parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	seconds, err := strconv.ParseUint(value, 10, 64)
	if err == nil {
		if seconds > 0 && seconds <= uint64((time.Duration(1<<63-1))/time.Second) {
			return time.Duration(seconds) * time.Second
		}
		return 0
	}

	when, err := http.ParseTime(value)
	if err != nil || !when.After(time.Now()) {
		return 0
	}
	return time.Until(when)
}
