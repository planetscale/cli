package tui

import (
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
	live "github.com/planetscale/cli/internal/connections"
)

func TestBlockerRowsWalksUpstreamChain(t *testing.T) {
	c := qt.New(t)
	connections := []live.Connection{
		{PID: 10, BlockedBy: []int{20}, WaitEventType: "Lock", WaitEvent: "transactionid"},
		{PID: 20, BlockedBy: []int{30}},
		{PID: 30},
	}
	list := live.ConnectionList{Connections: connections}

	rows := blockerRows(list, connections[0])
	c.Assert(len(rows), qt.Equals, 2)
	c.Assert(rows[0].PID, qt.Equals, 20)
	c.Assert(rows[0].Depth, qt.Equals, 0)
	c.Assert(rows[0].Present, qt.IsTrue)
	c.Assert(rows[0].WaitOn, qt.Equals, "Lock/transactionid")
	c.Assert(rows[1].PID, qt.Equals, 30)
	c.Assert(rows[1].Depth, qt.Equals, 1)
}

func TestBlockerRowsMarksUnknownBlockerAbsent(t *testing.T) {
	c := qt.New(t)
	connections := []live.Connection{
		{PID: 10, BlockedBy: []int{99}},
	}
	list := live.ConnectionList{Connections: connections}

	rows := blockerRows(list, connections[0])
	c.Assert(len(rows), qt.Equals, 1)
	c.Assert(rows[0].PID, qt.Equals, 99)
	c.Assert(rows[0].Present, qt.IsFalse)
}

func TestBlockerRowsDetectsCycle(t *testing.T) {
	c := qt.New(t)
	connections := []live.Connection{
		{PID: 10, BlockedBy: []int{20}},
		{PID: 20, BlockedBy: []int{10}},
	}
	list := live.ConnectionList{Connections: connections}

	rows := blockerRows(list, connections[0])
	c.Assert(len(rows), qt.Equals, 2)
	c.Assert(rows[1].PID, qt.Equals, 10)
	c.Assert(rows[1].Cycle, qt.IsTrue)
}

func TestBlockingRowsWalksDownstream(t *testing.T) {
	c := qt.New(t)
	connections := []live.Connection{
		{PID: 30},
		{PID: 20, BlockedBy: []int{30}},
		{PID: 10, BlockedBy: []int{20}},
	}
	list := live.ConnectionList{Connections: connections}

	rows := blockingRows(list, connections[0])
	c.Assert(len(rows), qt.Equals, 2)
	c.Assert(rows[0].PID, qt.Equals, 20)
	c.Assert(rows[1].PID, qt.Equals, 10)
	c.Assert(rows[1].Depth, qt.Equals, 1)
}

func TestCollapseRootSuffixRowsDropsRedundantSubtrees(t *testing.T) {
	c := qt.New(t)
	rows := []blockerRow{
		{Depth: 0, PID: 30},
		{Depth: 0, PID: 20},
		{Depth: 1, PID: 30},
	}
	collapsed := collapseRootSuffixRows(rows)
	c.Assert(len(collapsed), qt.Equals, 2)
	c.Assert(collapsed[0].PID, qt.Equals, 20)
	c.Assert(collapsed[1].PID, qt.Equals, 30)
}

func TestWalkBlockerRowsTruncatesPastMaxDepth(t *testing.T) {
	c := qt.New(t)
	connections := make([]live.Connection, 0, maxBlockerRows+5)
	connections = append(connections, live.Connection{PID: 1, BlockedBy: []int{2}})
	for i := 2; i < maxBlockerRows+5; i++ {
		connections = append(connections, live.Connection{PID: i, BlockedBy: []int{i + 1}})
	}
	connections = append(connections, live.Connection{PID: maxBlockerRows + 5})
	list := live.ConnectionList{Connections: connections}

	rows := blockerRows(list, connections[0])
	c.Assert(len(rows) >= maxBlockerRows, qt.IsTrue)
	last := rows[len(rows)-1]
	c.Assert(last.Truncated, qt.IsTrue)
	c.Assert(last.Remaining > 0, qt.IsTrue)
}

func TestBlockerLabelDescribesAbsentAndCycleRows(t *testing.T) {
	c := qt.New(t)
	c.Assert(blockerLabel(blockerRow{PID: 99}), qt.Contains, "session ended")
	cycleRow := blockerRow{
		PID:        77,
		Present:    true,
		Cycle:      true,
		Connection: live.Connection{ApplicationName: "psql", State: "active", QueryText: "SELECT 1"},
	}
	c.Assert(blockerLabel(cycleRow), qt.Contains, "(cycle)")
	c.Assert(blockerLabel(cycleRow), qt.Contains, "psql")

	waitRow := blockerRow{
		PID:     242763,
		Present: true,
		WaitOn:  "Lock/transactionid",
		Connection: live.Connection{
			State:     "active",
			QueryText: "SELECT pg_sleep(86400);",
		},
	}
	c.Assert(stripANSI(blockerLabel(waitRow)), qt.Contains, "Lock/transactionid")

	truncatedRow := blockerRow{
		PID:     242763,
		Present: true,
		Connection: live.Connection{
			ApplicationName: "qa-block-holder",
			State:           "active",
			QueryText:       "SELECT pg_sleep(86400);",
		},
	}
	label := blockerLabel(truncatedRow)
	c.Assert(strings.Contains(label, "…"), qt.IsTrue)
	c.Assert(strings.Contains(label, "qa-block-holde "), qt.IsFalse)
}
