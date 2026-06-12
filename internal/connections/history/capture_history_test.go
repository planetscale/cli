package history

import (
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/connections"
)

func TestCaptureHistoryAtReturnsPushedSample(t *testing.T) {
	c := qt.New(t)
	h := NewCaptureHistory(10)
	cursor := h.Push(listWithPID(101))

	list, ok := h.At(cursor)

	c.Assert(ok, qt.IsTrue)
	c.Assert(list.Connections[0].PID, qt.Equals, 101)
}

func TestCaptureHistoryEvictsOldestPastCapacity(t *testing.T) {
	c := qt.New(t)
	h := NewCaptureHistory(3)

	first := h.Push(listWithPID(100))
	h.Push(listWithPID(101))
	h.Push(listWithPID(102))
	h.Push(listWithPID(103))

	_, ok := h.At(first)
	c.Assert(ok, qt.IsFalse)
	c.Assert(h.Len(), qt.Equals, 3)
}

// Cursors remain valid across eviction: a stable identity lets the model pin a
// step position without recomputing on every Push.
func TestCaptureHistoryCursorsSurviveEvictionUntilEvicted(t *testing.T) {
	c := qt.New(t)
	h := NewCaptureHistory(2)

	first := h.Push(listWithPID(100))
	second := h.Push(listWithPID(101))
	h.Push(listWithPID(102))

	_, firstOK := h.At(first)
	list, secondOK := h.At(second)

	c.Assert(firstOK, qt.IsFalse)
	c.Assert(secondOK, qt.IsTrue)
	c.Assert(list.Connections[0].PID, qt.Equals, 101)
}

func TestCaptureHistoryStepMovesCursorWithinBounds(t *testing.T) {
	c := qt.New(t)
	h := NewCaptureHistory(10)
	first := h.Push(listWithPID(100))
	h.Push(listWithPID(101))
	third := h.Push(listWithPID(102))

	moved, ok := h.Step(first, 2)
	c.Assert(ok, qt.IsTrue)
	c.Assert(moved, qt.Equals, third)

	back, ok := h.Step(third, -2)
	c.Assert(ok, qt.IsTrue)
	c.Assert(back, qt.Equals, first)
}

func TestCaptureHistoryStepClampsAtBounds(t *testing.T) {
	c := qt.New(t)
	h := NewCaptureHistory(10)
	first := h.Push(listWithPID(100))
	h.Push(listWithPID(101))

	beforeOldest, ok := h.Step(first, -5)
	c.Assert(ok, qt.IsFalse)
	c.Assert(beforeOldest, qt.Equals, first)

	pastNewest, ok := h.Step(first, 99)
	c.Assert(ok, qt.IsTrue)
	c.Assert(pastNewest, qt.Equals, h.MustLatest())
}

func TestCaptureHistoryOldestLatestPosition(t *testing.T) {
	c := qt.New(t)
	h := NewCaptureHistory(10)
	first := h.Push(listWithPID(100))
	h.Push(listWithPID(101))
	third := h.Push(listWithPID(102))

	oldest, ok := h.Oldest()
	c.Assert(ok, qt.IsTrue)
	c.Assert(oldest, qt.Equals, first)

	latest, ok := h.Latest()
	c.Assert(ok, qt.IsTrue)
	c.Assert(latest, qt.Equals, third)

	pos, total := h.Position(first)
	c.Assert(pos, qt.Equals, 1)
	c.Assert(total, qt.Equals, 3)

	pos, total = h.Position(third)
	c.Assert(pos, qt.Equals, 3)
	c.Assert(total, qt.Equals, 3)
}

// Push must clone the Connections slice so callers (the TUI) can sort it in
// place without corrupting the stored history.
func TestCaptureHistoryPushClonesConnections(t *testing.T) {
	c := qt.New(t)
	h := NewCaptureHistory(10)
	input := listWithPID(100)
	cursor := h.Push(input)
	input.Connections[0].PID = 999

	stored, ok := h.At(cursor)
	c.Assert(ok, qt.IsTrue)
	c.Assert(stored.Connections[0].PID, qt.Equals, 100)
}

func TestCaptureHistoryAllReturnsSamplesInOrder(t *testing.T) {
	c := qt.New(t)
	h := NewCaptureHistory(2)
	h.Push(listWithPID(100))
	h.Push(listWithPID(101))
	h.Push(listWithPID(102))

	got := h.All()

	c.Assert(got, qt.HasLen, 2)
	c.Assert(got[0].Connections[0].PID, qt.Equals, 101)
	c.Assert(got[1].Connections[0].PID, qt.Equals, 102)

	got[0].Connections[0].PID = 999
	oldest, ok := h.Oldest()
	c.Assert(ok, qt.IsTrue)
	stored, ok := h.At(oldest)
	c.Assert(ok, qt.IsTrue)
	c.Assert(stored.Connections[0].PID, qt.Equals, 101)
}

func listWithPID(pid int) connections.ConnectionList {
	return connections.NewConnectionList(
		time.Date(2026, 5, 27, 12, 0, pid, 0, time.UTC),
		[]connections.Connection{{PID: pid, Instance: "primary"}},
		connections.SortByTransactionStart,
	)
}
