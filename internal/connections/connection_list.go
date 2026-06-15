package connections

import (
	"slices"
	"strconv"
	"strings"
	"time"
)

type SortMode string

const (
	SortByTransactionStart SortMode = "xact_start"
	SortByDuration         SortMode = "duration"
	SortByBlocked          SortMode = "blocked"
)

type DatabaseKind string

const (
	DatabaseKindMySQL      DatabaseKind = "mysql"
	DatabaseKindPostgreSQL DatabaseKind = "postgresql"
)

type InstanceMeta struct {
	ID    string `json:"id"`
	Role  string `json:"role"`
	Error string `json:"error,omitempty"`
}

type Topology struct {
	Keyspace string `json:"keyspace,omitempty"`
	Shard    string `json:"shard,omitempty"`
	Tablet   string `json:"tablet,omitempty"`
}

type ConnectionList struct {
	CapturedAt   time.Time      `json:"captured_at"`
	DatabaseKind DatabaseKind   `json:"database_kind,omitempty"`
	Connections  []Connection   `json:"connections"`
	Sort         SortMode       `json:"sort"`
	Instances    []InstanceMeta `json:"instances,omitempty"`
	Topology     *Topology      `json:"topology,omitempty"`
}

// HasUnreachableInstance reports whether the server flagged any instance as
// unreachable in the list response. These partial fan-out failures drive the
// "N of M instances unreachable" banner.
func (l ConnectionList) HasUnreachableInstance() bool {
	for _, inst := range l.Instances {
		if inst.Error != "" {
			return true
		}
	}
	return false
}

type Connection struct {
	PID             int           `json:"pid"`
	Instance        string        `json:"instance"`
	InstanceRole    string        `json:"instance_role,omitempty"`
	Username        string        `json:"username"`
	ApplicationName string        `json:"application_name"`
	DatabaseName    string        `json:"database,omitempty"`
	ClientAddr      string        `json:"client_addr"`
	State           string        `json:"state"`
	WaitEventType   string        `json:"wait_event_type"`
	WaitEvent       string        `json:"wait_event"`
	BackendType     string        `json:"backend_type"`
	XactStart       *time.Time    `json:"xact_start,omitempty"`
	QueryStart      *time.Time    `json:"query_start,omitempty"`
	ConnectionID    *string       `json:"connection_id,omitempty"`
	TransactionID   *string       `json:"transaction_id,omitempty"`
	QueryID         *string       `json:"query_id,omitempty"`
	Duration        time.Duration `json:"duration"`
	BlockedBy       []int         `json:"blocked_by,omitempty"`
	QueryText       string        `json:"query_text,omitempty"`
}

func NewConnectionList(capturedAt time.Time, connections []Connection, sort SortMode) ConnectionList {
	out := slices.Clone(connections)
	SortConnections(out, sort)

	return ConnectionList{
		CapturedAt:  capturedAt,
		Connections: out,
		Sort:        sort,
	}
}

func SortConnections(connections []Connection, mode SortMode) {
	switch mode {
	case SortByBlocked:
		counts := BlockingCounts(connections)
		slices.SortStableFunc(connections, func(a, b Connection) int {
			return compareBlocked(a, b, counts)
		})
	case SortByDuration:
		slices.SortStableFunc(connections, compareDuration)
	default:
		slices.SortStableFunc(connections, compareTransactionStart)
	}
}

func compareDuration(a, b Connection) int {
	if a.Duration > b.Duration {
		return -1
	}
	if a.Duration < b.Duration {
		return 1
	}
	if cmp := compareBool(durationSortHasWork(a), durationSortHasWork(b)); cmp != 0 {
		return cmp
	}
	return a.PID - b.PID
}

func durationSortHasWork(c Connection) bool {
	state := strings.ToLower(strings.TrimSpace(c.State))
	if state == "sleep" || state == "idle" || strings.HasPrefix(state, "idle ") {
		return false
	}
	return state != "" || strings.TrimSpace(c.QueryText) != ""
}

