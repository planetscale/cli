package planetscale

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"
)

// QueryPatternsReport represents a query patterns report for a branch.
type QueryPatternsReport struct {
	PublicID    string    `json:"id"`
	State       string    `json:"state"`
	Actor       *Actor    `json:"actor"`
	URL         string    `json:"url"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
	FinishedAt  time.Time `json:"finished_at"`
}

type CreateQueryPatternsReportRequest struct {
	Organization string
	Database     string
	Branch       string
}

type GetQueryPatternsReportRequest struct {
	Organization string
	Database     string
	Branch       string
	Report       string
}

type DownloadQueryPatternsReportRequest struct {
	Organization string
	Database     string
	Branch       string
	Report       string
}

type ListQueryPatternsReportsRequest struct {
	Organization string
	Database     string
	Branch       string
}

type DeleteQueryPatternsReportRequest struct {
	Organization string
	Database     string
	Branch       string
	Report       string
}

// QueryPatternsService is an interface for communicating with the PlanetScale
// query patterns API endpoints.
type QueryPatternsService interface {
	CreateReport(context.Context, *CreateQueryPatternsReportRequest) (*QueryPatternsReport, error)
	ListReports(context.Context, *ListQueryPatternsReportsRequest, ...ListOption) ([]*QueryPatternsReport, error)
	GetReport(context.Context, *GetQueryPatternsReportRequest) (*QueryPatternsReport, error)
	DeleteReport(context.Context, *DeleteQueryPatternsReportRequest) error
	DownloadReport(context.Context, *DownloadQueryPatternsReportRequest) (io.ReadCloser, error)
}

type queryPatternsService struct {
	client *Client
}

var _ QueryPatternsService = &queryPatternsService{}

func NewQueryPatternsService(client *Client) *queryPatternsService {
	return &queryPatternsService{
		client: client,
	}
}

// CreateReport starts generating a new query patterns report for a branch.
func (s *queryPatternsService) CreateReport(ctx context.Context, createReq *CreateQueryPatternsReportRequest) (*QueryPatternsReport, error) {
	path := queryPatternsAPIPath(createReq.Organization, createReq.Database, createReq.Branch)
	req, err := s.client.newRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	report := &QueryPatternsReport{}
	if err := s.client.do(ctx, req, report); err != nil {
		return nil, err
	}

	return report, nil
}

// ListReports returns generated query patterns reports for a branch.
func (s *queryPatternsService) ListReports(ctx context.Context, listReq *ListQueryPatternsReportsRequest, opts ...ListOption) ([]*QueryPatternsReport, error) {
	listOpts := defaultListOptions(WithLimit(25))
	for _, opt := range opts {
		if err := opt(listOpts); err != nil {
			return nil, err
		}
	}

	req, err := s.client.newRequest(http.MethodGet, queryPatternsAPIPath(listReq.Organization, listReq.Database, listReq.Branch), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp := &CursorPaginatedResponse[*QueryPatternsReport]{}
	if err := s.client.do(ctx, req, resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetReport returns a single query patterns report for a branch.
func (s *queryPatternsService) GetReport(ctx context.Context, getReq *GetQueryPatternsReportRequest) (*QueryPatternsReport, error) {
	path := queryPatternsReportAPIPath(getReq.Organization, getReq.Database, getReq.Branch, getReq.Report)
	req, err := s.client.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	report := &QueryPatternsReport{}
	if err := s.client.do(ctx, req, report); err != nil {
		return nil, err
	}

	return report, nil
}

// DeleteReport deletes a generated query patterns report.
func (s *queryPatternsService) DeleteReport(ctx context.Context, deleteReq *DeleteQueryPatternsReportRequest) error {
	path := queryPatternsReportAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.Branch, deleteReq.Report)
	req, err := s.client.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	return s.client.do(ctx, req, nil)
}

// DownloadReport returns the content of a completed query patterns report.
// The caller must close the returned io.ReadCloser.
func (s *queryPatternsService) DownloadReport(ctx context.Context, downloadReq *DownloadQueryPatternsReportRequest) (io.ReadCloser, error) {
	reqPath := path.Join(queryPatternsReportAPIPath(downloadReq.Organization, downloadReq.Database, downloadReq.Branch, downloadReq.Report), "download")
	req, err := s.client.newRequest(http.MethodGet, reqPath, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	body, err := s.client.downloadSignedURL(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("downloading query patterns report: %w", err)
	}
	return decompressedReadCloser(body)
}

// decompressedReadCloser wraps body so gzip-compressed content is transparently
// decompressed, sniffing the gzip magic bytes so an uncompressed body passes
// through unchanged.
func decompressedReadCloser(body io.ReadCloser) (io.ReadCloser, error) {
	head := make([]byte, 2)
	n, err := io.ReadFull(body, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		body.Close()
		return nil, err
	}

	content := io.MultiReader(bytes.NewReader(head[:n]), body)
	if n == 2 && head[0] == 0x1f && head[1] == 0x8b {
		gz, err := gzip.NewReader(content)
		if err != nil {
			body.Close()
			return nil, err
		}
		content = gz
	}

	return &bodyReadCloser{Reader: content, body: body}, nil
}

type bodyReadCloser struct {
	io.Reader
	body io.ReadCloser
}

func (r *bodyReadCloser) Close() error { return r.body.Close() }

func queryPatternsAPIPath(org, db, branch string) string {
	return path.Join(databaseBranchAPIPath(org, db, branch), "query-patterns")
}

func queryPatternsReportAPIPath(org, db, branch, report string) string {
	return path.Join(queryPatternsAPIPath(org, db, branch), report)
}
