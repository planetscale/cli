package d1

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

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
