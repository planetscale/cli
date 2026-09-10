package planetscale

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/hashicorp/go-cleanhttp"
)

// LogSignature contains the signed URL and credentials used to query a
// branch's logs.
type LogSignature struct {
	Signature string `json:"sig"`
	ExpiresAt string `json:"exp"`
	URL       string `json:"url"`
}

// CreateLogSignatureRequest identifies the branch whose logs will be queried.
type CreateLogSignatureRequest struct {
	Organization string
	Database     string
	Branch       string
}

// QueryLogsRequest contains a signed logs URL and the query parameters to add
// to it.
type QueryLogsRequest struct {
	URL   string
	Query string
	Limit int
}

// LogsService communicates with the branch log-signature endpoint and the
// signed logs endpoint.
type LogsService interface {
	CreateSignature(context.Context, *CreateLogSignatureRequest) (*LogSignature, error)
	Query(context.Context, *QueryLogsRequest) (io.ReadCloser, error)
}

type logsService struct {
	client *Client
}

var _ LogsService = &logsService{}

// NewLogsService creates a service for querying branch logs.
func NewLogsService(client *Client) *logsService {
	return &logsService{client: client}
}

// CreateSignature returns a signed branch logs URL.
func (s *logsService) CreateSignature(ctx context.Context, createReq *CreateLogSignatureRequest) (*LogSignature, error) {
	reqPath := path.Join(databaseBranchAPIPath(createReq.Organization, createReq.Database, createReq.Branch), "logs/signatures")
	req, err := s.client.newRequest(http.MethodPost, reqPath, nil)
	if err != nil {
		return nil, fmt.Errorf("creating log signature request: %w", err)
	}

	signature := &LogSignature{}
	if err := s.client.do(ctx, req, signature); err != nil {
		return nil, err
	}

	return signature, nil
}

// Query fetches logs from a signed URL. It deliberately uses an
// unauthenticated HTTP client so PlanetScale API credentials are never sent to
// the logs host.
func (s *logsService) Query(ctx context.Context, queryReq *QueryLogsRequest) (io.ReadCloser, error) {
	u, err := url.Parse(queryReq.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing signed logs URL: %w", err)
	}
	if !u.IsAbs() || u.Host == "" {
		return nil, fmt.Errorf("parsing signed logs URL: URL must be absolute")
	}

	query := u.Query()
	query.Set("limit", fmt.Sprintf("%d", queryReq.Limit))
	query.Set("query", queryReq.Query)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating logs query request: %w", err)
	}
	req.Header.Set("User-Agent", s.client.UserAgent)

	res, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("querying branch logs: %w", err)
	}
	if res.StatusCode >= http.StatusMultipleChoices {
		defer res.Body.Close()
		body, readErr := io.ReadAll(io.LimitReader(res.Body, 4096))
		if readErr != nil {
			return nil, fmt.Errorf("querying branch logs: HTTP %s", res.Status)
		}
		return nil, fmt.Errorf("querying branch logs: HTTP %s: %s", res.Status, bodySnippet(body))
	}

	return res.Body, nil
}
