package auditlog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func authAttemptTestHelper(org, baseURL string, format printer.Format, buf *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)
	p.SetHumanOutput(io.Discard)

	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{AccessToken: "token", Organization: org, BaseURL: baseURL},
		Client: func() (*ps.Client, error) {
			return ps.NewClient(ps.WithBaseURL(baseURL), ps.WithAccessToken("token"))
		},
	}
}

func TestAuthAttemptsDownloadCreatesPollsAndWritesFile(t *testing.T) {
	c := qt.New(t)

	previousInterval := authAttemptExportPollInterval
	authAttemptExportPollInterval = 5 * time.Second
	t.Cleanup(func() { authAttemptExportPollInterval = previousInterval })

	var createRequests, statusRequests, downloadRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/organizations/my-org/auth-attempt-exports":
			createRequests++
			var body struct {
				StartAt string `json:"start_at"`
				EndAt   string `json:"end_at"`
				Format  string `json:"format"`
				Filters struct {
					SourceIPs        []string `json:"source_ips"`
					Branches         []string `json:"branches"`
					Outcomes         []string `json:"outcomes"`
					Usernames        []string `json:"usernames"`
					StartupDatabases []string `json:"startup_databases"`
					FailureReasons   []string `json:"failure_reasons"`
					BackendRoutes    []string `json:"backend_routes"`
				} `json:"filters"`
			}
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body.StartAt, qt.Equals, "2026-07-29T00:00:00Z")
			c.Assert(body.EndAt, qt.Equals, "2026-07-29T01:00:00Z")
			c.Assert(body.Format, qt.Equals, "jsonl")
			c.Assert(body.Filters.SourceIPs, qt.DeepEquals, []string{"203.0.113.0/24", "2001:db8::/112"})
			c.Assert(body.Filters.Branches, qt.DeepEquals, []string{"db/production"})
			c.Assert(body.Filters.Outcomes, qt.DeepEquals, []string{"deny"})
			c.Assert(body.Filters.Usernames, qt.DeepEquals, []string{"incident-user"})
			c.Assert(body.Filters.StartupDatabases, qt.DeepEquals, []string{""})
			c.Assert(body.Filters.FailureReasons, qt.DeepEquals, []string{"bad_password"})
			c.Assert(body.Filters.BackendRoutes, qt.DeepEquals, []string{"postgres"})
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"export1","state":"pending","format":"jsonl"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/my-org/auth-attempt-exports/export1":
			statusRequests++
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"export1","state":"ready","start_at":"2026-07-29T00:00:00Z","end_at":"2026-07-29T01:00:00Z","format":"jsonl"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/my-org/auth-attempt-exports/export1/download":
			downloadRequests++
			http.Redirect(w, r, "/blob", http.StatusFound)
		case r.Method == http.MethodGet && r.URL.Path == "/blob":
			_, _ = io.WriteString(w, "PK\x03\x04zip")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	output := filepath.Join(t.TempDir(), "auth-attempts.zip")
	var printed bytes.Buffer
	cmd := AuthAttemptsCmd(authAttemptTestHelper("my-org", server.URL, printer.JSON, &printed))
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"download",
		"--start-at", "2026-07-29T00:00:00Z",
		"--end-at", "2026-07-29T01:00:00Z",
		"--source-ip", "203.0.113.0/24,2001:db8::/112",
		"--branch", "db/production",
		"--outcome", "deny",
		"--username", "incident-user",
		"--startup-database", "",
		"--failure-reason", "bad_password",
		"--backend-route", "postgres",
		"--output", output,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)
	c.Assert(cmd.ExecuteContext(ctx), qt.IsNil)
	c.Assert(createRequests, qt.Equals, 1)
	c.Assert(statusRequests, qt.Equals, 1)
	c.Assert(downloadRequests, qt.Equals, 1)

	downloaded, err := os.ReadFile(output)
	c.Assert(err, qt.IsNil)
	c.Assert(downloaded, qt.DeepEquals, []byte("PK\x03\x04zip"))
	c.Assert(printed.String(), qt.JSONEquals, &AuthAttemptExportDownload{
		ID: "export1", State: "ready", Format: "jsonl",
		StartAt: "2026-07-29T00:00:00Z", EndAt: "2026-07-29T01:00:00Z", File: output,
	})
}

