package planetscale

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogsServiceCreatesSignatureAndQueriesSignedURLWithoutCredentials(t *testing.T) {
	var serverURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/organizations/acme/databases/db/branches/main/logs/signatures", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("signature method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer api-token" {
			t.Fatalf("signature authorization = %q, want Bearer api-token", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"sig": "signed",
			"exp": "12345",
			"url": serverURL + "/signed?sig=signed&exp=12345",
		})
	})
	mux.HandleFunc("/signed", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("signed logs request leaked authorization header %q", got)
		}
		if got := r.URL.Query().Get("sig"); got != "signed" {
			t.Fatalf("sig = %q, want signed", got)
		}
		if got := r.URL.Query().Get("exp"); got != "12345" {
			t.Fatalf("exp = %q, want 12345", got)
		}
		if got := r.URL.Query().Get("limit"); got != "25" {
			t.Fatalf("limit = %q, want 25", got)
		}
		if got := r.URL.Query().Get("query"); got != "* _time:1h" {
			t.Fatalf("query = %q, want %q", got, "* _time:1h")
		}
		_, _ = io.WriteString(w, "log response")
	})

	server := httptest.NewServer(mux)
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := NewClient(WithBaseURL(server.URL), WithAccessToken("api-token"))
	if err != nil {
		t.Fatal(err)
	}

	signature, err := client.Logs.CreateSignature(context.Background(), &CreateLogSignatureRequest{
		Organization: "acme",
		Database:     "db",
		Branch:       "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if signature.Signature != "signed" || signature.ExpiresAt != "12345" {
		t.Fatalf("unexpected signature: %#v", signature)
	}

	body, err := client.Logs.Query(context.Background(), &QueryLogsRequest{
		URL:   signature.URL,
		Query: "* _time:1h",
		Limit: 25,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()

	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "log response" {
		t.Fatalf("body = %q, want log response", got)
	}
}

func TestLogsServiceQueryErrorIncludesStatusAndShortBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "logs temporarily unavailable")
	}))
	t.Cleanup(server.Close)

	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Logs.Query(context.Background(), &QueryLogsRequest{URL: server.URL, Query: "*", Limit: 100})
	if err == nil {
		t.Fatal("expected query error")
	}
	want := "HTTP 502 Bad Gateway: logs temporarily unavailable"
	if got := err.Error(); got != "querying branch logs: "+want {
		t.Fatalf("error = %q, want %q", got, "querying branch logs: "+want)
	}
}
