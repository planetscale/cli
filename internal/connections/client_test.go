package connections

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const sampleListResponse = `{
  "type": "list",
  "next_page": null,
  "prev_page": null,
  "captured_at": "2026-04-29T12:34:56.789Z",
  "instances": [
    {"id": "primary", "role": "primary", "error": null},
    {"id": "replica-1", "role": "replica", "error": null}
  ],
  "data": [
    {
      "pid": 123,
      "instance": "primary",
      "captured_at": "2026-04-29T12:34:56.789Z",
      "duration_ms": 1234,
      "blocked_by": [456],
      "query_text": "SELECT 1 FROM t",
      "state": "active",
      "wait_event": "ClientRead",
      "wait_event_type": "Client",
      "client_addr": "10.0.0.1",
      "application_name": "psql",
      "backend_type": "client backend",
      "usename": "alice",
      "xact_start": "2026-04-29T12:34:00.000Z",
      "query_start": "2026-04-29T12:34:30.000Z",
      "connection_id": "primary-123-1779113716123456",
      "transaction_id": "primary-123-1777466040000000",
      "query_id": "primary-123-1777466070000000"
    },
    {
      "pid": 456,
      "instance": "primary",
      "captured_at": "2026-04-29T12:34:56.789Z",
      "duration_ms": 50,
      "blocked_by": [],
      "query_text": "",
      "state": "idle",
      "wait_event": "",
      "wait_event_type": "",
      "client_addr": "",
      "application_name": "",
      "backend_type": "client backend",
      "usename": "bob",
      "xact_start": null,
      "query_start": null,
      "transaction_id": null,
      "query_id": null
    }
  ]
}`