func TestAuthAttemptsDownloadRefusesStdoutToTerminalBeforeCreatingExport(t *testing.T) {
	c := qt.New(t)
	previousTTY := printer.IsTTY
	printer.IsTTY = true
	t.Cleanup(func() { printer.IsTTY = previousTTY })

	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)

	cmd := AuthAttemptsCmd(authAttemptTestHelper("my-org", server.URL, printer.JSON, &bytes.Buffer{}))
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"download",
		"--start-at", "2026-07-29T00:00:00Z",
		"--end-at", "2026-07-29T01:00:00Z",
		"--output", "-",
	})

	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, `^cannot write raw ZIP bytes to an interactive terminal; redirect stdout \(for example, --output - > report\.zip\) or use --output <file>$`)
	c.Assert(requests, qt.Equals, 0)
}

func TestAuthAttemptsDownloadWritesRawZIPToRedirectedStdout(t *testing.T) {
	c := qt.New(t)
	previousTTY := printer.IsTTY
	printer.IsTTY = false
	t.Cleanup(func() { printer.IsTTY = previousTTY })

	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"export1","state":"ready","format":"jsonl","start_at":"2026-07-29T00:00:00Z","end_at":"2026-07-29T01:00:00Z"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/my-org/auth-attempt-exports/export1/download":
			_, _ = io.WriteString(w, "PK\x03\x04zip")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	cmd := AuthAttemptsCmd(authAttemptTestHelper("my-org", server.URL, printer.Human, &bytes.Buffer{}))
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"download",
		"--start-at", "2026-07-29T00:00:00Z",
		"--end-at", "2026-07-29T01:00:00Z",
		"--output", "-",
	})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(requests, qt.Equals, 2)
	c.Assert(stdout.Bytes(), qt.DeepEquals, []byte("PK\x03\x04zip"))
}

func TestAuthAttemptsDownloadReportsExpiredExport(t *testing.T) {
	c := qt.New(t)

	previousInterval := authAttemptExportPollInterval
	authAttemptExportPollInterval = time.Millisecond
	t.Cleanup(func() { authAttemptExportPollInterval = previousInterval })

	for _, test := range []struct {
		name         string
		createdState string
		expectedPath string
	}{
		{name: "while polling", createdState: "pending", expectedPath: "/v1/organizations/my-org/auth-attempt-exports/export1"},
		{name: "while downloading", createdState: "ready", expectedPath: "/v1/organizations/my-org/auth-attempt-exports/export1/download"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost:
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, fmt.Sprintf(`{"id":"export1","state":"%s","format":"jsonl"}`, test.createdState))
				case r.Method == http.MethodGet && r.URL.Path == test.expectedPath:
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusGone)
					_, _ = io.WriteString(w, `{"id":"export1","state":"ready","expired":true,"recovery_hint":"Create a new export with the original time range."}`)
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)

			var printed bytes.Buffer
			cmd := AuthAttemptsCmd(authAttemptTestHelper("my-org", server.URL, printer.JSON, &printed))
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{
				"download",
				"--start-at", "2026-07-29T00:00:00Z",
				"--end-at", "2026-07-29T01:00:00Z",
				"--output", filepath.Join(t.TempDir(), "auth-attempts.zip"),
			})

			err := cmd.Execute()
			c.Assert(err, qt.ErrorMatches, `^auth attempt export export1 expired: Create a new export with the original time range\.$`)
		})
	}
}

