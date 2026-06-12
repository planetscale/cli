package history

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/connections"

	qt "github.com/frankban/quicktest"
)

func TestCaptureReaderReadsCapturesWrittenByCaptureWriter(t *testing.T) {
	c := qt.New(t)
	var buffer bytes.Buffer
	writer := NewCaptureWriter(&buffer)
	c.Assert(writer.WriteCaptureStart(CaptureStart{
		At:           time.Date(2026, 4, 28, 14, 59, 0, 0, time.UTC),
		Organization: "acme",
		Database:     "prod",
		Branch:       "main",
	}), qt.IsNil)
	c.Assert(writer.Write(traceCapture()), qt.IsNil)

	reader := NewCaptureReader(&buffer)
	capture, err := reader.Read()

	c.Assert(err, qt.IsNil)
	c.Assert(capture.List.Connections[0].PID, qt.Equals, 101)
	c.Assert(capture.List.Connections[0].QueryText, qt.Equals, "SELECT * FROM widgets")

	_, err = reader.Read()
	c.Assert(err, qt.Equals, io.EOF)
}

func TestCaptureReaderRejectsUnsupportedSchemaVersion(t *testing.T) {
	c := qt.New(t)
	input := `{"type":"capture_start","schema_version":2}` + "\n"
	reader := NewCaptureReader(strings.NewReader(input))

	_, err := reader.Read()

	c.Assert(err, qt.ErrorMatches, `capture line 1: schema version 2 is not supported \(expected 1\)`)
}

// Tail tolerance: a SIGINT during write can leave a torn final line. The
// reader should return EOF, not surface a JSON error, so replay still works on
// a partially-flushed file.
func TestCaptureReaderToleratesTornFinalLine(t *testing.T) {
	c := qt.New(t)
	var buffer bytes.Buffer
	writer := NewCaptureWriter(&buffer)
	c.Assert(writer.Write(traceCapture()), qt.IsNil)
	buffer.WriteString(`{"at":"2026-04-28T15:00:01Z"`)

	reader := NewCaptureReader(&buffer)
	_, err := reader.Read()
	c.Assert(err, qt.IsNil)
	_, err = reader.Read()
	c.Assert(err, qt.Equals, io.EOF)
}

// A complete final line that is malformed JSON must surface as an error, not
// be silently dropped — otherwise file corruption goes undetected.
func TestCaptureReaderReportsMalformedCompleteFinalLine(t *testing.T) {
	c := qt.New(t)
	var buffer bytes.Buffer
	writer := NewCaptureWriter(&buffer)
	c.Assert(writer.Write(traceCapture()), qt.IsNil)
	buffer.WriteString(`{"at":` + "\n")

	reader := NewCaptureReader(&buffer)
	_, err := reader.Read()
	c.Assert(err, qt.IsNil)
	_, err = reader.Read()
	c.Assert(err, qt.ErrorMatches, `capture line 2: .*`)
}

func TestCaptureReaderReadAllReturnsCapturesInOrder(t *testing.T) {
	c := qt.New(t)
	var buffer bytes.Buffer
	writer := NewCaptureWriter(&buffer)
	c.Assert(writer.WriteCaptureStart(CaptureStart{At: time.Date(2026, 4, 28, 14, 59, 0, 0, time.UTC)}), qt.IsNil)

	at := time.Date(2026, 4, 28, 15, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		list := connections.NewConnectionList(at.Add(time.Duration(i)*time.Second), []connections.Connection{{
			PID:      100 + i,
			Instance: "primary",
		}}, connections.SortByTransactionStart)
		c.Assert(writer.Write(NewCapture(list)), qt.IsNil)
	}

	reader := NewCaptureReader(&buffer)
	captures, err := reader.ReadAll()

	c.Assert(err, qt.IsNil)
	c.Assert(captures, qt.HasLen, 3)
	for i, capture := range captures {
		c.Assert(capture.List.Connections[0].PID, qt.Equals, 100+i)
	}
}

func TestCaptureReaderReadAllSkipsMalformedCaptureStartMetadata(t *testing.T) {
	c := qt.New(t)
	var buffer bytes.Buffer
	buffer.WriteString(`{"type":"capture_start","schema_version":1,"at":{}}` + "\n")
	writer := NewCaptureWriter(&buffer)
	c.Assert(writer.Write(traceCapture()), qt.IsNil)

	reader := NewCaptureReader(bytes.NewReader(buffer.Bytes()))
	captures, err := reader.ReadAll()

	c.Assert(err, qt.IsNil)
	c.Assert(captures, qt.HasLen, 1)
	c.Assert(captures[0].List.Connections[0].PID, qt.Equals, 101)
}

// Round-trip byte identity: writing → reading → writing the same captures
// produces byte-identical output. Locks the format against accidental
// reordering or escaping drift in the serializer.
func TestCaptureRoundTripIsByteIdentical(t *testing.T) {
	c := qt.New(t)
	var first bytes.Buffer
	w1 := NewCaptureWriter(&first)
	c.Assert(w1.WriteCaptureStart(CaptureStart{
		At:           time.Date(2026, 4, 28, 14, 59, 0, 0, time.UTC),
		Organization: "acme",
		Database:     "prod",
		Branch:       "main",
	}), qt.IsNil)
	c.Assert(w1.Write(traceCapture()), qt.IsNil)

	reader := NewCaptureReader(bytes.NewReader(first.Bytes()))
	captures, err := reader.ReadAll()
	c.Assert(err, qt.IsNil)

	var second bytes.Buffer
	w2 := NewCaptureWriter(&second)
	c.Assert(w2.WriteCaptureStart(CaptureStart{
		At:           time.Date(2026, 4, 28, 14, 59, 0, 0, time.UTC),
		Organization: "acme",
		Database:     "prod",
		Branch:       "main",
	}), qt.IsNil)
	for _, capture := range captures {
		c.Assert(w2.Write(capture), qt.IsNil)
	}

	c.Assert(second.Bytes(), qt.DeepEquals, first.Bytes())
}

// Anti-regression: a pre-amendment capture file with old-only fields must
// surface a clear decode error rather than being accepted as a replayable
// capture.
func TestCaptureReaderRejectsOldShapeCaptureCleanly(t *testing.T) {
	c := qt.New(t)
	oldShape := `{"at":"2026-04-28T15:00:00Z","capture":{"captured_at":"2026-04-28T15:00:00Z","connections":[{"id":"primary-10","pid":10,"instance":"primary","username":"brett","application_name":"psql","client_addr":"127.0.0.1","state":"active","wait_event_type":"","wait_event":"","backend_type":"","query_id":"primary-10-1779113716123456","duration":5000000000}],"sort":"xact_start"}}` + "\n"

	reader := NewCaptureReader(strings.NewReader(oldShape))
	_, err := reader.Read()

	c.Assert(err, qt.ErrorMatches, `capture line 1: json: unknown field "id"`)
}
