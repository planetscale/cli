package connections

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	live "github.com/planetscale/cli/internal/connections"
	"github.com/planetscale/cli/internal/connections/history"
)

type stubClientInterface struct {
	lists  []live.ConnectionList
	cursor int
	err    error
}

func (s *stubClientInterface) List(ctx context.Context, sort live.SortMode) (live.ConnectionList, error) {
	if s.err != nil {
		return live.ConnectionList{}, s.err
	}
	if s.cursor >= len(s.lists) {
		<-ctx.Done()
		return live.ConnectionList{}, ctx.Err()
	}
	list := s.lists[s.cursor]
	s.cursor++
	return list, nil
}

type discardWriter struct{}

func (discardWriter) Write(history.Capture) error { return nil }
func (discardWriter) Close() error                { return nil }

func TestRunHeadlessCaptureContinuousModeReturnsNilOnContextCancel(t *testing.T) {
	src := &stubClientInterface{lists: []live.ConnectionList{{CapturedAt: time.UnixMilli(1)}}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := runHeadlessCapture(ctx, src, discardWriter{}, 0, time.Hour)

	if err != nil {
		t.Fatalf("runHeadlessCapture: %v", err)
	}
}

func TestRunHeadlessCaptureNoSamplesForFiniteDuration(t *testing.T) {
	src := &stubClientInterface{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := runHeadlessCapture(ctx, src, discardWriter{}, 10*time.Millisecond, time.Hour)

	if err == nil || !strings.Contains(err.Error(), "capture produced no samples") {
		t.Fatalf("err = %v, want no samples error", err)
	}
}

func TestRunHeadlessCaptureReturnsFirstSourceError(t *testing.T) {
	sourceErr := errors.New("source failed")
	src := &stubClientInterface{err: sourceErr}

	err := runHeadlessCapture(context.Background(), src, discardWriter{}, time.Second, time.Hour)

	if !errors.Is(err, sourceErr) {
		t.Fatalf("err = %v, want source error", err)
	}
}

func TestRunHeadlessCaptureJoinsCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	src := &stubClientInterface{lists: []live.ConnectionList{{CapturedAt: time.UnixMilli(1)}}}
	writer := &countingWriter{closeErr: closeErr}

	err := runHeadlessCapture(context.Background(), src, writer, 1*time.Millisecond, time.Hour)

	if !errors.Is(err, closeErr) {
		t.Fatalf("err = %v, want close error", err)
	}
}

func TestRunHeadlessCaptureWritesInstancesMetadata(t *testing.T) {
	list := live.ConnectionList{
		CapturedAt:  time.Date(2026, 4, 28, 15, 12, 4, 0, time.UTC),
		Connections: []live.Connection{},
		Sort:        live.SortByTransactionStart,
		Instances: []live.InstanceMeta{
			{ID: "primary", Role: "primary"},
		},
	}
	src := &stubClientInterface{lists: []live.ConnectionList{list}}

	var buf bytes.Buffer
	writer := history.NewCaptureWriter(&buf)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = runHeadlessCapture(ctx, src, writer, 0, 10*time.Millisecond)

	if !strings.Contains(buf.String(), `"instances":[{"id":"primary","role":"primary"}]`) {
		t.Fatalf("instances not found in capture output:\n%s", buf.String())
	}
}

type countingWriter struct {
	writes   int
	closeErr error
}

func (w *countingWriter) Write(history.Capture) error {
	w.writes++
	return nil
}

func (w *countingWriter) Close() error {
	return w.closeErr
}