func TestAuthAttemptsDownloadHonorsContextCancellation(t *testing.T) {
	for _, test := range []struct {
		name string
		ctx  func() (context.Context, context.CancelFunc)
		want error
	}{
		{name: "canceled", ctx: func() (context.Context, context.CancelFunc) {
			return context.WithCancel(context.Background())
		}, want: context.Canceled},
		{name: "deadline exceeded", ctx: func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), 20*time.Millisecond)
		}, want: context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := qt.New(t)
			previousInterval := authAttemptExportPollInterval
			authAttemptExportPollInterval = time.Hour
			t.Cleanup(func() { authAttemptExportPollInterval = previousInterval })

			created := make(chan struct{})
			var downloads int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost:
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"id":"export1","state":"pending","format":"jsonl"}`)
					close(created)
				case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/my-org/auth-attempt-exports/export1/download":
					downloads++
					http.Error(w, "unexpected download", http.StatusInternalServerError)
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)

			ctx, cancel := test.ctx()
			t.Cleanup(cancel)
			var printed bytes.Buffer
			cmd := AuthAttemptsCmd(authAttemptTestHelper("my-org", server.URL, printer.JSON, &printed))
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{
				"download",
				"--start-at", "2026-07-29T00:00:00Z",
				"--end-at", "2026-07-29T01:00:00Z",
				"--output", filepath.Join(t.TempDir(), "auth-attempts.zip"),
			})

			result := make(chan error, 1)
			go func() { result <- cmd.ExecuteContext(ctx) }()
			select {
			case <-created:
			case <-time.After(time.Second):
				t.Fatal("export was not created")
			}
			if test.name == "canceled" {
				time.Sleep(50 * time.Millisecond)
				cancel()
			}
			err := <-result
			c.Assert(errors.Is(err, test.want), qt.IsTrue)
			c.Assert(err, qt.ErrorMatches, `.*export1.*`)
			c.Assert(err, qt.ErrorMatches, `.*pscale api organizations/my-org/auth-attempt-exports/export1.*`)
			c.Assert(downloads, qt.Equals, 0)
		})
	}
}

func TestAuthAttemptsDownloadFilterPresence(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAbsent []string
		want       map[string][]string
	}{
		{name: "absent", wantAbsent: []string{
			"source_ips", "branches", "outcomes", "usernames", "startup_databases", "failure_reasons", "backend_routes",
		}},
		{name: "empty startup database", args: []string{"--startup-database", ""}, want: map[string][]string{"startup_databases": {""}}},
		{name: "startup database order", args: []string{"--startup-database", "", "--startup-database", "postgres"}, want: map[string][]string{"startup_databases": {"", "postgres"}}},
		{name: "username commas stay together", args: []string{"--username", "a,b"}, want: map[string][]string{"usernames": {"a,b"}}},
		{name: "source and enum comma values", args: []string{"--source-ip", "203.0.113.0/24,2001:db8::/112", "--outcome", "allow,deny", "--failure-reason", "bad_password,other", "--backend-route", "postgres,unknown"}, want: map[string][]string{
			"source_ips": {"203.0.113.0/24", "2001:db8::/112"}, "outcomes": {"allow", "deny"}, "failure_reasons": {"bad_password", "other"}, "backend_routes": {"postgres", "unknown"},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := qt.New(t)
			var posted map[string]json.RawMessage
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					var body struct {
						Filters map[string]json.RawMessage `json:"filters"`
					}
					c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
					posted = body.Filters
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"id":"export1","state":"ready","format":"jsonl","start_at":"2026-07-29T00:00:00Z","end_at":"2026-07-29T01:00:00Z"}`)
					return
				}
				if r.URL.Path == "/v1/organizations/my-org/auth-attempt-exports/export1/download" {
					http.Redirect(w, r, "/blob", http.StatusFound)
					return
				}
				if r.URL.Path == "/blob" {
					_, _ = io.WriteString(w, "PK\x03\x04zip")
					return
				}
				http.NotFound(w, r)
			}))
			t.Cleanup(server.Close)

			args := append([]string{"download", "--start-at", "2026-07-29T00:00:00Z", "--end-at", "2026-07-29T01:00:00Z", "--output", filepath.Join(t.TempDir(), "auth-attempts.zip")}, test.args...)
			var printed bytes.Buffer
			cmd := AuthAttemptsCmd(authAttemptTestHelper("my-org", server.URL, printer.JSON, &printed))
			cmd.SetArgs(args)
			c.Assert(cmd.Execute(), qt.IsNil)

			for key, values := range test.want {
				var got []string
				c.Assert(json.Unmarshal(posted[key], &got), qt.IsNil)
				c.Assert(got, qt.DeepEquals, values)
			}
			for _, key := range test.wantAbsent {
				_, ok := posted[key]
				c.Assert(ok, qt.IsFalse, qt.Commentf("filter %s was sent", key))
			}
		})
	}
}

