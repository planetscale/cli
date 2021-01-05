package psapi

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/planetscale/cli/testutil"
	"github.com/stretchr/testify/assert"
)

func TestDo(t *testing.T) {
	tests := []struct {
		desc       string
		response   string
		setupReq   func(client *Client, path string) (*http.Request, error)
		path       string
		statusCode int
	}{
		{
			desc:       "returns an HTTP response 200 when everything is fine",
			statusCode: http.StatusOK,
			setupReq: func(client *Client, path string) (*http.Request, error) {
				req, err := client.NewRequest(http.MethodGet, path, nil)
				if err != nil {
					return nil, err
				}
				return req, nil
			},
			response: `{}`,
			path:     "/api-endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			srv, cleanup := testutil.SetupServer(func(mux *http.ServeMux) {
				mux.HandleFunc(fmt.Sprintf("%s", tt.path), func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)

					_, err := w.Write([]byte(tt.response))
					if err != nil {
						t.Fatal(err)
					}
				})
			})

			t.Cleanup(func() {
				cleanup()
			})

			client, err := NewClient(cleanhttp.DefaultClient(), SetBaseURL(srv.URL))
			if err != nil {
				t.Fatal(err)
			}
			req, err := tt.setupReq(client, tt.path)
			if err != nil {
				t.Fatal(err)
			}

			res, err := client.Do(context.Background(), req, nil)
			if err != nil {
				// TODO(iheanyi): Check and see if the errors are expected here.
				t.Fatal(err)
				return
			}

			assert.NotNil(t, res)
			assert.Equal(t, res.StatusCode, tt.statusCode)
		})
	}
}
