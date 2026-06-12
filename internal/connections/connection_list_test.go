package connections

import (
	"encoding/json"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestNewConnectionListSortsByTransactionStartWithNullsLast(t *testing.T) {
	c := qt.New(t)
	now := time.Date(2026, 4, 28, 15, 12, 4, 0, time.UTC)
	oldXact := now.Add(-20 * time.Minute)
	newXact := now.Add(-5 * time.Minute)
	oldQuery := now.Add(-10 * time.Minute)
	newQuery := now.Add(-2 * time.Minute)

	list := NewConnectionList(now, []Connection{
		{PID: 4, XactStart: nil, QueryStart: &newQuery},
		{PID: 2, XactStart: &newXact, QueryStart: &oldQuery},
		{PID: 3, XactStart: nil, QueryStart: &oldQuery},
		{PID: 1, XactStart: &oldXact, QueryStart: &newQuery},
	}, SortByTransactionStart)

	c.Assert(pids(list.Connections), qt.DeepEquals, []int{1, 2, 3, 4})
}

func TestNewConnectionListSortsByTransactionStartWithQueryStartAndPIDTieBreakers(t *testing.T) {
	c := qt.New(t)
	now := time.Date(2026, 4, 28, 15, 12, 4, 0, time.UTC)
	xactStart := now.Add(-20 * time.Minute)
	oldQuery := now.Add(-10 * time.Minute)
	newQuery := now.Add(-2 * time.Minute)

	list := NewConnectionList(now, []Connection{
		{PID: 6, XactStart: &xactStart, QueryStart: nil},
		{PID: 4, XactStart: &xactStart, QueryStart: &oldQuery},
		{PID: 2, XactStart: &xactStart, QueryStart: &oldQuery},
		{PID: 5, XactStart: &xactStart, QueryStart: &newQuery},
		{PID: 3, XactStart: &xactStart, QueryStart: nil},
	}, SortByTransactionStart)

	c.Assert(pids(list.Connections), qt.DeepEquals, []int{2, 4, 5, 3, 6})
}

func TestConnectionListJSONUsesFoundationShape(t *testing.T) {
	c := qt.New(t)
	capturedAt := time.Date(2026, 4, 28, 15, 12, 4, 0, time.UTC)
	xactStart := capturedAt.Add(-20 * time.Minute)
	queryStart := capturedAt.Add(-5 * time.Minute)

	list := ConnectionList{
		CapturedAt: capturedAt,
		Connections: []Connection{{
			PID:             101,
			Instance:        "primary",
			Username:        "brett",
			ApplicationName: "psql",
			ClientAddr:      "127.0.0.1",
			State:           "active",
			WaitEventType:   "Lock",
			WaitEvent:       "transactionid",
			BackendType:     "client backend",
			XactStart:       &xactStart,
			QueryStart:      &queryStart,
			Duration:        5 * time.Second,
			BlockedBy:       []int{201},
			QueryText:       "SELECT * FROM widgets",
		}},
		Sort: SortByTransactionStart,
	}

	data, err := json.Marshal(list)
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.JSONEquals, map[string]any{
		"captured_at": capturedAt.Format(time.RFC3339),
		"connections": []any{map[string]any{
			"pid":              101,
			"instance":         "primary",
			"username":         "brett",
			"application_name": "psql",
			"client_addr":      "127.0.0.1",
			"state":            "active",
			"wait_event_type":  "Lock",
			"wait_event":       "transactionid",
			"backend_type":     "client backend",
			"xact_start":       xactStart.Format(time.RFC3339),
			"query_start":      queryStart.Format(time.RFC3339),
			"duration":         5000000000,
			"blocked_by":       []any{201},
			"query_text":       "SELECT * FROM widgets",
		}},
		"sort": "xact_start",
	})
}

func TestConnectionRoundTripsCompositeKey(t *testing.T) {
	c := qt.New(t)

	original := Connection{PID: 123, Instance: "primary"}

	data, err := json.Marshal(original)
	c.Assert(err, qt.IsNil)

	var decoded Connection
	c.Assert(json.Unmarshal(data, &decoded), qt.IsNil)
	c.Assert(decoded.PID, qt.Equals, 123)
	c.Assert(decoded.Instance, qt.Equals, "primary")
}

func TestConnectionListJSONRoundTripsInstances(t *testing.T) {
	c := qt.New(t)
	capturedAt := time.Date(2026, 4, 28, 15, 12, 4, 0, time.UTC)

	original := ConnectionList{
		CapturedAt:  capturedAt,
		Connections: []Connection{},
		Sort:        SortByTransactionStart,
		Instances: []InstanceMeta{
			{ID: "primary", Role: "primary"},
			{ID: "replica-1", Role: "replica", Error: "timeout after 2s"},
		},
	}

	data, err := json.Marshal(original)
	c.Assert(err, qt.IsNil)

	var decoded ConnectionList
	c.Assert(json.Unmarshal(data, &decoded), qt.IsNil)
	c.Assert(decoded.Instances, qt.DeepEquals, original.Instances)
}

func TestConnectionJSONOmitsOpaqueIDField(t *testing.T) {
	c := qt.New(t)
	conn := Connection{PID: 123, Instance: "primary"}

	data, err := json.Marshal(conn)
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Not(qt.Contains), `"id"`)
}