func TestAuthAttemptsDownloadInterruptedCopyDoesNotPublishPartialFile(t *testing.T) {
	c := qt.New(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"export1","state":"ready","format":"jsonl"}`)
		case strings.HasSuffix(r.URL.Path, "/download"):
			http.Redirect(w, r, "/blob", http.StatusFound)
		case r.URL.Path == "/blob":
			_, _ = io.WriteString(w, "partial")
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			<-r.Context().Done()
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	output := filepath.Join(t.TempDir(), "auth-attempts.zip")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	t.Cleanup(cancel)
	cmd := AuthAttemptsCmd(authAttemptTestHelper("my-org", server.URL, printer.JSON, &bytes.Buffer{}))
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"download", "--start-at", "2026-07-29T00:00:00Z", "--end-at", "2026-07-29T01:00:00Z", "--output", output})
	c.Assert(cmd.ExecuteContext(ctx), qt.IsNotNil)
	_, err := os.Stat(output)
	c.Assert(err, qt.ErrorIs, os.ErrNotExist)
	entries, err := os.ReadDir(filepath.Dir(output))
	c.Assert(err, qt.IsNil)
	c.Assert(len(entries), qt.Equals, 0)
}

func TestAuthAttemptsDownloadRetriesFailedFileDownloadWithFreshSignedURL(t *testing.T) {
	c := qt.New(t)
	var downloadRequests, blobRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"export1","state":"ready","format":"jsonl"}`)
		case strings.HasSuffix(r.URL.Path, "/download"):
			downloadRequests++
			http.Redirect(w, r, fmt.Sprintf("/blob/%d", downloadRequests), http.StatusFound)
		case strings.HasPrefix(r.URL.Path, "/blob/"):
			blobRequests++
			if blobRequests == 1 {
				connection, writer, err := w.(http.Hijacker).Hijack()
				c.Assert(err, qt.IsNil)
				_, _ = writer.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 10\r\n\r\npartial")
				c.Assert(writer.Flush(), qt.IsNil)
				_ = connection.Close()
				return
			}
			_, _ = io.WriteString(w, "complete")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	output := filepath.Join(t.TempDir(), "auth-attempts.zip")
	cmd := AuthAttemptsCmd(authAttemptTestHelper("my-org", server.URL, printer.JSON, &bytes.Buffer{}))
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"download", "--start-at", "2026-07-29T00:00:00Z", "--end-at", "2026-07-29T01:00:00Z", "--output", output})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(downloadRequests, qt.Equals, 2)
	c.Assert(blobRequests, qt.Equals, 2)
	content, err := os.ReadFile(output)
	c.Assert(err, qt.IsNil)
	c.Assert(content, qt.DeepEquals, []byte("complete"))
}

func TestAuthAttemptsDownloadDoesNotRetryStdoutAfterPartialOutput(t *testing.T) {
	c := qt.New(t)
	var downloadRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"export1","state":"ready","format":"jsonl"}`)
		case strings.HasSuffix(r.URL.Path, "/download"):
			downloadRequests++
			connection, writer, err := w.(http.Hijacker).Hijack()
			c.Assert(err, qt.IsNil)
			_, _ = writer.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 10\r\n\r\npartial")
			c.Assert(writer.Flush(), qt.IsNil)
			_ = connection.Close()
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	cmd := AuthAttemptsCmd(authAttemptTestHelper("my-org", server.URL, printer.JSON, &bytes.Buffer{}))
	cmd.SilenceUsage = true
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"download", "--start-at", "2026-07-29T00:00:00Z", "--end-at", "2026-07-29T01:00:00Z", "--output", "-"})

	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, `(?s).*export1.*pscale api organizations/my-org/auth-attempt-exports/export1/download > report\.zip.*`)
	c.Assert(downloadRequests, qt.Equals, 1)
	c.Assert(stdout.Bytes(), qt.DeepEquals, []byte("partial"))
}

func TestAuthAttemptsDownloadKeepsRecoveryGuidanceAfterRetryFailure(t *testing.T) {
	c := qt.New(t)
	var downloadRequests int
	blob := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(blob.Close)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"export1","state":"ready","format":"jsonl"}`)
		case strings.HasSuffix(r.URL.Path, "/download"):
			downloadRequests++
			if downloadRequests == 1 {
				http.Redirect(w, r, "http://127.0.0.1:1/signed?X-Amz-Signature=secret", http.StatusFound)
				return
			}
			http.Redirect(w, r, blob.URL, http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	cmd := AuthAttemptsCmd(authAttemptTestHelper("my-org", server.URL, printer.JSON, &bytes.Buffer{}))
	cmd.SilenceUsage = true
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"download", "--start-at", "2026-07-29T00:00:00Z", "--end-at", "2026-07-29T01:00:00Z", "--output", filepath.Join(t.TempDir(), "auth-attempts.zip")})

	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, `(?s).*export1.*signed download returned Service Unavailable.*pscale api organizations/my-org/auth-attempt-exports/export1/download > report\.zip.*`)
	c.Assert(err.Error(), qt.Not(qt.Contains), "X-Amz-Signature")
	c.Assert(downloadRequests, qt.Equals, 2)
}
