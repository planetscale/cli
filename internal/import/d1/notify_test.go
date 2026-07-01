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

func TestShouldNotifyProgressMajorStages(t *testing.T) {
	for _, stage := range []string{
		ImportStageConnecting,
		ImportStageSQLiteStaging,
		ImportStageSchema,
		ImportStageIndexes,
		ImportStageSequences,
	} {
		if !shouldNotifyProgress(ImportProgress{Stage: stage}) {
			t.Fatalf("expected stage %q to notify", stage)
		}
	}
}

func TestShouldNotifyProgressPgloaderTables(t *testing.T) {
	if !shouldNotifyProgress(ImportProgress{Stage: ImportStagePgloader, Current: 1, Total: 19, Detail: "users"}) {
		t.Fatal("expected first pgloader table to notify")
	}
	for _, current := range []int{0, 2, 19} {
		if shouldNotifyProgress(ImportProgress{Stage: ImportStagePgloader, Current: current, Total: 19, Detail: "users"}) {
			t.Fatalf("expected pgloader table %d to skip slack notification", current)
		}
	}
}

func TestShouldNotifyProgressRowCounts(t *testing.T) {
	if !shouldNotifyProgress(ImportProgress{Stage: VerifyStageRowCounts, Total: 19}) {
		t.Fatal("expected row count stage start to notify")
	}
	for _, current := range []int{1, 2, 19} {
		if shouldNotifyProgress(ImportProgress{Stage: VerifyStageRowCounts, Current: current, Total: 19, Detail: "users (sqlite)"}) {
			t.Fatalf("expected row count progress %d to skip slack notification", current)
		}
	}
}

func TestFormatNotifyProgressMessageAggregates(t *testing.T) {
	got := formatNotifyProgressMessage(ImportProgress{Stage: ImportStagePgloader, Current: 1, Total: 19, Detail: "users"})
	want := "Loading tables... (19 tables)"
	if got != want {
		t.Fatalf("pgloader message = %q, want %q", got, want)
	}
	got = formatNotifyProgressMessage(ImportProgress{Stage: VerifyStageRowCounts, Total: 19})
	want = "Comparing row counts... (19 tables)"
	if got != want {
		t.Fatalf("row counts message = %q, want %q", got, want)
	}
}

func TestFormatProgressMessageSQLiteStaging(t *testing.T) {
	got := FormatProgressMessage(ImportProgress{Stage: ImportStageSQLiteStaging})
	want := "Staging SQLite database from export..."
	if got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}

func TestShouldNotifyProgressUnknownStage(t *testing.T) {
	if shouldNotifyProgress(ImportProgress{Stage: "custom_stage", Current: 1, Detail: "working"}) {
		t.Fatal("expected unknown stage to skip slack notification")
	}
}

func TestImportProgressPgloaderUsesReportProgress(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	opts := ImportOptions{
		Org:         "org",
		Database:    "db",
		Branch:      "main",
		MigrationID: "abc123",
		Method:      MethodPgloader,
		NotifyAPI: NotifyAPIConfig{
			Client: testNotifyClient(t, srv.URL),
		},
		OnProgress: func(p ImportProgress) {
			if p.Stage != ImportStagePgloader {
				t.Fatalf("OnProgress stage = %q, want %q", p.Stage, ImportStagePgloader)
			}
		},
	}

	opts.reportProgress(ImportProgress{
		Stage:   ImportStagePgloader,
		Current: 1,
		Total:   3,
		Detail:  "users",
	})

	deadline := time.After(2 * time.Second)
	for calls.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("expected pgloader progress Slack notification")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
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