func TestConnectionCarriesActionIDs(t *testing.T) {
	c := qt.New(t)
	xactStart := time.Date(2026, 4, 28, 15, 12, 4, 123_000, time.UTC)
	queryStart := xactStart.Add(time.Second)
	connID := "primary-101-1779113716123456"
	txID := "primary-101-1777476480123456"
	qID := "primary-101-1777476481456789"

	conn := Connection{
		PID:           101,
		Instance:      "primary",
		XactStart:     &xactStart,
		QueryStart:    &queryStart,
		ConnectionID:  &connID,
		TransactionID: &txID,
		QueryID:       &qID,
	}

	data, err := json.Marshal(conn)
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Contains, `"connection_id":"primary-101-1779113716123456"`)
	c.Assert(string(data), qt.Contains, `"transaction_id":"primary-101-1777476480123456"`)
	c.Assert(string(data), qt.Contains, `"query_id":"primary-101-1777476481456789"`)

	var decoded Connection
	c.Assert(json.Unmarshal(data, &decoded), qt.IsNil)
	c.Assert(decoded.ConnectionID, qt.IsNotNil)
	c.Assert(*decoded.ConnectionID, qt.Equals, "primary-101-1779113716123456")
	c.Assert(decoded.TransactionID, qt.IsNotNil)
	c.Assert(*decoded.TransactionID, qt.Equals, "primary-101-1777476480123456")
	c.Assert(decoded.QueryID, qt.IsNotNil)
	c.Assert(*decoded.QueryID, qt.Equals, "primary-101-1777476481456789")
}

func TestConnectionOmitsNilActionIDs(t *testing.T) {
	c := qt.New(t)
	conn := Connection{PID: 1, Instance: "primary"}

	data, err := json.Marshal(conn)
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Not(qt.Contains), `"connection_id"`)
	c.Assert(string(data), qt.Not(qt.Contains), `"transaction_id"`)
	c.Assert(string(data), qt.Not(qt.Contains), `"query_id"`)
}

func TestSortConnectionsByDurationDescending(t *testing.T) {
	c := qt.New(t)

	connections := []Connection{
		{PID: 1, Duration: 1 * time.Second},
		{PID: 2, Duration: 10 * time.Second},
		{PID: 3, Duration: 5 * time.Second},
	}
	SortConnections(connections, SortByDuration)

	c.Assert(pids(connections), qt.DeepEquals, []int{2, 3, 1})
}

func TestSortConnectionsByDurationPrioritizesActiveRowsOnTie(t *testing.T) {
	c := qt.New(t)

	connections := []Connection{
		{PID: 10, State: "Sleep", Duration: 0},
		{PID: 20, State: "Query/update", Duration: 0, QueryText: "INSERT INTO events VALUES (1)"},
		{PID: 30, State: "Sleep", Duration: 0},
	}
	SortConnections(connections, SortByDuration)

	c.Assert(pids(connections), qt.DeepEquals, []int{20, 10, 30})
}

func TestSortConnectionsByBlockedPutsActiveBlockedFirst(t *testing.T) {
	c := qt.New(t)
	xactStart := time.Date(2026, 4, 28, 15, 12, 4, 0, time.UTC)

	connections := []Connection{
		{PID: 1, State: "idle", BlockedBy: []int{99}, XactStart: &xactStart},
		{PID: 2, State: "active", BlockedBy: []int{99}, XactStart: &xactStart},
		{PID: 3, State: "active", XactStart: &xactStart},
	}
	SortConnections(connections, SortByBlocked)

	c.Assert(pids(connections), qt.DeepEquals, []int{2, 1, 3})
}

func TestSortConnectionsByBlockedPutsRootBlockersFirst(t *testing.T) {
	c := qt.New(t)
	xactStart := time.Date(2026, 4, 28, 15, 12, 4, 0, time.UTC)

	connections := []Connection{
		{PID: 20, State: "active", BlockedBy: []int{10}, XactStart: &xactStart},
		{PID: 30, State: "active", BlockedBy: []int{20}, XactStart: &xactStart},
		{PID: 40, State: "active", BlockedBy: []int{10}, XactStart: &xactStart},
		{PID: 10, State: "idle in transaction", XactStart: &xactStart},
		{PID: 50, State: "active", XactStart: &xactStart},
	}
	SortConnections(connections, SortByBlocked)

	c.Assert(pids(connections), qt.DeepEquals, []int{10, 20, 30, 40, 50})
}

func TestBlockingCountsCountsDownstreamDepth(t *testing.T) {
	c := qt.New(t)

	connections := []Connection{
		{PID: 1},
		{PID: 2, BlockedBy: []int{1}},
		{PID: 3, BlockedBy: []int{2}},
		{PID: 4, BlockedBy: []int{1}},
	}

	counts := BlockingCounts(connections)

	c.Assert(counts[1], qt.Equals, 3)
	c.Assert(counts[2], qt.Equals, 1)
	c.Assert(counts[3], qt.Equals, 0)
}

func pids(connections []Connection) []int {
	out := make([]int, 0, len(connections))
	for _, conn := range connections {
		out = append(out, conn.PID)
	}
	return out
}