func TestClient_ListDecodesConnectionList(t *testing.T) {
	var gotPath, gotMethod, gotAuth, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, sampleListResponse)
	}))
	defer srv.Close()

	client, err := NewClient(ClientConfig{
		BaseURL:        srv.URL,
		Organization:   "acme",
		Database:       "prod",
		Branch:         "main",
		ServiceTokenID: "tid",
		ServiceToken:   "secret",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	list, err := client.List(context.Background(), SortByTransactionStart)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	wantPath := "/v1/organizations/acme/databases/prod/branches/main/connections"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotAuth != "tid:secret" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "tid:secret")
	}
	if gotUA != "pscale-cli" {
		t.Errorf("User-Agent = %q, want %q", gotUA, "pscale-cli")
	}

	wantCaptured, _ := time.Parse(time.RFC3339Nano, "2026-04-29T12:34:56.789Z")
	if !list.CapturedAt.Equal(wantCaptured) {
		t.Errorf("CapturedAt = %v, want %v", list.CapturedAt, wantCaptured)
	}
	if list.Sort != SortByTransactionStart {
		t.Errorf("Sort = %q, want %q", list.Sort, SortByTransactionStart)
	}
	if len(list.Connections) != 2 {
		t.Fatalf("len(Connections) = %d, want 2", len(list.Connections))
	}

	first := list.Connections[0]
	if first.PID != 123 {
		t.Errorf("first.PID = %d, want 123", first.PID)
	}
	if first.Instance != "primary" {
		t.Errorf("first.Instance = %q, want primary", first.Instance)
	}
	if first.Username != "alice" {
		t.Errorf("first.Username = %q, want alice", first.Username)
	}
	if first.State != "active" {
		t.Errorf("first.State = %q, want active", first.State)
	}
	if first.WaitEvent != "ClientRead" {
		t.Errorf("first.WaitEvent = %q, want ClientRead", first.WaitEvent)
	}
	if first.WaitEventType != "Client" {
		t.Errorf("first.WaitEventType = %q, want Client", first.WaitEventType)
	}
	if first.ApplicationName != "psql" {
		t.Errorf("first.ApplicationName = %q, want psql", first.ApplicationName)
	}
	if first.ClientAddr != "10.0.0.1" {
		t.Errorf("first.ClientAddr = %q, want 10.0.0.1", first.ClientAddr)
	}
	if first.BackendType != "client backend" {
		t.Errorf("first.BackendType = %q, want client backend", first.BackendType)
	}
	if first.QueryText != "SELECT 1 FROM t" {
		t.Errorf("first.QueryText = %q, want SELECT 1 FROM t", first.QueryText)
	}
	if first.Duration != 1234*time.Millisecond {
		t.Errorf("first.Duration = %v, want 1234ms", first.Duration)
	}
	if len(first.BlockedBy) != 1 || first.BlockedBy[0] != 456 {
		t.Errorf("first.BlockedBy = %v, want [456]", first.BlockedBy)
	}
	if first.XactStart == nil {
		t.Error("first.XactStart = nil, want non-nil")
	}
	if first.QueryStart == nil {
		t.Error("first.QueryStart = nil, want non-nil")
	}

	second := list.Connections[1]
	if second.Username != "bob" {
		t.Errorf("second.Username = %q, want bob", second.Username)
	}
	if len(second.BlockedBy) != 0 {
		t.Errorf("second.BlockedBy = %v, want empty", second.BlockedBy)
	}
	if second.XactStart != nil {
		t.Errorf("second.XactStart = %v, want nil", second.XactStart)
	}

	if len(list.Instances) != 2 {
		t.Fatalf("len(Instances) = %d, want 2", len(list.Instances))
	}
	if list.Instances[0] != (InstanceMeta{ID: "primary", Role: "primary"}) {
		t.Errorf("Instances[0] = %+v, want primary/primary/(no error)", list.Instances[0])
	}
	if list.Instances[1] != (InstanceMeta{ID: "replica-1", Role: "replica"}) {
		t.Errorf("Instances[1] = %+v, want replica-1/replica/(no error)", list.Instances[1])
	}

	if first.InstanceRole != "primary" {
		t.Errorf("first.InstanceRole = %q, want primary (joined from Instances metadata)", first.InstanceRole)
	}
	if second.InstanceRole != "primary" {
		t.Errorf("second.InstanceRole = %q, want primary (both rows on primary)", second.InstanceRole)
	}

	if first.TransactionID == nil || *first.TransactionID != "primary-123-1777466040000000" {
		got := "nil"
		if first.TransactionID != nil {
			got = *first.TransactionID
		}
		t.Errorf("first.TransactionID = %s, want primary-123-1777466040000000", got)
	}
	if first.QueryID == nil || *first.QueryID != "primary-123-1777466070000000" {
		got := "nil"
		if first.QueryID != nil {
			got = *first.QueryID
		}
		t.Errorf("first.QueryID = %s, want primary-123-1777466070000000", got)
	}
	if first.ConnectionID == nil || *first.ConnectionID != "primary-123-1779113716123456" {
		got := "nil"
		if first.ConnectionID != nil {
			got = *first.ConnectionID
		}
		t.Errorf("first.ConnectionID = %s, want primary-123-1779113716123456", got)
	}
	if second.TransactionID != nil {
		t.Errorf("second.TransactionID = %v, want nil (wire null)", *second.TransactionID)
	}
	if second.QueryID != nil {
		t.Errorf("second.QueryID = %v, want nil (wire null)", *second.QueryID)
	}
}

