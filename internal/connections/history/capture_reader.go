package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// CaptureReader reads Capture records from a stream previously written by
// CaptureWriter. It skips capture_start headers and rejects unsupported schema
// versions on capture_start.
type CaptureReader struct {
	reader       *bufio.Reader
	line         int
	done         bool
	captureStart *CaptureStart
}

func NewCaptureReader(r io.Reader) *CaptureReader {
	return &CaptureReader{reader: bufio.NewReader(r)}
}

// Read returns the next Capture in the stream, or io.EOF when exhausted. A
// torn final line (no trailing newline, partial JSON) is treated as EOF so
// captures interrupted by SIGINT remain replayable.
func (r *CaptureReader) Read() (Capture, error) {
	if r.done {
		return Capture{}, io.EOF
	}

	for {
		line, err := r.reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return Capture{}, err
		}
		if err == io.EOF {
			r.done = true
		}
		if line == "" && err == io.EOF {
			return Capture{}, io.EOF
		}

		r.line++
		if strings.TrimSpace(line) == "" {
			if err == io.EOF {
				return Capture{}, io.EOF
			}
			continue
		}

		complete := strings.HasSuffix(line, "\n")
		var envelope struct {
			Type          string `json:"type"`
			SchemaVersion int    `json:"schema_version"`
		}
		if decodeErr := json.Unmarshal([]byte(line), &envelope); decodeErr != nil {
			if err == io.EOF && !complete {
				return Capture{}, io.EOF
			}
			return Capture{}, fmt.Errorf("capture line %d: %w", r.line, decodeErr)
		}
		if envelope.Type == "capture_start" {
			if envelope.SchemaVersion != traceSchemaVersion {
				return Capture{}, fmt.Errorf("capture line %d: schema version %d is not supported (expected %d)", r.line, envelope.SchemaVersion, traceSchemaVersion)
			}
			var start CaptureStart
			if decodeErr := json.Unmarshal([]byte(line), &start); decodeErr == nil {
				r.captureStart = &start
			}
			if err == io.EOF {
				return Capture{}, io.EOF
			}
			continue
		}
		if envelope.Type != "" {
			return Capture{}, fmt.Errorf("capture line %d: record type %q is not supported", r.line, envelope.Type)
		}

		var capture Capture
		decoder := json.NewDecoder(strings.NewReader(line))
		decoder.DisallowUnknownFields()
		if decodeErr := decoder.Decode(&capture); decodeErr != nil {
			return Capture{}, fmt.Errorf("capture line %d: %w", r.line, decodeErr)
		}
		return capture, nil
	}
}

// CaptureStart returns the parsed capture_start metadata when the trace
// contains a well-formed header.
func (r *CaptureReader) CaptureStart() (CaptureStart, bool) {
	if r.captureStart == nil {
		return CaptureStart{}, false
	}
	return *r.captureStart, true
}

// ReadAll consumes the stream and returns every complete capture in order.
func (r *CaptureReader) ReadAll() ([]Capture, error) {
	var captures []Capture
	for {
		capture, err := r.Read()
		if err == io.EOF {
			return captures, nil
		}
		if err != nil {
			return nil, err
		}
		captures = append(captures, capture)
	}
}
