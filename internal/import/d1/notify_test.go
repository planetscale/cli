package d1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

func testNotifyClient(t *testing.T, baseURL string) *ps.Client {
	t.Helper()

	client, err := ps.NewClient(
		ps.WithBaseURL(baseURL),
		ps.WithAccessToken("token"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestNotifyImportEventSync_CompletesBeforeReturn(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	start := time.Now()
	NotifyImportEventSync(NotifyAPIConfig{
		Client: testNotifyClient(t, srv.URL),
	}, "org", "db", "main", "abc123", NotifyEventFailed, importNotificationPayload{
		Error: "boom",
	})
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("NotifyImportEventSync returned in %v, expected to wait for request", elapsed)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestNotifyImportEvent_FireAndForget(t *testing.T) {
	var calls atomic.Int32
	done := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		done <- struct{}{}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	NotifyImportEvent(NotifyAPIConfig{
		Client: testNotifyClient(t, srv.URL),
	}, "org", "db", "main", "abc123", "start", importNotificationPayload{
		Method: "pgloader",
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected notification request")
	}

	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestNotifyImportEvent_SkipsWhenDisabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected notification request")
	}))
	defer srv.Close()

	NotifyImportEvent(NotifyAPIConfig{
		Client:   testNotifyClient(t, srv.URL),
		Disabled: true,
	}, "org", "db", "main", "abc123", "start", importNotificationPayload{})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	<-ctx.Done()
}

func TestNotifyImportEvent_SkipsWithoutClient(t *testing.T) {
	NotifyImportEvent(NotifyAPIConfig{}, "org", "db", "main", "abc123", "start", importNotificationPayload{})
}

func TestNotifyImportEvent_DoesNotFailImportOnAPIError(t *testing.T) {
	done := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		done <- struct{}{}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	NotifyImportEvent(NotifyAPIConfig{
		Client: testNotifyClient(t, srv.URL),
	}, "org", "db", "main", "abc123", "failed", importNotificationPayload{
		Error: "boom",
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected notification request")
	}
}

func TestPostImportNotification_ProgressSendsStageAndMessage(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("unmarshal body: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	err := postImportNotification(context.Background(), NotifyAPIConfig{
		Client: testNotifyClient(t, srv.URL),
	}, "org", "db", importNotificationPayload{
		MigrationID: "abc123",
		Event:       NotifyEventProgress,
		Method:      "pgloader",
		Stage:       ImportStageSQLiteStaging,
		Message:     "Staging SQLite database from export...",
	})
	if err != nil {
		t.Fatalf("postImportNotification: %v", err)
	}
	if body["event"] != "progress" {
		t.Fatalf("event = %v, want progress", body["event"])
	}
	if body["method"] != "pgloader" {
		t.Fatalf("method = %v, want pgloader", body["method"])
	}
	if body["stage"] != ImportStageSQLiteStaging {
		t.Fatalf("stage = %v, want %s", body["stage"], ImportStageSQLiteStaging)
	}
	if body["message"] != "Staging SQLite database from export..." {
		t.Fatalf("message = %v", body["message"])
	}
	if _, ok := body["error"]; ok {
		t.Fatalf("error = %v, want omitted for progress status text", body["error"])
	}
}

func TestPostImportNotification_UsesInternalRoute(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	err := postImportNotification(context.Background(), NotifyAPIConfig{
		Client: testNotifyClient(t, srv.URL),
	}, "org", "db", importNotificationPayload{
		MigrationID: "abc123",
		Event:       "start",
	})
	if err != nil {
		t.Fatalf("postImportNotification: %v", err)
	}
	if path != "/internal/organizations/org/databases/db/d1-import-notifications" {
		t.Fatalf("path = %q", path)
	}
}

func TestFormatNotifyError_MigrationError(t *testing.T) {
	err := fmt.Errorf("pgloader table team_members: %w", newMigrationError(
		ErrCodeImportFailed,
		`pgloader copied 0 rows into "team_members" (expected 700 from dump)`,
		pgloaderNoRowsRemediation,
	))

	got := formatNotifyError(err, nil)
	want := "[IMPORT_FAILED] pgloader copied 0 rows into \"team_members\" (expected 700 from dump)\n" + pgloaderNoRowsRemediation
	if got != want {
		t.Fatalf("formatNotifyError() = %q, want %q", got, want)
	}
}

func TestVerifyFailureSummary(t *testing.T) {
	summary := verifyFailureSummary(&VerifyResult{
		Tables: []TableVerifyResult{
			{Table: "team_members", SourceRows: 700, DestRows: 0, Match: false},
			{Table: "organizations", SourceRows: 28, DestRows: 28, Match: true},
		},
	})
	if summary != "team_members: sqlite=700 postgres=0" {
		t.Fatalf("summary = %q", summary)
	}
}

func TestNotifyImportFailure_SendsFailedEvent(t *testing.T) {
	done := make(chan struct{}, 1)
	var body map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { done <- struct{}{} }()
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("unmarshal body: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	notifyImportFailure(NotifyAPIConfig{
		Client: testNotifyClient(t, srv.URL),
	}, "org", "db", "main", "abc123", importNotificationPayload{}, newMigrationError(
		ErrCodeImportFailed,
		"pgloader matched 0 source tables",
		pgloaderNoRowsRemediation,
	), nil)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected notification request")
	}

	if body["event"] != "failed" {
		t.Fatalf("event = %v, want failed", body["event"])
	}
	if !strings.Contains(body["error"].(string), "[IMPORT_FAILED]") {
		t.Fatalf("error = %v, want IMPORT_FAILED prefix", body["error"])
	}
}