func compareBlocked(a, b Connection, counts map[int]int) int {
	if cmp := compareIntDesc(counts[a.PID], counts[b.PID]); cmp != 0 {
		return cmp
	}
	if cmp := compareBool(activeAndBlocked(a), activeAndBlocked(b)); cmp != 0 {
		return cmp
	}
	if cmp := compareBool(len(a.BlockedBy) > 0, len(b.BlockedBy) > 0); cmp != 0 {
		return cmp
	}
	return compareTransactionStart(a, b)
}

func activeAndBlocked(c Connection) bool {
	return c.State == "active" && len(c.BlockedBy) > 0
}

func compareBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return -1
	default:
		return 1
	}
}

func compareIntDesc(a, b int) int {
	if a > b {
		return -1
	}
	if a < b {
		return 1
	}
	return 0
}

// BlockingCounts returns each connection's downstream blocking depth by PID.
func BlockingCounts(connections []Connection) map[int]int {
	downstream := make(map[int][]int)
	for _, conn := range connections {
		for _, blockerPID := range conn.BlockedBy {
			downstream[blockerPID] = append(downstream[blockerPID], conn.PID)
		}
	}

	counts := make(map[int]int)
	for _, conn := range connections {
		count := downstreamCount(conn.PID, downstream, map[int]bool{conn.PID: true})
		if count > 0 {
			counts[conn.PID] = count
		}
	}
	return counts
}

func downstreamCount(pid int, downstream map[int][]int, seen map[int]bool) int {
	count := 0
	for _, blockedPID := range downstream[pid] {
		if seen[blockedPID] {
			continue
		}
		seen[blockedPID] = true
		count += 1 + downstreamCount(blockedPID, downstream, seen)
	}
	return count
}

func compareTransactionStart(a, b Connection) int {
	if cmp := compareNullableTime(a.XactStart, b.XactStart); cmp != 0 {
		return cmp
	}
	if cmp := compareNullableTime(a.QueryStart, b.QueryStart); cmp != 0 {
		return cmp
	}
	return a.PID - b.PID
}

func compareNullableTime(a, b *time.Time) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	case a.Before(*b):
		return -1
	case a.After(*b):
		return 1
	default:
		return 0
	}
}

// HumanFields returns the connection's scalar fields as ordered (name, value)
// pairs for vertical, MySQL `\G`-style rendering. The query text is excluded —
// callers render it separately because it is multi-line. This is the single
// source of truth shared by the agent-cli `list --format human` output and the
// interactive detail view, so the two never drift.
func (c Connection) HumanFields() [][2]string {
	return [][2]string{
		{"pid", strconv.Itoa(c.PID)},
		{"instance", c.Instance},
		{"role", c.InstanceRole},
		{"state", c.State},
		{"duration", c.Duration.String()},
		{"wait", JoinWaitEvents(c.WaitEventType, c.WaitEvent)},
		{"user", c.Username},
		{"application", c.ApplicationName},
		{"client_addr", c.ClientAddr},
		{"blocked_by", JoinInts(c.BlockedBy)},
		{"query_id", DerefString(c.QueryID)},
		{"transaction_id", DerefString(c.TransactionID)},
		{"connection_id", DerefString(c.ConnectionID)},
	}
}

// JoinWaitEvents renders the wait-event type/event pair as "type/event",
// collapsing to whichever side is present when one is empty.
func JoinWaitEvents(waitEventType, waitEvent string) string {
	if waitEventType == "" {
		return waitEvent
	}
	if waitEvent == "" {
		return waitEventType
	}
	return waitEventType + "/" + waitEvent
}

// JoinInts renders a blocked-by PID slice as a comma-separated string.
func JoinInts(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ",")
}

// DerefString returns the pointed-to string, or "" when nil.
func DerefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
