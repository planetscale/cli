package history

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/connections"

	qt "github.com/frankban/quicktest"
)

func TestReplaySourceReturnsLatestCapture(t *testing.T) {
	c := qt.New(t)
	var buffer bytes.Buffer
	writer := NewCaptureWriter(&buffer)

	at := time.Date(2026, 4, 28, 15, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		list := connections.NewConnectionList(at.Add(time.Duration(i)*time.Second), []connections.Connection{{
			PID:      100 + i,
			Instance: "primary",
		}}, connections.SortByTransactionStart)
		c.Assert(writer.Write(NewCapture(list)), qt.IsNil)
	}

	source, err := NewReplaySource(&buffer)
	c.Assert(err, qt.IsNil)

	list, err := source.List(context.Background(), connections.SortByTransactionStart)
	c.Assert(err, qt.IsNil)
	c.Assert(list.Connections, qt.HasLen, 1)
	c.Assert(list.Connections[0].PID, qt.Equals, 102)
}

func TestReplaySourceAppliesSortOnEachList(t *testing.T) {
	c := qt.New(t)
	var buffer bytes.Buffer
	writer := NewCaptureWriter(&buffer)
	at := time.Date(2026, 4, 28, 15, 0, 0, 0, time.UTC)
	xact1 := at.Add(-30 * time.Second)
	xact2 := at.Add(-10 * time.Second)
	list := connections.NewConnectionList(at, []connections.Connection{
		{PID: 10, Instance: "primary", State: "active", XactStart: &xact2, Duration: 1 * time.Second},
		{PID: 20, Instance: "primary", State: "active", XactStart: &xact1, Duration: 30 * time.Second},
	}, connections.SortByTransactionStart)
	c.Assert(writer.Write(NewCapture(list)), qt.IsNil)

	source, err := NewReplaySource(&buffer)
	c.Assert(err, qt.IsNil)

	byXact, err := source.List(context.Background(), connections.SortByTransactionStart)
	c.Assert(err, qt.IsNil)
	c.Assert(byXact.Connections[0].PID, qt.Equals, 20)
	c.Assert(byXact.Sort, qt.Equals, connections.SortByTransactionStart)

	byDuration, err := source.List(context.Background(), connections.SortByDuration)
	c.Assert(err, qt.IsNil)
	c.Assert(byDuration.Connections[0].PID, qt.Equals, 20)
	c.Assert(byDuration.Sort, qt.Equals, connections.SortByDuration)
}

// Replay must preserve InstanceMeta.Error so the partial-failure banner that
// reads it renders against captured data the same way it does live.
func TestReplaySourcePreservesInstancesMetadata(t *testing.T) {
	c := qt.New(t)
	var buffer bytes.Buffer
	writer := NewCaptureWriter(&buffer)
	at := time.Date(2026, 4, 28, 15, 0, 0, 0, time.UTC)
	list := connections.NewConnectionList(at, nil, connections.SortByTransactionStart)
	list.Instances = []connections.InstanceMeta{
		{ID: "primary", Role: "primary"},
		{ID: "replica-1", Role: "replica", Error: "connection refused"},
	}
	c.Assert(writer.Write(NewCapture(list)), qt.IsNil)

	source, err := NewReplaySource(&buffer)
	c.Assert(err, qt.IsNil)

	replayed, err := source.List(context.Background(), connections.SortByTransactionStart)
	c.Assert(err, qt.IsNil)
	c.Assert(replayed.Instances, qt.HasLen, 2)
	c.Assert(replayed.Instances[1].Error, qt.Equals, "connection refused")
}

func TestNewReplaySourceRejectsEmptyTrace(t *testing.T) {
	c := qt.New(t)
	_, err := NewReplaySource(strings.NewReader(""))
	c.Assert(err, qt.ErrorMatches, "capture file contains no replayable snapshots")
}

func TestNewReplaySourceRejectsHeaderOnlyTrace(t *testing.T) {
	c := qt.New(t)
	var buffer bytes.Buffer
	writer := NewCaptureWriter(&buffer)
	c.Assert(writer.WriteCaptureStart(CaptureStart{At: time.Now()}), qt.IsNil)

	_, err := NewReplaySource(&buffer)
	c.Assert(err, qt.ErrorMatches, "capture file contains no replayable snapshots")
}