func TestClient_ListDecodesVitessProcesslist(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"type": "list",
			"database_kind": "mysql",
			"next_page": null,
			"prev_page": null,
			"captured_at": "2026-06-04T12:30:00.000Z",
			"instances": [],
			"topology": {
				"keyspace": "commerce",
				"shard": "-80",
				"tablet": "zone1-1001"
			},
			"data": [
				{
					"pid": 101,
					"instance": "zone1-1001",
					"connection_id": "101",
					"transaction_id": null,
					"query_id": "101",
					"leader_pid": null,
					"datname": "checkout",
					"state": "Query/executing",
					"usename": "vt_app",
					"wait_event": null,
					"client_addr": "10.0.0.12:54231",
					"wait_event_type": null,
					"application_name": "",
					"backend_type": "client backend",
					"query_start": null,
					"xact_start": null,
					"backend_start": null,
					"state_change": null,
					"duration_ms": 42000,
					"blocked_by": [],
					"query_text": "SELECT 1"
				}
			]
		}`)
	}))
	defer srv.Close()

	client, err := NewClient(ClientConfig{
		BaseURL:        srv.URL,
		Organization:   "acme",
		Database:       "shop",
		Branch:         "main",
		Keyspace:       "commerce",
		Shard:          "-80",
		ServiceTokenID: "tid",
		ServiceToken:   "secret",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	list, err := client.List(context.Background(), SortByDuration)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if gotPath != "/v1/organizations/acme/databases/shop/branches/main/connections" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "keyspace=commerce&shard=-80" {
		t.Errorf("query = %q, want keyspace=commerce&shard=-80", gotQuery)
	}
	wantCaptured, _ := time.Parse(time.RFC3339Nano, "2026-06-04T12:30:00.000Z")
	if !list.CapturedAt.Equal(wantCaptured) {
		t.Errorf("CapturedAt = %v, want %v", list.CapturedAt, wantCaptured)
	}
	if list.DatabaseKind != DatabaseKindMySQL {
		t.Errorf("DatabaseKind = %q, want %q", list.DatabaseKind, DatabaseKindMySQL)
	}
	if list.Topology == nil {
		t.Fatal("Topology = nil, want Vitess topology")
	}
	if list.Topology.Keyspace != "commerce" || list.Topology.Shard != "-80" || list.Topology.Tablet != "zone1-1001" {
		t.Errorf("Topology = %+v", list.Topology)
	}
	if len(list.Connections) != 1 {
		t.Fatalf("len(Connections) = %d, want 1", len(list.Connections))
	}
	conn := list.Connections[0]
	if conn.PID != 101 || conn.Instance != "zone1-1001" || conn.Username != "vt_app" || conn.DatabaseName != "checkout" {
		t.Errorf("Connection = %+v", conn)
	}
	if conn.State != "Query/executing" {
		t.Errorf("State = %q, want Query/executing", conn.State)
	}
	if conn.Duration != 42*time.Second {
		t.Errorf("Duration = %v, want 42s", conn.Duration)
	}
	if conn.ConnectionID == nil || *conn.ConnectionID != "101" {
		t.Errorf("ConnectionID = %v, want 101", conn.ConnectionID)
	}
	if conn.QueryID == nil || *conn.QueryID != "101" {
		t.Errorf("QueryID = %v, want 101", conn.QueryID)
	}
}

func TestClient_ListLeavesInstanceRoleEmptyForUnknownInstance(t *testing.T) {
	const body = `{
	  "type": "list",
	  "captured_at": "2026-04-29T12:34:56.789Z",
	  "instances": [{"id": "primary", "role": "primary"}],
	  "data": [{"pid": 1, "instance": "ghost", "duration_ms": 0, "state": "idle"}]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	list, err := mustClient(t, srv.URL).List(context.Background(), SortByTransactionStart)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list.Connections[0].InstanceRole != "" {
		t.Errorf("InstanceRole = %q, want empty for unknown instance id", list.Connections[0].InstanceRole)
	}
}

