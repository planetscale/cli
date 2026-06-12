package tui

import (
	"fmt"

	live "github.com/planetscale/cli/internal/connections"
)

type blockerRow struct {
	Depth      int
	PID        int
	Instance   string
	Connection live.Connection
	Present    bool
	Cycle      bool
	Truncated  bool
	Remaining  int
	// WaitOn is the lock type the waiter on this edge is blocked on (e.g.
	// "Lock/transactionid"), so the wait-chain names what kind of lock the
	// blocked session is waiting for, not just who holds it.
	WaitOn string
}

const maxBlockerRows = 32

func blockerRows(list live.ConnectionList, root live.Connection) []blockerRow {
	byKey := connectionsByKey(list.Connections)
	return walkBlockerRows(byKey, root, func(c live.Connection) []int {
		return c.BlockedBy
	}, func(parent, child live.Connection) live.Connection {
		// Upstream: the parent waits on each child (the holder).
		return parent
	})
}

func blockingRows(list live.ConnectionList, root live.Connection) []blockerRow {
	byKey := connectionsByKey(list.Connections)
	downstream := make(map[int][]int)
	for _, conn := range list.Connections {
		for _, blockerPID := range conn.BlockedBy {
			downstream[blockerPID] = append(downstream[blockerPID], conn.PID)
		}
	}
	return walkBlockerRows(byKey, root, func(c live.Connection) []int {
		return downstream[c.PID]
	}, func(parent, child live.Connection) live.Connection {
		// Downstream: each child is the session waiting on the parent.
		return child
	})
}

func connectionsByKey(connections []live.Connection) map[int]live.Connection {
	byKey := make(map[int]live.Connection, len(connections))
	for _, conn := range connections {
		byKey[conn.PID] = conn
	}
	return byKey
}

func walkBlockerRows(byKey map[int]live.Connection, root live.Connection, nextPIDs func(live.Connection) []int, waiterFor func(parent, child live.Connection) live.Connection) []blockerRow {
	var rows []blockerRow
	path := map[int]bool{root.PID: true}
	var walk func(conn live.Connection, depth int)
	walk = func(conn live.Connection, depth int) {
		for _, pid := range nextPIDs(conn) {
			if len(rows) >= maxBlockerRows {
				rows = append(rows, blockerRow{Depth: depth, Truncated: true, Remaining: countBlockerLevels(byKey, conn, clonePath(path), nextPIDs)})
				return
			}
			row := blockerRow{Depth: depth, PID: pid}
			blocker, ok := byKey[pid]
			if ok {
				row.Connection = blocker
				row.Instance = blocker.Instance
				row.Present = true
				row.WaitOn = waitOnText(waiterFor(conn, blocker))
			}
			if path[pid] {
				row.Cycle = true
				rows = append(rows, row)
				continue
			}
			rows = append(rows, row)
			if ok {
				path[pid] = true
				walk(blocker, depth+1)
				delete(path, pid)
			}
		}
	}
	walk(root, 0)
	return collapseRootSuffixRows(rows)
}

func clonePath(path map[int]bool) map[int]bool {
	out := make(map[int]bool, len(path))
	for k, v := range path {
		out[k] = v
	}
	return out
}

func countBlockerLevels(byKey map[int]live.Connection, conn live.Connection, path map[int]bool, nextPIDs func(live.Connection) []int) int {
	count := 0
	for _, pid := range nextPIDs(conn) {
		count++
		blocker, ok := byKey[pid]
		if !ok || path[pid] {
			continue
		}
		path[pid] = true
		count += countBlockerLevels(byKey, blocker, path, nextPIDs)
		delete(path, pid)
	}
	return count
}

// collapseRootSuffixRows drops a root-level subtree when its blocker chain
// is a proper suffix of another root's chain. Without this, the tree double-
// renders the shared tail: e.g. when sessions A and B both lead to the same
// upstream C, we'd otherwise show [A→C, B→C, C] — operators end up scrolling
// past redundant chains in real lock-contention snapshots where many waiters
// converge on one lock holder.
func collapseRootSuffixRows(rows []blockerRow) []blockerRow {
	subtrees := rootSubtrees(rows)
	if len(subtrees) < 2 {
		return rows
	}
	signatures := make([][]int, len(subtrees))
	for i, subtree := range subtrees {
		signatures[i] = blockerRowSignature(subtree)
	}

	collapsed := make([]blockerRow, 0, len(rows))
	for i, subtree := range subtrees {
		keep := true
		for j, candidate := range signatures {
			if i == j {
				continue
			}
			if isProperIntSuffix(signatures[i], candidate) {
				keep = false
				break
			}
		}
		if keep {
			collapsed = append(collapsed, subtree...)
		}
	}
	return collapsed
}

func rootSubtrees(rows []blockerRow) [][]blockerRow {
	var subtrees [][]blockerRow
	for _, row := range rows {
		if row.Depth == 0 || len(subtrees) == 0 {
			subtrees = append(subtrees, []blockerRow{row})
			continue
		}
		last := len(subtrees) - 1
		subtrees[last] = append(subtrees[last], row)
	}
	return subtrees
}

func blockerRowSignature(rows []blockerRow) []int {
	signature := make([]int, 0, len(rows))
	for _, row := range rows {
		value := row.PID
		if row.Cycle {
			value = -value
		}
		signature = append(signature, value)
	}
	return signature
}

func isProperIntSuffix(value, candidate []int) bool {
	if len(value) == 0 || len(value) >= len(candidate) {
		return false
	}
	offset := len(candidate) - len(value)
	for i := range value {
		if value[i] != candidate[offset+i] {
			return false
		}
	}
	return true
}

func detailBlockerRows(list live.ConnectionList, root live.Connection) []blockerRow {
	rows := blockerRows(list, root)
	return append(rows, blockingRows(list, root)...)
}

// blockerLabel returns the text rendered for a node in the Blockers tab tree.
// PID / app / state are fixed-width so eye can column-scan; the trailing
// query preview is variable.
func blockerLabel(row blockerRow) string {
	if row.Truncated {
		return fmt.Sprintf("... (truncated, %d more levels)", row.Remaining)
	}
	if !row.Present {
		return fmt.Sprintf("%-7d  %s", row.PID, mutedStyle.Render("(session ended)"))
	}
	suffix := ""
	if row.Cycle {
		suffix = " " + mutedStyle.Render("(cycle)")
	}
	if row.WaitOn != "" {
		suffix = " " + mutedStyle.Render("("+row.WaitOn+")") + suffix
	}
	return fmt.Sprintf(
		"%-7d  %-14s  %-12s  %s%s",
		row.PID,
		clipLine(emptyDash(row.Connection.ApplicationName), 14),
		clipLine(emptyDash(stateText(row.Connection.State)), 12),
		queryPreview(row.Connection.QueryText),
		suffix,
	)
}

// waitOnText returns the lock type the connection is waiting on (e.g.
// "Lock/transactionid"), or "" when it is not waiting on anything.
func waitOnText(conn live.Connection) string {
	wait := waitText(conn)
	if wait == "-" {
		return ""
	}
	return wait
}
