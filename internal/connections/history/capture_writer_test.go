package history

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/connections"

	qt "github.com/frankban/quicktest"
)

func TestTraceFormatStability(t *testing.T) {
	c := qt.New(t)
	var buffer bytes.Buffer
	writer := NewCaptureWriter(&buffer)

	err := writer.Write(traceCapture())

	c.Assert(err, qt.IsNil)
	c.Assert(buffer.String(), qt.JSONEquals, map[string]any{
		"at": "2026-04-28T15:00:00Z",
		"capture": map[string]any{
			"captured_at": "2026-04-28T15:00:00Z",
			"connections": []any{map[string]any{
				"pid":              101,
				"instance":         "",
				"username":         "brett",
				"application_name": "psql",
				"client_addr":      "127.0.0.1",
				"state":            "active",
				"wait_event_type":  "",
				"wait_event":       "",
				"backend_type":     "",
				"duration":         5000000000,
				"query_text":       "SELECT * FROM widgets",
			}},
			"sort": "xact_start",
		},
	})
}

func TestCaptureWriterWritesCaptureStart(t *testing.T) {
	c := qt.New(t)
	var buffer bytes.Buffer
	writer := NewCaptureWriter(&buffer)
	c.Assert(writer.WriteCaptureStart(CaptureStart{
		At:            time.Date(2026, 4, 28, 14, 59, 0, 0, time.UTC),
		Organization:  "acme",
		Database:      "prod",
		Branch:        "main",
		SchemaVersion: 1,
	}), qt.IsNil)
	c.Assert(buffer.String(), qt.Equals, `{"type":"capture_start","at":"2026-04-28T14:59:00Z","org":"acme","database":"prod","branch":"main","schema_version":1}`+"\n")
}

// When a filter is active at capture time, the header records it so replay
// tooling can distinguish a partial snapshot from a complete branch view.
func TestCaptureWriterRecordsFilter(t *testing.T) {
	c := qt.New(t)
	var buffer bytes.Buffer
	writer := NewCaptureWriter(&buffer)
	c.Assert(writer.WriteCaptureStart(CaptureStart{
		At:            time.Date(2026, 4, 28, 14, 59, 0, 0, time.UTC),
		Organization:  "acme",
		Database:      "prod",
		Branch:        "main",
		SchemaVersion: 1,
		Filter:        &CaptureFilter{Role: "replica"},
	}), qt.IsNil)
	c.Assert(buffer.String(), qt.JSONEquals, map[string]any{
		"type":           "capture_start",
		"at":             "2026-04-28T14:59:00Z",
		"org":            "acme",
		"database":       "prod",
		"branch":         "main",
		"schema_version": 1,
		"filter": map[string]any{
			"role": "replica",
		},
	})
}

func TestCaptureStartRecordsVitessTarget(t *testing.T) {
	c := qt.New(t)
	var buffer bytes.Buffer
	writer := NewCaptureWriter(&buffer)
	c.Assert(writer.WriteCaptureStart(CaptureStart{
		At:            time.Date(2026, 4, 28, 14, 59, 0, 0, time.UTC),
		Organization:  "acme",
		Database:      "prod",
		Branch:        "main",
		SchemaVersion: 1,
		Target: &CaptureTarget{
			Keyspace: "commerce",
			Shard:    "-80",
		},
	}), qt.IsNil)

	c.Assert(buffer.String(), qt.JSONEquals, map[string]any{
		"type":           "capture_start",
		"at":             "2026-04-28T14:59:00Z",
		"org":            "acme",
		"database":       "prod",
		"branch":         "main",
		"schema_version": 1,
		"target": map[string]any{
			"keyspace": "commerce",
			"shard":    "-80",
		},
	})

	reader := NewCaptureReader(bytes.NewReader(buffer.Bytes()))
	_, err := reader.Read()
	c.Assert(err, qt.Equals, io.EOF)
	start, ok := reader.CaptureStart()
	c.Assert(ok, qt.IsTrue)
	c.Assert(start.Target, qt.DeepEquals, &CaptureTarget{
		Keyspace: "commerce",
		Shard:    "-80",
	})
}

func traceCapture() Capture {
	capturedAt := time.Date(2026, 4, 28, 15, 0, 0, 0, time.UTC)
	return NewCapture(connections.NewConnectionList(capturedAt, []connections.Connection{{
		PID:             101,
		Username:        "brett",
		ApplicationName: "psql",
		ClientAddr:      "127.0.0.1",
		State:           "active",
		Duration:        5 * time.Second,
		QueryText:       "SELECT * FROM widgets",
	}}, connections.SortByTransactionStart))
}