func TestClient_ListLeavesInstanceRoleEmptyWithoutInstancesField(t *testing.T) {
	const body = `{
	  "type": "list",
	  "captured_at": "2026-04-29T12:34:56.789Z",
	  "data": [{"pid": 1, "instance": "primary", "duration_ms": 0, "state": "idle"}]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	list, err := mustClient(t, srv.URL).List(context.Background(), SortByTransactionStart)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list.Connections[0].InstanceRole != "" {
		t.Errorf("InstanceRole = %q, want empty when Instances is absent", list.Connections[0].InstanceRole)
	}
}

func TestClient_ListRetriesOn503AndReturnsList(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		call := calls.Add(1)
		if call < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, "cache lock held")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, sampleListResponse)
	}))
	defer srv.Close()

	c := newClientWithTimings(t, srv.URL, 500*time.Millisecond, 10*time.Millisecond, 0)
	list, err := c.List(context.Background(), SortByTransactionStart)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("calls = %d, want 3", got)
	}
	if len(list.Connections) != 2 {
		t.Fatalf("len(Connections) = %d, want 2", len(list.Connections))
	}
}

func TestClient_ListReturnsFriendlyWarmingMessageAfter503Budget(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, "cache lock held")
	}))
	defer srv.Close()

	c := newClientWithTimings(t, srv.URL, 35*time.Millisecond, 10*time.Millisecond, 0)
	start := time.Now()
	_, err := c.List(context.Background(), SortByTransactionStart)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("List: want warming error, got nil")
	}
	if got, want := err.Error(), "list connections: server is warming up, please retry in a moment"; got != want {
		t.Fatalf("err = %q, want %q", got, want)
	}
	if calls.Load() < 2 {
		t.Fatalf("calls = %d, want at least 2", calls.Load())
	}
	if elapsed > time.Second {
		t.Fatalf("List took %v, want retry budget to expire promptly", elapsed)
	}
}

func TestClient_ListDoesNotRetryOn500(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "internal host detail")
	}))
	defer srv.Close()

	c := newClientWithTimings(t, srv.URL, 500*time.Millisecond, 10*time.Millisecond, 0)
	_, err := c.List(context.Background(), SortByTransactionStart)
	if err == nil {
		t.Fatal("List: want error, got nil")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls = %d, want 1", got)
	}
	if strings.Contains(err.Error(), "internal host detail") {
		t.Fatalf("err = %q, must not surface 5xx body", err.Error())
	}
	if !strings.Contains(err.Error(), "HTTP 500: Internal Server Error") {
		t.Fatalf("err = %q, want HTTP 500 status text", err.Error())
	}
}

func TestClient_ListDoesNotRetryOn429(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, "slow down")
	}))
	defer srv.Close()

	c := newClientWithTimings(t, srv.URL, 500*time.Millisecond, 10*time.Millisecond, 0)
	_, err := c.List(context.Background(), SortByTransactionStart)
	if err == nil {
		t.Fatal("List: want error, got nil")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls = %d, want 1", got)
	}
	if got, want := err.Error(), "list connections: rate limited, please retry in a moment"; got != want {
		t.Fatalf("err = %q, want %q", got, want)
	}
}

func TestClient_ListReturnsStructuredAvailableTargets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{
			"message": "keyspace is required",
			"available": {
				"keyspaces": ["lookup", "main"],
				"shards": ["-80", "80-"]
			}
		}`)
	}))
	defer srv.Close()

	_, err := mustClient(t, srv.URL).List(context.Background(), SortByTransactionStart)
	if err == nil {
		t.Fatal("List: want error, got nil")
	}

	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("List error = %T %[1]v, want *HTTPError", err)
	}
	if httpErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", httpErr.StatusCode)
	}
	if httpErr.Message != "keyspace is required" {
		t.Errorf("Message = %q, want keyspace is required", httpErr.Message)
	}
	if !slices.Equal(httpErr.Available.Keyspaces, []string{"lookup", "main"}) {
		t.Errorf("Available.Keyspaces = %v, want [lookup main]", httpErr.Available.Keyspaces)
	}
	if !slices.Equal(httpErr.Available.Shards, []string{"-80", "80-"}) {
		t.Errorf("Available.Shards = %v, want [-80 80-]", httpErr.Available.Shards)
	}
	if got, want := err.Error(), "list connections: HTTP 400: keyspace is required (available keyspaces: lookup, main) (available shards: -80, 80-)"; got != want {
		t.Fatalf("err = %q, want %q", got, want)
	}
}

func TestClientListPartialResponseFriendlyMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"connections":[`) // deliberately truncated JSON
	}))
	defer srv.Close()

	c := newClientWithTimings(t, srv.URL, 500*time.Millisecond, 10*time.Millisecond, 0)
	_, err := c.List(context.Background(), SortByTransactionStart)
	if err == nil {
		t.Fatal("List: want error on truncated body, got nil")
	}
	if got, want := err.Error(), "list connections: received an invalid response, please retry"; got != want {
		t.Fatalf("err = %q, want %q", got, want)
	}
}

// partialInstanceListResponse models a 200 OK that the server returns while a
// fan-out to one instance failed: the primary is flagged unreachable while the
// replica reports cleanly. The error string mirrors a real captured trace.
const partialInstanceListResponse = `{
  "type": "list",
  "captured_at": "2026-04-29T12:34:56.789Z",
  "instances": [
    {"id": "primary", "role": "primary", "error": "remote service unavailable"},
    {"id": "replica-1", "role": "replica", "error": null}
  ],
  "data": []
}`

func TestClient_ListRetriesPartialInstanceErrorThenReturnsClean(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if calls.Add(1) == 1 {
			_, _ = io.WriteString(w, partialInstanceListResponse)
			return
		}
		_, _ = io.WriteString(w, sampleListResponse)
	}))
	defer srv.Close()

	c := newClientWithTimings(t, srv.URL, 500*time.Millisecond, 10*time.Millisecond, 0)
	list, err := c.List(context.Background(), SortByTransactionStart)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("calls = %d, want 2 (retried past the partial-instance failure)", got)
	}
	for _, inst := range list.Instances {
		if inst.Error != "" {
			t.Fatalf("returned list still reports unreachable instance %q (%q), want a clean retry result", inst.ID, inst.Error)
		}
	}
}

func TestClient_ListReturnsPartialAfterRetryBudgetExhausted(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, partialInstanceListResponse)
	}))
	defer srv.Close()

	c := newClientWithTimings(t, srv.URL, 40*time.Millisecond, 10*time.Millisecond, 0)
	start := time.Now()
	list, err := c.List(context.Background(), SortByTransactionStart)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("List: want partial list, got error %v", err)
	}
	if calls.Load() < 2 {
		t.Fatalf("calls = %d, want at least 2 (retried before giving up)", calls.Load())
	}
	if elapsed > time.Second {
		t.Fatalf("List took %v, want retry budget to expire promptly", elapsed)
	}
	// A persistent partial failure still returns useful data: the reachable
	// instances plus the per-instance error that drives the unreachable banner.
	var found bool
	for _, inst := range list.Instances {
		if inst.ID == "primary" && inst.Error == "remote service unavailable" {
			found = true
		}
	}
	if !found {
		t.Fatalf("partial list lost the instance error; instances = %+v", list.Instances)
	}
}

func TestClient_ListReturnsSavedPartialWhenLaterAttemptTimesOut(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, partialInstanceListResponse)
			return
		}
		// Hang a later attempt until the request timeout fires; respect the
		// request context so server shutdown doesn't block on this handler.
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer srv.Close()

	// Budget allows a retry; the per-request timeout fires inside the hung second
	// attempt — the timeout-while-holding-a-saved-partial path. It must return
	// the partial (matching the wait path), not a failed-refresh error.
	c := newClientWithTimings(t, srv.URL, 2*time.Second, 10*time.Millisecond, 0)
	c.cfg.RequestTimeout = 300 * time.Millisecond

	list, err := c.List(context.Background(), SortByTransactionStart)
	if err != nil {
		t.Fatalf("List: want the saved partial, got error %v", err)
	}
	if calls.Load() < 2 {
		t.Fatalf("calls = %d, want at least 2 (timed out on a later attempt)", calls.Load())
	}
	var found bool
	for _, inst := range list.Instances {
		if inst.ID == "primary" && inst.Error == "remote service unavailable" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the saved partial after the timeout; instances = %+v", list.Instances)
	}
}

func TestClient_ListDoesNotRetryPartialWhenBudgetDisabled(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, partialInstanceListResponse)
	}))
	defer srv.Close()

	c := newClientWithTimings(t, srv.URL, 0, 10*time.Millisecond, 0)
	list, err := c.List(context.Background(), SortByTransactionStart)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls = %d, want 1 (no retry when budget disabled)", got)
	}
	if got := list.Instances[0].Error; got != "remote service unavailable" {
		t.Fatalf("Instances[0].Error = %q, want decoded instance error", got)
	}
}

func TestClient_ListRetryWaitHonorsContextCancellation(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := newClientWithTimings(t, srv.URL, 500*time.Millisecond, 250*time.Millisecond, 0)
	start := time.Now()
	_, err := c.List(ctx, SortByTransactionStart)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("List: want context error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls = %d, want 1 because context expired during retry wait", got)
	}
	if elapsed > time.Second {
		t.Fatalf("List took %v, want context cancellation to interrupt retry wait", elapsed)
	}
}

func TestClient_ListPartialRetryWaitHonorsContextCancellation(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, partialInstanceListResponse)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := newClientWithTimings(t, srv.URL, 500*time.Millisecond, 250*time.Millisecond, 0)
	start := time.Now()
	_, err := c.List(ctx, SortByTransactionStart)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("List: want context error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls = %d, want 1 because context expired during partial retry wait", got)
	}
	if elapsed > time.Second {
		t.Fatalf("List took %v, want context cancellation to interrupt partial retry wait", elapsed)
	}
}

func TestClient_ListHonorsRetryAfterHeader(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		call := calls.Add(1)
		if call == 1 {
			// api-bb format from phase-2 RFC line 234: fractional seconds.
			w.Header().Set("Retry-After", "0.020")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, sampleListResponse)
	}))
	defer srv.Close()

	// Backoff floor is longer than RequestTimeout. The call can only succeed if
	// the server-supplied Retry-After hint overrides the local floor.
	c := newClientWithTimings(t, srv.URL, 2*time.Second, 2*time.Second, 0)
	c.cfg.RequestTimeout = 750 * time.Millisecond
	_, err := c.List(context.Background(), SortByTransactionStart)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestClient_RetryDelayUsesJitter(t *testing.T) {
	c := &Client{retryBackoff: 100 * time.Millisecond, retryJitter: 25 * time.Millisecond}
	seen := make(map[time.Duration]bool)
	for range 20 {
		d := c.retryDelay()
		if d < 100*time.Millisecond {
			t.Fatalf("retryDelay() = %v, want >= 100ms floor", d)
		}
		if d >= 125*time.Millisecond {
			t.Fatalf("retryDelay() = %v, want < 125ms ceiling", d)
		}
		seen[d] = true
	}
	if len(seen) <= 1 {
		t.Fatalf("retryDelay() produced %d distinct values across 20 calls, want > 1", len(seen))
	}
}

func TestParseRetryAfterAcceptsPlainDecimalSeconds(t *testing.T) {
	cases := []struct {
		value string
		want  time.Duration
	}{
		{value: "1", want: time.Second},
		{value: "0.020", want: 20 * time.Millisecond},
		{value: " 2.5 ", want: 2500 * time.Millisecond},
	}

	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			if got := parseRetryAfter(tc.value); got != tc.want {
				t.Fatalf("parseRetryAfter(%q) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestParseRetryAfterRejectsInvalidDeltaSeconds(t *testing.T) {
	for _, value := range []string{
		"",
		"NaN",
		"+Inf",
		"-1",
		"+1",
		"1e3",
		"0x1p+2",
		".5",
		"1.",
		"1..2",
		"1.2.3",
	} {
		t.Run(value, func(t *testing.T) {
			if got := parseRetryAfter(value); got != 0 {
				t.Fatalf("parseRetryAfter(%q) = %v, want 0", value, got)
			}
		})
	}
}

func TestParseRetryAfterAcceptsFutureHTTPDate(t *testing.T) {
	value := time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)
	got := parseRetryAfter(value)
	if got <= 0 {
		t.Fatalf("parseRetryAfter(%q) = %v, want positive duration", value, got)
	}
}

func TestParseRetryAfterRejectsPastHTTPDate(t *testing.T) {
	value := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)
	if got := parseRetryAfter(value); got != 0 {
		t.Fatalf("parseRetryAfter(%q) = %v, want 0", value, got)
	}
}

func TestClient_ListServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "upstream exploded: pod=postgres-primary-7-internal")
	}))
	defer srv.Close()

	client := mustClient(t, srv.URL)
	_, err := client.List(context.Background(), SortByTransactionStart)
	if err == nil {
		t.Fatal("List: want error, got nil")
	}
	if strings.Contains(err.Error(), "upstream exploded") || strings.Contains(err.Error(), "postgres-primary-7-internal") {
		t.Errorf("err = %v, must not surface 5xx body (would leak backend topology)", err)
	}
	if strings.Contains(err.Error(), "api-bb") {
		t.Errorf("err = %v, must not leak internal service name", err)
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("err = %v, want status 502 in message", err)
	}
	if !strings.Contains(err.Error(), "Bad Gateway") {
		t.Errorf("err = %v, want generic status text 'Bad Gateway'", err)
	}
	if !strings.Contains(err.Error(), "list connections") {
		t.Errorf("err = %v, want user-facing 'list connections' prefix", err)
	}
}

func TestClient_ListRequiresCapturedAt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"type":"list","data":[]}`)
	}))
	defer srv.Close()

	client := mustClient(t, srv.URL)
	_, err := client.List(context.Background(), SortByTransactionStart)
	if err == nil {
		t.Fatal("List: want missing captured_at error, got nil")
	}
	if !strings.Contains(err.Error(), "captured_at") {
		t.Fatalf("err = %v, want captured_at", err)
	}
}

func mustClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	client, err := NewClient(ClientConfig{
		BaseURL:        baseURL,
		Organization:   "acme",
		Database:       "prod",
		Branch:         "main",
		ServiceTokenID: "tid",
		ServiceToken:   "secret",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

// newClientWithTimings builds a Client with custom retry timings for tests.
// Direct field mutation is intentional and safe: no package-level state, so
// parallel tests cannot interfere with one another.
func newClientWithTimings(t *testing.T, url string, budget, backoff, jitter time.Duration) *Client {
	t.Helper()
	c := mustClient(t, url)
	c.retryBudget = budget
	c.retryBackoff = backoff
	c.retryJitter = jitter
	return c
}

func TestClient_ListHonorsContextDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(30 * time.Second):
		}
	}))
	defer srv.Close()

	client := mustClient(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.List(ctx, SortByTransactionStart)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("List: want context error, got nil")
	}
	if elapsed > 5*time.Second {
		t.Errorf("List took %v, want context deadline to fire promptly", elapsed)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("List error = %v, want context deadline", err)
	}
}

func TestClient_ListHonorsRequestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(30 * time.Second):
		}
	}))
	defer srv.Close()

	client, err := NewClient(ClientConfig{
		BaseURL:        srv.URL,
		Organization:   "acme",
		Database:       "prod",
		Branch:         "main",
		ServiceTokenID: "tid",
		ServiceToken:   "secret",
		RequestTimeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	start := time.Now()
	_, err = client.List(context.Background(), SortByTransactionStart)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("List: want timeout error, got nil")
	}
	if elapsed > time.Second {
		t.Errorf("List took %v, want request timeout to fire promptly", elapsed)
	}
}

