package history

import (
	"encoding/json"
	"io"
	"time"
)

const traceSchemaVersion = 1

type CaptureStart struct {
	Type          string         `json:"type"`
	At            time.Time      `json:"at"`
	Organization  string         `json:"org"`
	Database      string         `json:"database"`
	Branch        string         `json:"branch"`
	SchemaVersion int            `json:"schema_version"`
	Filter        *CaptureFilter `json:"filter,omitempty"`
	Target        *CaptureTarget `json:"target,omitempty"`
}

// CaptureTarget records the Vitess target active at capture time.
type CaptureTarget struct {
	Keyspace string `json:"keyspace,omitempty"`
	Shard    string `json:"shard,omitempty"`
}

// CaptureFilter records any client-side row filter active at capture time so
// replay tooling can tell a partial snapshot from a complete branch view.
// Omitted when no filter was set.
type CaptureFilter struct {
	Instance string `json:"instance,omitempty"`
	Role     string `json:"role,omitempty"`
}

// CaptureWriter writes captures as newline-delimited JSON records.
type CaptureWriter struct {
	writer io.Writer
}

func NewCaptureWriter(writer io.Writer) *CaptureWriter {
	return &CaptureWriter{writer: writer}
}

func (w *CaptureWriter) Write(capture Capture) error {
	return w.writeRecord(capture)
}

func (w *CaptureWriter) WriteCaptureStart(start CaptureStart) error {
	start.Type = "capture_start"
	if start.SchemaVersion == 0 {
		start.SchemaVersion = traceSchemaVersion
	}
	return w.writeRecord(start)
}

func (w *CaptureWriter) writeRecord(value any) error {
	record, err := json.Marshal(value)
	if err != nil {
		return err
	}
	record = append(record, '\n')
	n, err := w.writer.Write(record)
	if err != nil {
		return err
	}
	if n != len(record) {
		return io.ErrShortWrite
	}
	return w.Flush()
}

func (w *CaptureWriter) Flush() error {
	flusher, ok := w.writer.(interface{ Flush() error })
	if !ok {
		return nil
	}
	return flusher.Flush()
}

func (w *CaptureWriter) Close() error {
	flushErr := w.Flush()
	closer, ok := w.writer.(io.Closer)
	if !ok {
		return flushErr
	}
	closeErr := closer.Close()
	if flushErr != nil {
		return flushErr
	}
	return closeErr
}
