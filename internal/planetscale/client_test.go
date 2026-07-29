package planetscale

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestDo(t *testing.T) {
	tests := []struct {
		desc          string
		response      string
		statusCode    int
		method        string
		expectedError error
		clientOptions []ClientOption
		wantHeaders   map[string]string
		body          interface{}
		v             interface{}
		want          interface{}
	}{
		{
			desc:       "returns an HTTP response and no error for 2xx responses",
			statusCode: http.StatusOK,
			response:   `{}`,
			method:     http.MethodGet,
			wantHeaders: map[string]string{
				"User-Agent": "planetscale-go/unknown",
			},
		},
		{
			desc:          "sets a custom header with the request option",
			statusCode:    http.StatusOK,
			response:      `{}`,
			method:        http.MethodGet,
			clientOptions: []ClientOption{WithUserAgent("test-user-agent"), WithRequestHeaders(map[string]string{"Test-Header": "test-value"})},
			wantHeaders: map[string]string{
				"Test-Header": "test-value",
				"User-Agent":  "test-user-agent planetscale-go/unknown",
			},
		},
		{
			desc:       "returns ErrorResponse for 4xx errors",
			statusCode: http.StatusNotFound,
			method:     http.MethodGet,
			response: `{
				"code": "not_found",
				"message": "Not Found"
			}`,
			expectedError: &Error{
				msg:  "Not Found",
				Code: ErrNotFound,
			},
		},
		{
			desc:       "maps bad_request errors to invalid",
			statusCode: http.StatusBadRequest,
			method:     http.MethodPost,
			response: `{
				"code": "bad_request",
				"message": "Bad Request"
			}`,
			expectedError: &Error{
				msg:  "Bad Request",
				Code: ErrInvalid,
			},
		},
		{
			desc:       "preserves raw API code for schema_mutation_blocked",
			statusCode: http.StatusUnprocessableEntity,
			method:     http.MethodPost,
			response: `{
				"code": "schema_mutation_blocked",
				"message": "MoveTables create in progress"
			}`,
			expectedError: &Error{
				msg:     "MoveTables create in progress",
				APICode: "schema_mutation_blocked",
			},
		},
		{
			desc:       "returns ErrorResponse for 5xx errors",
			statusCode: http.StatusInternalServerError,
			method:     http.MethodGet,
			response:   `{}`,
			expectedError: &Error{
				msg:  "received HTTP 500 with an unrecognized error response: {}",
				Code: ErrInternal,
			},
		},
		{
			desc:       "returns an HTTP response 200 when posting a request",
			statusCode: http.StatusOK,
			response: `
{ 
	"id": "509",
	"type": "database",
	"name": "foo-bar",
	"notes": ""
}`,
			body: &Database{
				Name: "foo-bar",
			},
			v: &Database{},
			want: &Database{
				Name: "foo-bar",
			},
		},
		{
			desc:       "returns an HTTP response 204 when deleting a request",
			statusCode: http.StatusNoContent,
			method:     http.MethodDelete,
			response:   "",
			body:       nil,
			v:          &Database{},
			want:       nil,
		},
		{
			desc:       "returns an non-204 HTTP response when deleting a request",
			statusCode: http.StatusAccepted,
			method:     http.MethodDelete,
			response: `{
			"id": "test"
			}`,
			body: nil,
			v:    &DatabaseDeletionRequest{},
			want: &DatabaseDeletionRequest{
				ID: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			c := qt.New(t)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)

				if tt.wantHeaders != nil {
					for key, value := range tt.wantHeaders {
						c.Assert(r.Header.Get(key), qt.Equals, value)
					}
				}

				res := []byte(tt.response)
				if tt.response == "" {
					res = nil
				}
				_, err := w.Write(res)
				if err != nil {
					t.Fatal(err)
				}
			}))
			t.Cleanup(ts.Close)

			opts := append(tt.clientOptions, WithBaseURL(ts.URL))
			client, err := NewClient(opts...)
			if err != nil {
				t.Fatal(err)
			}

			req, err := client.newRequest(tt.method, "/api-endpoint", tt.body)
			if err != nil {
				t.Fatal(err)
			}

			res, err := client.client.Do(req)
			c.Assert(err, qt.IsNil)
			defer res.Body.Close()

			err = client.handleResponse(ctx, res, &tt.v)
			if err != nil {
				if tt.expectedError != nil {
					c.Assert(tt.expectedError.Error(), qt.Equals, err.Error())
					if want, ok := tt.expectedError.(*Error); ok && want.APICode != "" {
						got, ok := err.(*Error)
						c.Assert(ok, qt.IsTrue)
						c.Assert(got.APICode, qt.Equals, want.APICode)
					}
				}
			}

			c.Assert(res, qt.Not(qt.IsNil))
			c.Assert(res.StatusCode, qt.Equals, tt.statusCode)

			if tt.v != nil && tt.want != nil {
				c.Assert(tt.want, qt.DeepEquals, tt.v)
			}
		})
	}
}

