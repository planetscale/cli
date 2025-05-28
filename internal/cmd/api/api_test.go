package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseField(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		init   map[string]interface{}
		want   map[string]interface{}
	}{
		{
			name: "fields only",
			fields: []string{
				"hello.world=1",
				"hello.monde=2",
				`salut="le monde"`,
			},
			want: map[string]interface{}{
				"hello": map[string]interface{}{
					"world": float64(1),
					"monde": float64(2),
				},
				"salut": "le monde",
			},
		},
		{
			name: "update from fields",
			init: map[string]interface{}{
				"hello": map[string]interface{}{
					"monde": float64(42),
				},
				"salut": "fred",
				"bye":   "ivon",
			},
			fields: []string{
				"hello.world=1",
				"hello.monde=2",
				`salut="le monde"`,
			},
			want: map[string]interface{}{
				"hello": map[string]interface{}{
					"world": float64(1),
					"monde": float64(2),
				},
				"salut": "le monde",
				"bye":   "ivon",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := parseFields(tt.init, tt.fields)
			require.NoError(t, err)
			require.Equal(t, tt.want, out)
		})
	}
}

func TestExtractRootDomain(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{
			name: "simple domain",
			host: "example.com",
			want: "example.com",
		},
		{
			name: "subdomain",
			host: "api.example.com",
			want: "example.com",
		},
		{
			name: "multiple subdomains",
			host: "v1.api.example.com",
			want: "example.com",
		},
		{
			name: "with port",
			host: "example.com:8080",
			want: "example.com",
		},
		{
			name: "subdomain with port",
			host: "api.example.com:8080",
			want: "example.com",
		},
		{
			name: "localhost",
			host: "localhost",
			want: "localhost",
		},
		{
			name: "localhost with port",
			host: "localhost:8080",
			want: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRootDomain(tt.host)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRedirectCheck(t *testing.T) {
	tests := []struct {
		name                  string
		originalHost          string
		redirectHost          string
		expectUseLastResponse bool
	}{
		{
			name:                  "same domain",
			originalHost:          "api.example.com",
			redirectHost:          "www.example.com",
			expectUseLastResponse: false,
		},
		{
			name:                  "different domain",
			originalHost:          "api.example.com",
			redirectHost:          "api.another.com",
			expectUseLastResponse: true,
		},
		{
			name:                  "localhost to domain",
			originalHost:          "localhost:8080",
			redirectHost:          "example.com",
			expectUseLastResponse: true,
		},
		{
			name:                  "domain to localhost",
			originalHost:          "example.com",
			redirectHost:          "localhost:8080",
			expectUseLastResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalDomain := extractRootDomain(tt.originalHost)
			redirectCheck := makeRedirectCheck(originalDomain)

			// Create a test request simulating a redirect
			req, _ := http.NewRequest("GET", "https://"+tt.redirectHost+"/path", nil)

			// Run the redirect check
			err := redirectCheck(req, []*http.Request{})

			// Check the result
			if tt.expectUseLastResponse {
				require.Equal(t, http.ErrUseLastResponse, err,
					"Expected ErrUseLastResponse for cross-domain redirect")
			} else {
				require.NoError(t, err,
					"Expected nil error for same-domain redirect")
			}
		})
	}
}

func TestHandleRedirect(t *testing.T) {
	// Create a test context
	ctx := context.Background()

	// Create an original request with auth header
	originalReq, _ := http.NewRequest("GET", "https://api.example.com/path", nil)
	originalReq.Header.Set("Authorization", "Bearer token123")
	originalReq.Header.Set("User-Agent", "test-agent")

	// Create a mock original response
	originalRes := &http.Response{
		StatusCode: 302,
		Header:     http.Header{},
	}
	originalRes.Header.Set("Location", "https://other-domain.com/newpath")

	// Mock a response from the redirect target
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ensure auth header is not present
		if r.Header.Get("Authorization") != "" {
			t.Error("Auth header was incorrectly passed to redirect target")
		}

		// Ensure other headers were preserved
		if r.Header.Get("User-Agent") != "test-agent" {
			t.Error("Other headers were not preserved in redirect")
		}

		w.Write([]byte("Redirect target content"))
	}))
	defer mockServer.Close()

	// Test the handleRedirect function with the mock server
	redirectRes, err := handleRedirect(ctx, originalReq, originalRes, mockServer.URL, false)

	// Verify the result
	require.NoError(t, err)
	require.NotNil(t, redirectRes)

	// Read the response body
	body, err := io.ReadAll(redirectRes.Body)
	require.NoError(t, err)
	redirectRes.Body.Close()

	// Verify the response content
	require.Equal(t, "Redirect target content", string(body))
}
