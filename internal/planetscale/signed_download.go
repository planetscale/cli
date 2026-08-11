package planetscale

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-cleanhttp"
)

type signedDownloadTransportError struct {
	host  string
	cause error
}

func (e *signedDownloadTransportError) Error() string {
	return fmt.Sprintf("signed download from %s: %v", e.host, e.cause)
}

func (e *signedDownloadTransportError) Unwrap() error {
	return e.cause
}

// IsSignedDownloadTransportError reports whether a signed download failed before its response body was available.
func IsSignedDownloadTransportError(err error) bool {
	var transportErr *signedDownloadTransportError
	return errors.As(err, &transportErr)
}

func newSignedDownloadTransportError(request *http.Request, err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	return &signedDownloadTransportError{host: request.URL.Host, cause: err}
}

// The download endpoint redirects to blob storage. The client's
// credentials live in its transport, so following the redirect with that
// client would send the Authorization header to the storage host, which
// rejects requests carrying credentials beyond the presigned URL. Stop at
// the redirect and fetch its target with an unauthenticated client.
func (c *Client) downloadSignedURL(ctx context.Context, req *http.Request) (io.ReadCloser, error) {
	httpClient := *c.client
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	res, err := httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, newSignedDownloadTransportError(req, err)
	}

	switch {
	case res.StatusCode >= 300 && res.StatusCode < 400:
		location, err := res.Location()
		res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("parsing signed download location: %w", err)
		}

		blobReq, err := http.NewRequestWithContext(ctx, http.MethodGet, location.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("creating signed download request: %w", err)
		}
		blobReq.Header.Set("User-Agent", c.UserAgent)

		res, err = cleanhttp.DefaultClient().Do(blobReq)
		if err != nil {
			return nil, newSignedDownloadTransportError(blobReq, err)
		}
		if res.StatusCode >= 300 {
			res.Body.Close()
			return nil, newSignedDownloadTransportError(blobReq, fmt.Errorf("signed download returned %s", http.StatusText(res.StatusCode)))
		}
	case res.StatusCode >= 400:
		defer res.Body.Close()
		return nil, c.handleResponse(ctx, res, nil)
	}

	return res.Body, nil
}
