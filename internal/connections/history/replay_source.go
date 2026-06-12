package history

import (
	"context"
	"errors"
	"io"

	"github.com/planetscale/cli/internal/connections"
)

// ReplaySource serves captured snapshots from a trace file as if they were
// live.
type ReplaySource struct {
	captures []Capture
	start    CaptureStart
	hasStart bool
}

// NewReplaySource loads every Capture from r into memory. Returns an error
// when the trace contains no complete capture records — replay against an
// empty or header-only file is not meaningful.
func NewReplaySource(r io.Reader) (*ReplaySource, error) {
	reader := NewCaptureReader(r)
	captures, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(captures) == 0 {
		return nil, errors.New("capture file contains no replayable snapshots")
	}
	start, hasStart := reader.CaptureStart()
	return &ReplaySource{captures: captures, start: start, hasStart: hasStart}, nil
}

// List returns the latest captured snapshot, sorted by mode.
func (s *ReplaySource) List(_ context.Context, mode connections.SortMode) (connections.ConnectionList, error) {
	out := s.captures[len(s.captures)-1].List
	connections.SortConnections(out.Connections, mode)
	out.Sort = mode
	return out, nil
}

// Captures returns the loaded captures in order. Used by the CLI to seed the
// TUI's capture history so the operator can step the full timeline.
func (s *ReplaySource) Captures() []Capture {
	return s.captures
}

// CaptureStart returns the trace metadata header when the capture file has one.
func (s *ReplaySource) CaptureStart() (CaptureStart, bool) {
	return s.start, s.hasStart
}
