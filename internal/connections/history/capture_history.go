package history

import (
	"slices"

	"github.com/planetscale/cli/internal/connections"
)

// CaptureCursor identifies a capture stored in a CaptureHistory. Cursors are
// monotonic and stable across eviction — once a cursor's capture is evicted,
// At returns false for it, but subsequent captures keep advancing the cursor
// space so the model can hold a step position without recomputing on push.
type CaptureCursor int

// CaptureHistory is a capped ring of captured ConnectionLists. New captures
// push out the oldest once capacity is reached.
type CaptureHistory struct {
	capacity int
	samples  []connections.ConnectionList
	base     CaptureCursor
}

func NewCaptureHistory(capacity int) *CaptureHistory {
	if capacity < 1 {
		capacity = 1
	}
	return &CaptureHistory{capacity: capacity}
}

// Push stores list and returns its cursor. The Connections slice is cloned so
// callers can mutate (e.g. sort in place) without corrupting history.
func (h *CaptureHistory) Push(list connections.ConnectionList) CaptureCursor {
	list.Connections = slices.Clone(list.Connections)
	h.samples = append(h.samples, list)
	if len(h.samples) > h.capacity {
		h.samples = h.samples[len(h.samples)-h.capacity:]
		h.base += CaptureCursor(1)
	}
	return h.base + CaptureCursor(len(h.samples)-1)
}

func (h *CaptureHistory) At(cursor CaptureCursor) (connections.ConnectionList, bool) {
	idx := int(cursor - h.base)
	if idx < 0 || idx >= len(h.samples) {
		return connections.ConnectionList{}, false
	}
	return h.samples[idx], true
}

func (h *CaptureHistory) All() []connections.ConnectionList {
	out := make([]connections.ConnectionList, 0, len(h.samples))
	for _, list := range h.samples {
		list.Connections = slices.Clone(list.Connections)
		out = append(out, list)
	}
	return out
}

// Step moves cursor by delta, clamped to [Oldest, Latest]. Returns the new
// cursor and whether it moved. A cursor outside the live range is treated as
// "off the oldest edge" / "off the newest edge" — positive delta from below
// snaps to the oldest, negative from above snaps to the latest.
func (h *CaptureHistory) Step(cursor CaptureCursor, delta int) (CaptureCursor, bool) {
	if len(h.samples) == 0 || delta == 0 {
		return cursor, false
	}
	oldest := h.base
	latest := h.base + CaptureCursor(len(h.samples)-1)
	if cursor < oldest {
		if delta > 0 {
			return oldest, true
		}
		return cursor, false
	}
	if cursor > latest {
		if delta < 0 {
			return latest, true
		}
		return cursor, false
	}
	target := cursor + CaptureCursor(delta)
	if target < oldest {
		target = oldest
	}
	if target > latest {
		target = latest
	}
	return target, target != cursor
}

func (h *CaptureHistory) Oldest() (CaptureCursor, bool) {
	if len(h.samples) == 0 {
		return 0, false
	}
	return h.base, true
}

func (h *CaptureHistory) Latest() (CaptureCursor, bool) {
	if len(h.samples) == 0 {
		return 0, false
	}
	return h.base + CaptureCursor(len(h.samples)-1), true
}

// MustLatest panics when history is empty — convenient in tests where a
// caller has just pushed and knows the history is non-empty.
func (h *CaptureHistory) MustLatest() CaptureCursor {
	cursor, ok := h.Latest()
	if !ok {
		panic("CaptureHistory: MustLatest called on empty history")
	}
	return cursor
}

// Position returns the 1-based index of cursor in the live window and the
// total number of retained samples. Returns (0, total) when cursor has been
// evicted.
func (h *CaptureHistory) Position(cursor CaptureCursor) (n, total int) {
	total = len(h.samples)
	idx := int(cursor - h.base)
	if idx < 0 || idx >= total {
		return 0, total
	}
	return idx + 1, total
}

func (h *CaptureHistory) Len() int { return len(h.samples) }