func TestHandleResponse_UsesStatusCodeForInvalidNotFoundBody(t *testing.T) {
	tests := []struct {
		desc string
		body string
	}{
		{
			desc: "empty object",
			body: `{}`,
		},
		{
			desc: "malformed json",
			body: `not-json`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			c := qt.New(t)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(tt.body))
				c.Assert(err, qt.IsNil)
			}))
			t.Cleanup(ts.Close)

			client, err := NewClient(WithBaseURL(ts.URL))
			c.Assert(err, qt.IsNil)

			req, err := client.newRequest(http.MethodGet, "/api-endpoint", nil)
			c.Assert(err, qt.IsNil)

			res, err := client.client.Do(req)
			c.Assert(err, qt.IsNil)
			defer res.Body.Close()

			err = client.handleResponse(context.Background(), res, nil)
			c.Assert(err, qt.Not(qt.IsNil))

			perr, ok := err.(*Error)
			c.Assert(ok, qt.IsTrue)
			c.Assert(perr.Code, qt.Equals, ErrNotFound)
			c.Assert(perr.Error(), qt.Equals, http.StatusText(http.StatusNotFound))
		})
	}
}

func TestSameHostCheckRedirect(t *testing.T) {
	tests := []struct {
		name                  string
		apiHost               string
		redirectURL           string
		expectUseLastResponse bool
	}{
		{
			name:                  "exact same host",
			apiHost:               "api.example.com",
			redirectURL:           "https://api.example.com/path",
			expectUseLastResponse: false,
		},
		{
			name:                  "same host different port still same hostname",
			apiHost:               "api.example.com",
			redirectURL:           "https://api.example.com:443/path",
			expectUseLastResponse: false,
		},
		{
			name:                  "case-insensitive hostname match",
			apiHost:               "API.Example.Com",
			redirectURL:           "https://api.example.com/path",
			expectUseLastResponse: false,
		},
		{
			name:                  "sibling subdomain is not same host",
			apiHost:               "api.planetscale.com",
			redirectURL:           "https://evil.planetscale.com/path",
			expectUseLastResponse: true,
		},
		{
			name:                  "different domain",
			apiHost:               "api.example.com",
			redirectURL:           "https://attacker.tld/path",
			expectUseLastResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			check := makeSameHostCheckRedirect(tt.apiHost)

			req, err := http.NewRequest(http.MethodGet, tt.redirectURL, nil)
			c.Assert(err, qt.IsNil)

			err = check(req, []*http.Request{})
			if tt.expectUseLastResponse {
				c.Assert(err, qt.Equals, http.ErrUseLastResponse)
			} else {
				c.Assert(err, qt.IsNil)
			}
		})
	}
}

func TestClient_DoesNotSendCredentialsOnCrossHostRedirect(t *testing.T) {
	tests := []struct {
		name   string
		option ClientOption
	}{
		{
			name:   "access token",
			option: WithAccessToken("secret-access-token"),
		},
		{
			name:   "service token",
			option: WithServiceToken("tid", "secret-service-token"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			attackerHits := 0
			attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attackerHits++
				c.Assert(r.Header.Get("Authorization"), qt.Equals, "")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			t.Cleanup(attacker.Close)

			// httptest serves on 127.0.0.1; rewrite the Location host to
			// localhost so Hostname() differs while still reaching this
			// process (same port).
			attackerPort := attacker.Listener.Addr().(*net.TCPAddr).Port
			attackerURL := fmt.Sprintf("http://localhost:%d/harvest", attackerPort)

			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Assert(r.Header.Get("Authorization"), qt.Not(qt.Equals), "")
				http.Redirect(w, r, attackerURL, http.StatusFound)
			}))
			t.Cleanup(api.Close)

			client, err := NewClient(WithBaseURL(api.URL), tt.option)
			c.Assert(err, qt.IsNil)

			req, err := client.newRequest(http.MethodGet, "/v1/organizations", nil)
			c.Assert(err, qt.IsNil)

			// Cross-host redirect must stop before RoundTrip attaches auth to
			// the attacker host. ErrUseLastResponse returns the 302 with no error.
			res, err := client.client.Do(req)
			c.Assert(err, qt.IsNil)
			defer res.Body.Close()

			c.Assert(res.StatusCode, qt.Equals, http.StatusFound)
			c.Assert(attackerHits, qt.Equals, 0)
		})
	}
}

func TestClient_FollowsSameHostRedirectWithCredentials(t *testing.T) {
	c := qt.New(t)

	var paths []string
	var auths []string
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/v1/start", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		auths = append(auths, r.Header.Get("Authorization"))
		http.Redirect(w, r, server.URL+"/v1/final", http.StatusFound)
	})
	mux.HandleFunc("/v1/final", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		auths = append(auths, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"ok"}`))
	})

	client, err := NewClient(WithBaseURL(server.URL), WithServiceToken("tid", "secret"))
	c.Assert(err, qt.IsNil)

	var got map[string]string
	err = client.do(context.Background(), mustNewRequest(t, client, http.MethodGet, "/v1/start"), &got)
	c.Assert(err, qt.IsNil)
	c.Assert(got["name"], qt.Equals, "ok")
	c.Assert(paths, qt.DeepEquals, []string{"/v1/start", "/v1/final"})
	c.Assert(auths, qt.DeepEquals, []string{"tid:secret", "tid:secret"})
}

func mustNewRequest(t *testing.T, client *Client, method, path string) *http.Request {
	t.Helper()
	req, err := client.newRequest(method, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func Pointer[K any](val K) *K {
	return &val
}
