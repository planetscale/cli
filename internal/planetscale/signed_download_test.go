package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestAuthAttemptExportDownloadSanitizesSignedURLTransportError(t *testing.T) {
	c := qt.New(t)

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:1/sensitive-object?X-Amz-Signature=secret", http.StatusFound)
	}))
	t.Cleanup(api.Close)

	client, err := NewClient(WithBaseURL(api.URL))
	c.Assert(err, qt.IsNil)

	_, err = client.AuthAttemptExports.DownloadExport(context.Background(), &DownloadAuthAttemptExportRequest{
		Organization: "my-org",
		Export:       "export1",
	})
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "127.0.0.1:1")
	c.Assert(err.Error(), qt.Not(qt.Contains), "sensitive-object")
	c.Assert(err.Error(), qt.Not(qt.Contains), "X-Amz-Signature")
	c.Assert(err.Error(), qt.Not(qt.Contains), "secret")
}
