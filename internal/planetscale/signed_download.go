package planetscale

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
)

// downloadSignedURL uses an anonymous client to keep API credentials isolated to the request using them
func (c *Client) downloadSignedURL(ctx context.Context, req *http.Request) (io.ReadCloser, error) {
	httpClient := *c.client
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	res, err := httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("initial signed download request: %w", err)
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
			return nil, fmt.Errorf("signed download request: %w", err)
		}
		if res.StatusCode >= 300 {
			res.Body.Close()
			return nil, fmt.Errorf("signed download returned %s", http.StatusText(res.StatusCode))
		}
	case res.StatusCode >= 400:
		defer res.Body.Close()
		return nil, c.handleResponse(ctx, res, nil)
	}

	return res.Body, nil
}