func TestClient_ActionMethodsIssuePathSegmentedDeletes(t *testing.T) {
	qid := "primary-123-1777466070000000"
	xid := "primary-123-1777466040000000"
	cid := "primary-123-1779113716123456"
	const base = "/v1/organizations/acme/databases/prod/branches/main/connections"

	cases := []struct {
		name     string
		call     func(*Client) error
		wantPath string
	}{
		{
			name: "CancelQuery",
			call: func(c *Client) error {
				return c.CancelQuery(context.Background(), ActionTarget{Instance: "primary", PID: 123, QueryID: &qid})
			},
			wantPath: base + "/query/primary-123-1777466070000000",
		},
		{
			name: "TerminateTransaction",
			call: func(c *Client) error {
				return c.TerminateTransaction(context.Background(), ActionTarget{Instance: "primary", PID: 123, TransactionID: &xid})
			},
			wantPath: base + "/transaction/primary-123-1777466040000000",
		},
		{
			name: "TerminateConnection",
			call: func(c *Client) error {
				return c.TerminateConnection(context.Background(), ActionTarget{Instance: "primary", PID: 123, ConnectionID: &cid})
			},
			wantPath: base + "/connection/primary-123-1779113716123456",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath, gotQuery string
			var gotBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotQuery = r.URL.RawQuery
				gotBody, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			if err := tc.call(mustClient(t, srv.URL)); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}

			if gotMethod != http.MethodDelete {
				t.Errorf("method = %q, want DELETE", gotMethod)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
			if gotQuery != "" {
				t.Errorf("query string = %q, want empty (no mode= param)", gotQuery)
			}
			if len(gotBody) != 0 {
				t.Errorf("body = %q, want empty", gotBody)
			}
		})
	}
}

func TestDeleteActionUsesStaleGuardMessage(t *testing.T) {
	qid := "primary-123-1777466070000000"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"message":"selected query has already ended; refresh and try again"}`)
	}))
	defer srv.Close()

	err := mustClient(t, srv.URL).CancelQuery(context.Background(), ActionTarget{QueryID: &qid})
	if err == nil {
		t.Fatal("CancelQuery: want error, got nil")
	}

	if got, want := err.Error(), "cancel query: selected query has already ended; refresh and try again"; got != want {
		t.Fatalf("err = %q, want %q", got, want)
	}
	if strings.Contains(err.Error(), "HTTP 422") {
		t.Fatalf("err = %q, must not include HTTP 422", err.Error())
	}
}

func TestClient_ActionResultMethodsIncludeTargetQueryParams(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"keyspace":"commerce","shard":"-80","tablet":"zone1-1001","id":101,"kind":"query"}`)
	}))
	defer srv.Close()

	queryID := "101"
	client, err := NewClient(ClientConfig{
		BaseURL:        srv.URL,
		Organization:   "acme",
		Database:       "shop",
		Branch:         "main",
		Keyspace:       "commerce",
		Shard:          "-80",
		ServiceTokenID: "tid",
		ServiceToken:   "secret",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := client.CancelQueryResult(context.Background(), ActionTarget{QueryID: &queryID})
	if err != nil {
		t.Fatalf("CancelQueryResult: %v", err)
	}

	if gotPath != "/v1/organizations/acme/databases/shop/branches/main/connections/query/101" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "keyspace=commerce&shard=-80" {
		t.Errorf("query = %q, want keyspace=commerce&shard=-80", gotQuery)
	}
	if !result.Success || result.ID != 101 || result.Kind != "query" || result.Tablet != "zone1-1001" {
		t.Errorf("result = %+v", result)
	}
}

func TestClient_ActionResultMethodsRejectUnsuccessfulResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":false}`)
	}))
	defer srv.Close()

	connectionID := "primary-123-c"
	client := mustClient(t, srv.URL)

	_, err := client.TerminateConnectionResult(context.Background(), ActionTarget{ConnectionID: &connectionID})
	if err == nil {
		t.Fatal("TerminateConnectionResult returned nil error for success:false")
	}
	if !strings.Contains(err.Error(), "action did not succeed") {
		t.Fatalf("error = %q, want action did not succeed", err.Error())
	}
}

func TestClient_CancelQueryRejectsMissingQueryID(t *testing.T) {
	client := mustClient(t, "http://example")

	for _, target := range []ActionTarget{
		{Instance: "primary", PID: 123},
		{Instance: "primary", PID: 123, QueryID: ptrString("")},
	} {
		err := client.CancelQuery(context.Background(), target)
		if err == nil {
			t.Fatal("CancelQuery returned nil error for missing QueryID")
		}
		if !strings.Contains(err.Error(), "query_id") {
			t.Errorf("error = %q, want message mentioning query_id", err.Error())
		}
	}
}

func TestClient_TerminateTransactionRejectsMissingTransactionID(t *testing.T) {
	client := mustClient(t, "http://example")

	for _, target := range []ActionTarget{
		{Instance: "primary", PID: 123},
		{Instance: "primary", PID: 123, TransactionID: ptrString("")},
	} {
		err := client.TerminateTransaction(context.Background(), target)
		if err == nil {
			t.Fatal("TerminateTransaction returned nil error for missing TransactionID")
		}
		if !strings.Contains(err.Error(), "transaction_id") {
			t.Errorf("error = %q, want message mentioning transaction_id", err.Error())
		}
	}
}

func TestClient_TerminateConnectionRejectsMissingConnectionID(t *testing.T) {
	client := mustClient(t, "http://example")

	for _, target := range []ActionTarget{
		{Instance: "primary", PID: 123},
		{Instance: "primary", PID: 123, ConnectionID: ptrString("")},
	} {
		err := client.TerminateConnection(context.Background(), target)
		if err == nil {
			t.Fatal("TerminateConnection returned nil error for missing ConnectionID")
		}
		if !strings.Contains(err.Error(), "connection_id") {
			t.Errorf("error = %q, want message mentioning connection_id", err.Error())
		}
	}
}

func ptrString(v string) *string {
	return &v
}
