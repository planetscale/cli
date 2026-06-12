package tui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	qt "github.com/frankban/quicktest"
	"github.com/muesli/termenv"
	live "github.com/planetscale/cli/internal/connections"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

var tableRenderTestTime = time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func indexOf(slice []string, target string) int {
	for i, s := range slice {
		if s == target {
			return i
		}
	}
	return -1
}

func assertWidthAtMost(c *qt.C, value string, width int) {
	c.Helper()
	got := ansi.StringWidth(value)
	c.Assert(got <= width, qt.IsTrue, qt.Commentf("width=%d limit=%d value=%q", got, width, value))
}

func TestRenderTableEmptyAndLoadingStates(t *testing.T) {
	tests := []struct {
		name  string
		state tableState
		want  string
	}{
		{
			name:  "pre list",
			state: tableState{Width: 100, Height: 6},
			want:  "loading live connections...",
		},
		{
			name: "empty list",
			state: tableState{
				HasList: true,
				Width:   100,
				Height:  6,
			},
			want: "no live connections",
		},
		{
			name: "short body still shows rows",
			state: tableState{
				List:    live.NewConnectionList(tableRenderTestTime, []live.Connection{{PID: 1}, {PID: 2}, {PID: 3}}, live.SortByTransactionStart),
				HasList: true,
				Width:   100,
				Height:  4,
			},
			want: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			c.Assert(renderTable(tt.state), qt.Contains, tt.want)
		})
	}
}

func TestFooterHidesRowActionsOnEmptyList(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), nil, live.SortByTransactionStart)
	rendered := stripANSI(renderTable(tableState{
		List:    list,
		HasList: true,
		Width:   180,
		Height:  8,
	}))

	c.Assert(rendered, qt.Contains, "connections 0")
	c.Assert(rendered, qt.Contains, "no live connections")
	c.Assert(rendered, qt.Contains, "new connections appear on the next refresh")
	c.Assert(rendered, qt.Not(qt.Contains), "enter detail")
	c.Assert(rendered, qt.Not(qt.Contains), "cancel query")
	c.Assert(rendered, qt.Not(qt.Contains), "kill transaction")
	c.Assert(rendered, qt.Not(qt.Contains), "force terminate")
}

func TestRenderTableRendersNonEmptyList(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:             10,
		State:           "active",
		ApplicationName: "writer",
		QueryText:       "SELECT * FROM widgets",
	}}, live.SortByTransactionStart)

	rendered := renderTable(tableState{
		List:    list,
		HasList: true,
		Width:   180,
		Height:  6,
	})

	c.Assert(rendered, qt.Contains, "connections 1")
	c.Assert(rendered, qt.Contains, "PID")
	c.Assert(rendered, qt.Contains, "SELECT * FROM widgets")
}

func TestRenderTableInitialErrorDoesNotRepeatInFooter(t *testing.T) {
	c := qt.New(t)
	rendered := stripANSI(renderTable(tableState{
		LastError: "instances unreachable",
		Width:     120,
		Height:    8,
	}))

	c.Assert(strings.Count(rendered, "instances unreachable"), qt.Equals, 1)
	c.Assert(rendered, qt.Not(qt.Contains), "error: instances unreachable")
}

func TestRenderTableFitsTerminalHeightWithBanner(t *testing.T) {
	c := qt.New(t)
	conns := make([]live.Connection, 0, 8)
	for i := 0; i < 8; i++ {
		conns = append(conns, live.Connection{PID: i + 1, QueryText: "SELECT 1"})
	}
	list := live.NewConnectionList(time.Now(), conns, live.SortByTransactionStart)
	list.Instances = []live.InstanceMeta{
		{ID: "primary", Role: "primary"},
		{ID: "replica-1", Role: "replica"},
		{ID: "replica-2", Role: "replica", Error: "timeout"},
	}

	rendered := renderTable(tableState{
		List:    list,
		HasList: true,
		Width:   180,
		Height:  14,
	})
	lines := strings.Split(rendered, "\n")

	c.Assert(len(lines), qt.Equals, 14, qt.Commentf("rendered %d lines for height 14:\n%s", len(lines), rendered))
	c.Assert(lines[0], qt.Contains, "connections 8")
	c.Assert(lines[len(lines)-1], qt.Contains, "q")
	c.Assert(lines[len(lines)-1], qt.Contains, "quit")
	c.Assert(rendered, qt.Not(qt.Contains), "role marker")
	c.Assert(rendered, qt.Not(qt.Contains), "BLOCK: digit")
}

func TestRenderTablePinsFooterToViewportBottom(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:       10,
		State:     "active",
		QueryText: "SELECT * FROM widgets",
	}}, live.SortByTransactionStart)

	rendered := renderTable(tableState{
		List:    list,
		HasList: true,
		Width:   180,
		Height:  10,
	})
	lines := strings.Split(rendered, "\n")

	c.Assert(lines[len(lines)-1], qt.Contains, "q")
	c.Assert(lines[len(lines)-1], qt.Contains, "quit")
	c.Assert(rendered, qt.Not(qt.Contains), "role marker")
	c.Assert(rendered, qt.Not(qt.Contains), "BLOCK: digit")
}

func TestRenderFooterOmitsInstanceRoleLegend(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10}}, live.SortByTransactionStart)
	got := renderFooter(tableState{List: list, HasList: true, Width: 60, Height: 24})

	c.Assert(stripANSI(got), qt.Not(qt.Contains), "role marker")
	c.Assert(stripANSI(got), qt.Not(qt.Contains), "W=waiting on lock")
	for _, line := range strings.Split(stripANSI(got), "\n") {
		c.Assert(lipgloss.Width(line) <= 60, qt.IsTrue)
	}
}

func TestRenderFooterStaysWithinVeryNarrowWidthWithoutLegend(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10}}, live.SortByTransactionStart)
	got := renderFooter(tableState{List: list, HasList: true, Width: 8, Height: 24})

	for _, line := range strings.Split(stripANSI(got), "\n") {
		c.Assert(lipgloss.Width(line) <= 8, qt.IsTrue)
	}
}

func TestRenderTableShowsSelectedQueryStatus(t *testing.T) {
	c := qt.New(t)
	query := "SELECT id, owner_id FROM public.events WHERE owner_id = $1 ORDER BY id LIMIT 300"
	list := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 10, QueryText: "SELECT 1"},
		{PID: 20, QueryText: query},
	}, live.SortByTransactionStart)

	rendered := renderTable(tableState{
		List:     list,
		HasList:  true,
		Selected: 1,
		Width:    140,
		Height:   10,
	})

	c.Assert(selectedStatusLine(c, rendered), qt.Equals, "selected pid 20 | "+query)
}

func TestRenderTableClipsSelectedQueryStatus(t *testing.T) {
	c := qt.New(t)
	query := "SELECT id, owner_id, created_at FROM public.events WHERE owner_id = $1 ORDER BY created_at DESC LIMIT 300"
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:       39123,
		QueryText: query,
	}}, live.SortByTransactionStart)

	rendered := renderTable(tableState{
		List:     list,
		HasList:  true,
		Selected: 0,
		Width:    72,
		Height:   10,
	})
	status := selectedStatusLine(c, rendered)

	c.Assert(lipgloss.Width(status) <= 72, qt.IsTrue)
	c.Assert(status, qt.Contains, "selected pid 39123 | SELECT id")
	c.Assert(status, qt.Contains, "…")
}

func TestBuildConnectionRowsUsesCoreColumns(t *testing.T) {
	c := qt.New(t)

	headers, rows := buildConnectionRows([]live.Connection{{PID: 10}}, nil, 200, -1)

	c.Assert(headers, qt.DeepEquals, []string{"", "PID", "STATE", "BLOCK", "WAIT", "DURATION", "APP", "START", "QUERY"})
	c.Assert(rows, qt.HasLen, 1)
	c.Assert(rows[0], qt.HasLen, len(headers))
}

func TestBuildConnectionRowsProcesslistDisplay(t *testing.T) {
	c := qt.New(t)
	headers, rows := buildConnectionRowsForDisplay(connectionDisplayProcesslist, []live.Connection{{
		Instance:     "zone1-2001",
		PID:          101,
		State:        "Query/executing",
		Duration:     42 * time.Second,
		Username:     "vt_app",
		DatabaseName: "checkout",
		QueryText:    "SELECT 1",
	}}, nil, 120, 0)

	c.Assert(headers, qt.DeepEquals, []string{"", "PID", "TABLET", "STATE", "DURATION", "USER", "DB", "QUERY"})
	c.Assert(stripANSI(rows[0][1]), qt.Equals, "101")
	c.Assert(stripANSI(rows[0][2]), qt.Equals, "zone1-2001")
	c.Assert(stripANSI(rows[0][3]), qt.Equals, "Query")
	c.Assert(stripANSI(rows[0][5]), qt.Equals, "vt_app")
	c.Assert(stripANSI(rows[0][6]), qt.Equals, "checkout")
	c.Assert(strings.Join(headers, ","), qt.Not(qt.Contains), "BLOCK")
	c.Assert(strings.Join(headers, ","), qt.Not(qt.Contains), "WAIT")
	c.Assert(strings.Join(headers, ","), qt.Not(qt.Contains), "APP")
}

func TestBuildConnectionRowsProcesslistDisplayShowsZeroDurationActiveQuery(t *testing.T) {
	c := qt.New(t)
	headers, rows := buildConnectionRowsForDisplay(connectionDisplayProcesslist, []live.Connection{{
		PID:       101,
		State:     "Query/update",
		QueryText: "INSERT INTO events VALUES (1)",
	}}, nil, 120, 0)

	durationIdx := indexOf(headers, "DURATION")
	c.Assert(durationIdx, qt.Not(qt.Equals), -1)
	c.Assert(stripANSI(rows[0][durationIdx]), qt.Equals, "00:00")
}

func TestBuildConnectionRowsWidensAppColumnWhenSpaceAllows(t *testing.T) {
	c := qt.New(t)

	headers, rows := buildConnectionRows([]live.Connection{{
		PID:             10,
		ApplicationName: "interactive_client_47",
		QueryText:       "SELECT 1",
	}}, nil, 260, -1)

	appIdx := indexOf(headers, "APP")
	c.Assert(appIdx, qt.Not(qt.Equals), -1)
	c.Assert(stripANSI(rows[0][appIdx]), qt.Equals, "interactive_client_47")
}

func TestRenderConnectionTableProcesslistDisplayStaysLeftAligned(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		Instance:     "zone1-2001",
		PID:          101,
		State:        "Query/executing",
		Duration:     42 * time.Second,
		Username:     "vt_app",
		DatabaseName: "checkout",
		QueryText:    "SELECT 1",
	}}, live.SortByDuration)

	rendered := stripANSI(renderConnectionTable(tableState{
		List:          list,
		HasList:       true,
		Selected:      0,
		Width:         200,
		DisplayPreset: connectionDisplayProcesslist,
	}, 10))
	row := lineContaining(c, rendered, "SELECT 1")

	c.Assert(strings.Index(row, "101") <= 4, qt.IsTrue, qt.Commentf("row = %q", row))
	c.Assert(strings.HasPrefix(row, "▶"), qt.IsTrue, qt.Commentf("row = %q", row))
}

func TestRenderConnectionTableProcesslistDisplayStylesStates(t *testing.T) {
	c := qt.New(t)
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(prev)
	list := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 101, State: "Query/executing", QueryText: "SELECT 1"},
		{PID: 102, State: "Sleep"},
	}, live.SortByDuration)

	rendered := renderConnectionTable(tableState{
		List:          list,
		HasList:       true,
		Selected:      -1,
		Width:         120,
		DisplayPreset: connectionDisplayProcesslist,
	}, 10)
	queryRow := lineContaining(c, rendered, "SELECT 1")
	sleepRow := lineContaining(c, rendered, "102")

	c.Assert(queryRow, qt.Contains, "\x1b[")
	c.Assert(sleepRow, qt.Contains, "\x1b[")
}

func TestProcesslistHighlightedRowsRenderContiguousBackground(t *testing.T) {
	c := qt.New(t)
	prevProfile := lipgloss.ColorProfile()
	prevBackground := lipgloss.HasDarkBackground()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(prevProfile)
	defer lipgloss.SetHasDarkBackground(prevBackground)

	lipgloss.SetHasDarkBackground(true)
	list := live.NewConnectionList(tableRenderTestTime, []live.Connection{{
		PID:          101,
		Instance:     "zone1-2001",
		State:        "Query/update",
		Username:     "vt_app",
		DatabaseName: "checkout",
		QueryText:    "INSERT INTO events VALUES (1)",
	}}, live.SortByDuration)

	rendered := renderConnectionTable(tableState{
		List:          list,
		HasList:       true,
		Selected:      0,
		Width:         160,
		DisplayPreset: connectionDisplayProcesslist,
	}, 10)
	selectedRow := lineContaining(c, rendered, "INSERT INTO events")

	c.Assert(selectedRow, qt.Not(qt.Contains), "\x1b[0m \x1b[", qt.Commentf("row = %q", selectedRow))
}

func TestBuildConnectionRowsFormatsWaitAndStart(t *testing.T) {
	c := qt.New(t)
	xactStart := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	queryStart := time.Date(2026, 4, 29, 12, 1, 0, 0, time.UTC)

	_, rows := buildConnectionRows([]live.Connection{{
		PID:             10,
		InstanceRole:    "primary",
		State:           "active",
		WaitEventType:   "Lock",
		WaitEvent:       "tuple",
		ApplicationName: "writer",
		XactStart:       &xactStart,
		QueryStart:      &queryStart,
		QueryText:       "SELECT * FROM widgets",
	}}, nil, 200, -1)

	// STATE and WAIT are right-padded to a stable min width; trim for content
	// comparison.
	got := rows[0]
	got[2] = strings.TrimRight(got[2], " ")
	got[4] = strings.TrimRight(got[4], " ")
	c.Assert(got, qt.DeepEquals, []string{
		"  ",
		"10",
		"active",
		"-    ",
		"Lock/tuple",
		"-",
		"writer",
		"12:00:00",
		"SELECT * FROM widgets",
	})
}

func TestBuildConnectionRowsRoleMarkerLeftmostColumn(t *testing.T) {
	c := qt.New(t)
	connections := []live.Connection{
		{PID: 1, Instance: "primary", InstanceRole: "primary", State: "active"},
		{PID: 2, Instance: "replica-a", InstanceRole: "replica", State: "idle"},
		{PID: 3, Instance: "ghost", InstanceRole: "", State: "idle"},
	}

	_, rows := buildConnectionRows(connections, nil, 200, -1)

	c.Assert(rows[0][0], qt.Equals, "  ")
	c.Assert(rows[1][0], qt.Equals, " R")
	c.Assert(rows[2][0], qt.Equals, "  ")
}

func TestBuildConnectionRowsWaitColumnCombinesTypeAndEvent(t *testing.T) {
	c := qt.New(t)

	connections := []live.Connection{
		{PID: 1, WaitEventType: "Lock", WaitEvent: "transactionid"},
		{PID: 2, WaitEventType: "Client", WaitEvent: "ClientRead"},
		{PID: 3, WaitEvent: "ClientRead"},
		{PID: 4, WaitEventType: "IPC"},
		{PID: 5},
	}
	headers, rows := buildConnectionRows(connections, nil, 200, -1)

	waitIdx := indexOf(headers, "WAIT")
	c.Assert(waitIdx, qt.Not(qt.Equals), -1)
	// WAIT cells are right-padded to a stable min width; compare trimmed.
	c.Assert(strings.TrimRight(rows[0][waitIdx], " "), qt.Equals, "Lock/transactionid")
	c.Assert(strings.TrimRight(rows[1][waitIdx], " "), qt.Equals, "Client/ClientRead")
	c.Assert(strings.TrimRight(rows[2][waitIdx], " "), qt.Equals, "ClientRead")
	c.Assert(strings.TrimRight(rows[3][waitIdx], " "), qt.Equals, "IPC")
	c.Assert(strings.TrimRight(rows[4][waitIdx], " "), qt.Equals, "-")

	wait := waitTextForWidth(live.Connection{
		WaitEventType: "Client",
		WaitEvent:     "ClientRead",
	}, 80)
	c.Assert(strings.Contains(wait, "…"), qt.IsTrue)
	c.Assert(strings.Contains(wait, "..."), qt.IsFalse)
}

func TestBuildConnectionRowsNarrowKeepsWaitOverLowerPriorityColumns(t *testing.T) {
	c := qt.New(t)
	headers, rows := buildConnectionRows([]live.Connection{{
		PID:           10,
		State:         "idle in transaction",
		Duration:      90 * time.Second,
		WaitEventType: "Client",
		WaitEvent:     "ClientRead",
	}}, nil, 80, -1)

	c.Assert(indexOf(headers, "STATE"), qt.Not(qt.Equals), -1)
	c.Assert(indexOf(headers, "DURATION"), qt.Not(qt.Equals), -1)
	c.Assert(indexOf(headers, "BLOCK"), qt.Not(qt.Equals), -1)
	waitIdx := indexOf(headers, "WAIT")
	c.Assert(waitIdx, qt.Not(qt.Equals), -1)
	assertWidthAtMost(c, rows[0][waitIdx], 10)
}

func TestStateTextAbbreviatesIdleInTransaction(t *testing.T) {
	c := qt.New(t)
	c.Assert(stateText("idle"), qt.Equals, "idle")
	c.Assert(stateText("active"), qt.Equals, "active")
	c.Assert(stateText("idle in transaction"), qt.Equals, "idle/xact")
	c.Assert(stateText("idle in transaction (aborted)"), qt.Equals, "idle/xact (aborted)")
	c.Assert(stateText(""), qt.Equals, "")
}

func TestHighlightedRowsRenderContiguousBackground(t *testing.T) {
	c := qt.New(t)
	prevProfile := lipgloss.ColorProfile()
	prevBackground := lipgloss.HasDarkBackground()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(prevProfile)
	defer lipgloss.SetHasDarkBackground(prevBackground)

	lipgloss.SetHasDarkBackground(true)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:       10,
		QueryText: "SELECT 1",
	}}, live.SortByTransactionStart)

	rendered := renderConnectionTable(tableState{List: list, HasList: true, Selected: 0, Width: 120}, 10)
	selectedRow := lineContaining(c, rendered, "SELECT 1")

	c.Assert(selectedRow, qt.Not(qt.Contains), "\x1b[0m \x1b[")
}

func lineContaining(c *qt.C, text, needle string) string {
	c.Helper()
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	c.Fatalf("no line containing %q in:\n%s", needle, text)
	return ""
}

func TestAppTextRightTruncatesLongNames(t *testing.T) {
	c := qt.New(t)
	c.Assert(appText(""), qt.Equals, "-")
	c.Assert(appText("psql"), qt.Equals, "psql")
	c.Assert(appText("owner_writer_226"), qt.Equals, "owner_writer_2…")
	c.Assert(appText("interactive_client_47"), qt.Equals, "interactive_c…")
	c.Assert(appText("xxxxxxxxxxxxxx"), qt.Equals, "xxxxxxxxxxxxxx")
}

func TestBuildConnectionRowsMarksSelectedRow(t *testing.T) {
	c := qt.New(t)
	connections := []live.Connection{
		{PID: 10, InstanceRole: "primary"},
		{PID: 20, InstanceRole: "replica"},
		{PID: 30, InstanceRole: "unknown"},
	}

	headers, rows := buildConnectionRows(connections, nil, 200, 1)

	markerIdx := indexOf(headers, "")
	pidIdx := indexOf(headers, "PID")
	c.Assert(markerIdx, qt.Equals, 0)
	c.Assert(pidIdx, qt.Equals, 1)
	c.Assert(rows[0][markerIdx], qt.Equals, "  ")
	c.Assert(rows[0][pidIdx], qt.Equals, "10")
	c.Assert(rows[1][markerIdx], qt.Equals, "▶R")
	c.Assert(rows[1][pidIdx], qt.Equals, "20")
	c.Assert(rows[2][markerIdx], qt.Equals, "  ")
	c.Assert(rows[2][pidIdx], qt.Equals, "30")

	_, rows = buildConnectionRows(connections, nil, 200, 2)
	c.Assert(rows[2][markerIdx], qt.Equals, "▶ ")
	c.Assert(rows[2][pidIdx], qt.Equals, "30")
}

func TestBuildConnectionRowsSeparatesSelectedReplicaMarker(t *testing.T) {
	c := qt.New(t)
	_, rows := buildConnectionRows([]live.Connection{{PID: 10, InstanceRole: "replica"}}, nil, 120, 0)

	c.Assert(stripANSI(rows[0][0]), qt.Equals, "▶R")
	c.Assert(lipgloss.Width(stripANSI(rows[0][0])), qt.Equals, 2)
}

func TestBlockedTextRendersBlockedAndDownstream(t *testing.T) {
	c := qt.New(t)
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(prev)

	// Count-first encoding so the alarm digit leads. "W" appears alone for
	// pure-victim sessions, as a suffix when the connection both blocks and
	// waits. All cells right-padded to blockColumnMinWidth so the column
	// renders at least as wide as the BLOCK header.
	c.Assert(blockedText(live.Connection{}, 0), qt.Equals, "-    ")
	c.Assert(blockedText(live.Connection{BlockedBy: []int{99}}, 0), qt.Equals, "W    ")
	c.Assert(blockedText(live.Connection{}, 3), qt.Equals, "3    ")
	c.Assert(blockedText(live.Connection{BlockedBy: []int{99}}, 3), qt.Equals, "3 W  ")
}

func TestBuildConnectionRowsDropsStartOnNarrowTerminals(t *testing.T) {
	c := qt.New(t)
	conns := []live.Connection{{PID: 1}}

	wideHeaders, _ := buildConnectionRows(conns, nil, 180, -1)
	narrowHeaders, _ := buildConnectionRows(conns, nil, 120, -1)

	c.Assert(indexOf(wideHeaders, "START"), qt.Not(qt.Equals), -1)
	c.Assert(indexOf(narrowHeaders, "START"), qt.Equals, -1)
}

func TestRenderCapturedToken(t *testing.T) {
	tests := []struct {
		name     string
		state    tableState
		contains []string
		omits    []string
	}{
		{
			name: "same day omits date",
			state: tableState{
				List:     live.ConnectionList{CapturedAt: time.Date(2026, 5, 8, 21, 18, 14, 0, time.Local)},
				Now:      time.Date(2026, 5, 8, 21, 18, 14, 0, time.Local),
				Interval: time.Second,
			},
			contains: []string{"21:18:14"},
			omits:    []string{"2026-05-08"},
		},
		{
			name: "different day includes date",
			state: tableState{
				List:     live.ConnectionList{CapturedAt: time.Date(2026, 5, 7, 21, 18, 14, 0, time.Local)},
				Now:      time.Date(2026, 5, 8, 9, 0, 0, 0, time.Local),
				Interval: time.Second,
			},
			contains: []string{"2026-05-07 21:18:14"},
		},
		{
			name: "paused keeps age",
			state: tableState{
				List:     live.ConnectionList{CapturedAt: tableRenderTestTime},
				Now:      tableRenderTestTime.Add(30 * time.Second),
				Interval: time.Second,
				Paused:   true,
			},
			contains: []string{"captured ", "(30s ago)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			got := stripANSI(renderCapturedToken(tt.state))
			for _, want := range tt.contains {
				c.Assert(got, qt.Contains, want)
			}
			for _, notWant := range tt.omits {
				c.Assert(got, qt.Not(qt.Contains), notWant)
			}
		})
	}
}

func TestRenderHeaderStylesPausedToken(t *testing.T) {
	c := qt.New(t)
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(prev)

	got := renderHeader(tableState{Paused: true, Sort: live.SortByTransactionStart})

	c.Assert(stripANSI(got), qt.Contains, "paused")
	c.Assert(got, qt.Not(qt.Equals), stripANSI(got))
}

func TestRenderHeaderKeepsFreshnessAndPausedStateWhenNarrow(t *testing.T) {
	c := qt.New(t)

	now := tableRenderTestTime.Add(5 * time.Second)
	state := tableState{
		Target: Target{
			Database: "kind-live-connections",
			Branch:   "main",
		},
		List:          live.NewConnectionList(tableRenderTestTime, []live.Connection{{PID: 1}, {PID: 2}, {PID: 3}, {PID: 4}}, live.SortByTransactionStart),
		HasList:       true,
		Sort:          live.SortByTransactionStart,
		Now:           now,
		Interval:      time.Second,
		CaptureStatus: "rec off",
		Paused:        true,
		StepPos:       237,
		StepTotal:     300,
		Width:         80,
	}

	header := stripANSI(renderHeader(state))

	assertWidthAtMost(c, header, 80)
	c.Assert(strings.Contains(header, "captured "), qt.IsTrue)
	c.Assert(strings.Contains(header, "ago)"), qt.IsTrue)
	c.Assert(strings.Contains(header, "paused"), qt.IsTrue)
	c.Assert(strings.Contains(header, "step 237/300"), qt.IsTrue)
	c.Assert(strings.Contains(header, "…"), qt.IsTrue)
}

func TestRenderHeaderKeepsCompactVitessTargetWhenNarrow(t *testing.T) {
	c := qt.New(t)
	captured := time.Date(2026, 4, 29, 1, 3, 9, 0, time.Local)
	state := tableState{
		Target: Target{
			Database: "kind-live-connections-mysql",
			Branch:   "main",
			Keyspace: "planetscale",
			Shard:    "-",
		},
		List:          live.NewConnectionList(captured, make([]live.Connection, 14), live.SortByDuration),
		HasList:       true,
		Sort:          live.SortByDuration,
		CanSort:       false,
		Now:           captured,
		Interval:      time.Second,
		CaptureStatus: "rec off",
		Width:         80,
	}

	header := stripANSI(renderHeader(state))

	assertWidthAtMost(c, header, 80)
	c.Assert(header, qt.Equals, "kind-live-connection… / main / planetscale/- | ● 14 | captured 01:03:09 (0s ago)")

	state.Width = 40
	header = stripANSI(renderHeader(state))

	assertWidthAtMost(c, header, 40)
	c.Assert(header, qt.Contains, " / main | ● 14 | (0s)")
	c.Assert(header, qt.Contains, "…")
	c.Assert(header, qt.Not(qt.Contains), "planetscale")
}

func TestRenderHeaderVeryNarrowKeepsProtectedTokens(t *testing.T) {
	c := qt.New(t)
	captured := time.Date(2026, 4, 29, 1, 3, 9, 0, time.Local)
	state := tableState{
		Target: Target{
			Database: "kind-live-connections-mysql",
			Branch:   "very-long-branch-name",
			Keyspace: "planetscale",
			Shard:    "-",
		},
		List:     live.NewConnectionList(captured, make([]live.Connection, 14), live.SortByDuration),
		HasList:  true,
		Sort:     live.SortByDuration,
		CanSort:  false,
		Now:      captured,
		Interval: time.Second,
		Paused:   true,
		Width:    24,
	}

	header := stripANSI(renderHeader(state))

	assertWidthAtMost(c, header, 24)
	c.Assert(header, qt.Contains, "● 14")
	c.Assert(header, qt.Contains, "paused")
	c.Assert(header, qt.Contains, "(0s)")
}

func TestRenderHeaderNarrowPostgresKeepsTarget(t *testing.T) {
	c := qt.New(t)
	captured := time.Date(2026, 4, 29, 1, 3, 9, 0, time.Local)
	state := tableState{
		Target: Target{
			Database: "kind-live-connections-postgres",
			Branch:   "main",
		},
		List:          live.NewConnectionList(captured, []live.Connection{{PID: 1}, {PID: 2}}, live.SortByTransactionStart),
		HasList:       true,
		Sort:          live.SortByTransactionStart,
		CanSort:       true,
		Now:           captured,
		Interval:      time.Second,
		CaptureStatus: "rec off",
	}

	for _, width := range []int{40, 80, 120} {
		state.Width = width
		header := stripANSI(renderHeader(state))

		assertWidthAtMost(c, header, width)
		c.Assert(header, qt.Contains, " / main")
		if width < 120 {
			c.Assert(header, qt.Contains, "● 2")
		} else {
			c.Assert(header, qt.Contains, "● connections 2")
		}
		c.Assert(header, qt.Contains, "(0s")
	}
}

func TestRenderHeaderLabelsSingleSortAsStatic(t *testing.T) {
	c := qt.New(t)
	state := tableState{
		List:    live.NewConnectionList(tableRenderTestTime, []live.Connection{{PID: 1}}, live.SortByDuration),
		HasList: true,
		Sort:    live.SortByDuration,
		CanSort: false,
		Width:   120,
	}

	header := stripANSI(renderHeader(state))

	c.Assert(header, qt.Contains, "sorted by duration")
	c.Assert(header, qt.Not(qt.Contains), "sort duration")
}

func TestRenderHeaderBoundsLongFilterChipWhenNarrow(t *testing.T) {
	c := qt.New(t)

	now := tableRenderTestTime.Add(5 * time.Second)
	state := tableState{
		List:      live.NewConnectionList(tableRenderTestTime, []live.Connection{{PID: 1}}, live.SortByTransactionStart),
		HasList:   true,
		Sort:      live.SortByTransactionStart,
		Now:       now,
		Interval:  time.Second,
		Filter:    "filter: instance qa-replica-with-a-very-long-generated-name",
		Paused:    true,
		StepPos:   237,
		StepTotal: 300,
		Width:     80,
	}

	header := stripANSI(renderHeader(state))

	assertWidthAtMost(c, header, 80)
	c.Assert(strings.Contains(header, "filter:"), qt.IsTrue)
	c.Assert(strings.Contains(header, "captured "), qt.IsTrue)
	c.Assert(strings.Contains(header, "paused"), qt.IsTrue)
	c.Assert(strings.Contains(header, "step 237/300"), qt.IsTrue)
	c.Assert(strings.Contains(header, "…"), qt.IsTrue)
}

func TestFreshnessTierFor(t *testing.T) {
	c := qt.New(t)
	captured := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	interval := time.Second

	c.Assert(freshnessTierFor(captured, captured.Add(2*time.Second), interval), qt.Equals, freshnessFresh)
	c.Assert(freshnessTierFor(captured, captured.Add(3*time.Second-time.Millisecond), interval), qt.Equals, freshnessFresh)
	c.Assert(freshnessTierFor(captured, captured.Add(3*time.Second), interval), qt.Equals, freshnessStale)
	c.Assert(freshnessTierFor(captured, captured.Add(9*time.Second), interval), qt.Equals, freshnessStale)
	c.Assert(freshnessTierFor(captured, captured.Add(10*time.Second), interval), qt.Equals, freshnessVeryStale)
	c.Assert(freshnessTierFor(time.Time{}, captured, interval), qt.Equals, freshnessFresh)
	c.Assert(freshnessTierFor(captured, captured.Add(time.Hour), 0), qt.Equals, freshnessFresh)
}

func TestRenderHelpIncludesCurrentBindings(t *testing.T) {
	c := qt.New(t)

	rendered := renderHelp(180)

	for _, binding := range defaultBindings().ShortHelp() {
		help := binding.Help()
		c.Assert(rendered, qt.Contains, help.Key)
		c.Assert(rendered, qt.Contains, help.Desc)
	}

	bindings := defaultBindings()
	c.Assert(bindings.Navigate.Help().Desc, qt.Equals, "select")
	c.Assert(bindings.Cancel.Help().Desc, qt.Equals, "cancel query")
	c.Assert(bindings.TerminateTxn.Help().Desc, qt.Equals, "kill transaction")
	c.Assert(bindings.TerminateConn.Help().Desc, qt.Equals, "force terminate")
}

func TestRenderHelpDocumentsVDetailAlias(t *testing.T) {
	c := qt.New(t)

	footer := stripANSI(renderHelp(180))
	modal := stripANSI(renderHelpModal(helpState{
		Target:       Target{Database: "prod", Branch: "main"},
		Width:        120,
		Height:       40,
		CanSort:      true,
		Capabilities: DefaultConnectionCapabilities(),
	}))

	c.Assert(footer, qt.Contains, "enter/v detail")
	c.Assert(modal, qt.Contains, "enter/v detail")
}

func TestRenderHelpModalHidesRefreshWhilePaused(t *testing.T) {
	c := qt.New(t)
	modal := stripANSI(renderHelpModal(helpState{
		Target:       Target{Database: "prod", Branch: "main"},
		Width:        120,
		Height:       40,
		Paused:       true,
		Capabilities: DefaultConnectionCapabilities(),
	}))

	c.Assert(modal, qt.Contains, "space resume")
	c.Assert(modal, qt.Not(qt.Contains), "r refresh")
}

func TestRenderHelpShowsShiftKForForceTerminate(t *testing.T) {
	c := qt.New(t)
	footer := stripANSI(renderHelp(160))

	c.Assert(footer, qt.Contains, "shift+K force terminate")
}

func TestRenderHelpHidesUnavailableActions(t *testing.T) {
	c := qt.New(t)
	footer := stripANSI(renderHelpFor(160, false, false, connectionDisplayProcesslist, ConnectionCapabilities{
		CancelQuery:         ActionTargetQueryID,
		TerminateConnection: ActionTargetConnectionID,
		ShowBlockers:        false,
	}, true, true, true, true, false))

	c.Assert(footer, qt.Contains, "c KILL QUERY")
	c.Assert(footer, qt.Contains, "shift+K KILL")
	c.Assert(footer, qt.Not(qt.Contains), "kill transaction")
}

func TestRenderHelpUsesOperatorActionCopy(t *testing.T) {
	c := qt.New(t)

	help := stripANSI(renderHelpModal(helpState{
		Target:       Target{Database: "prod", Branch: "main"},
		Width:        120,
		Height:       40,
		CanSort:      true,
		Capabilities: DefaultConnectionCapabilities(),
	}))

	c.Assert(help, qt.Contains, "Actions")
	c.Assert(help, qt.Not(qt.Contains), "Safe Actions")
	c.Assert(help, qt.Contains, "c  Cancel the selected query (pg_cancel_backend)")
	c.Assert(help, qt.Contains, "k  Kill the selected transaction (pg_terminate_backend)")
	c.Assert(help, qt.Contains, "K  Force terminate the selected connection (pg_terminate_backend)")
	c.Assert(help, qt.Contains, "c, k, and K require confirmation. Replay mode blocks backend actions.")
}

func TestVisibleConnectionsLimitsRowsFromTop(t *testing.T) {
	c := qt.New(t)
	connections := []live.Connection{{PID: 1}, {PID: 2}, {PID: 3}, {PID: 4}, {PID: 5}}

	visible := visibleConnections(connections, 0, 3)

	c.Assert(visible, qt.DeepEquals, []live.Connection{{PID: 1}, {PID: 2}, {PID: 3}})
}

func TestViewportStartForSelectionCentersNearViewportEdges(t *testing.T) {
	c := qt.New(t)
	connections := []live.Connection{{PID: 1}, {PID: 2}, {PID: 3}, {PID: 4}, {PID: 5}}
	start := viewportStartForSelection(0, 2, len(connections), 3)

	c.Assert(visibleConnections(connections, start, 3), qt.DeepEquals, []live.Connection{{PID: 2}, {PID: 3}, {PID: 4}})
}

func TestFormatDuration(t *testing.T) {
	c := qt.New(t)

	c.Assert(formatDuration(0), qt.Equals, "-")
	c.Assert(formatDuration(250*time.Millisecond), qt.Equals, "00:00")
	c.Assert(formatDuration(1500*time.Millisecond), qt.Equals, "00:01")
	c.Assert(formatDuration(90*time.Second), qt.Equals, "01:30")
}

func TestEmptyDash(t *testing.T) {
	c := qt.New(t)

	c.Assert(emptyDash(" \t "), qt.Equals, "-")
	c.Assert(emptyDash("app"), qt.Equals, "app")
}

func TestClipLine(t *testing.T) {
	c := qt.New(t)

	c.Assert(clipLine("abcdef", 0), qt.Equals, "abcdef")
	c.Assert(clipLine("abcdef", 1), qt.Equals, "a")
	c.Assert(clipLine("abcdef", 3), qt.Equals, "ab…")
	c.Assert(clipLine("abcdef", 5), qt.Equals, "abcd…")

	clipped := clipLine("the quick brown fox jumps", 12)
	c.Assert(strings.Contains(clipped, "…"), qt.IsTrue)
	c.Assert(strings.Contains(clipped, "..."), qt.IsFalse)
}

func TestRenderDetailTabsHaveEqualWidths(t *testing.T) {
	c := qt.New(t)

	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(prev)

	query := renderDetailTabs(tabQuery, DefaultConnectionCapabilities())
	blockers := renderDetailTabs(tabBlockers, DefaultConnectionCapabilities())
	c.Assert(ansi.StringWidth(blockers), qt.Equals, ansi.StringWidth(query))
	c.Assert(ansi.StringWidth(clipLine(blockers, 200)), qt.Equals, ansi.StringWidth(blockers))
	c.Assert(ansi.StringWidth(clipLine(query, 200)), qt.Equals, ansi.StringWidth(query))
}

func TestRenderFooterShowsConfirmPrompt(t *testing.T) {
	c := qt.New(t)
	state := tableState{
		HasList:  true,
		Width:    180,
		Height:   24,
		Interval: time.Second,
		Confirm:  "Force terminate PID 10 on primary? (y/n)",
	}
	got := renderFooter(state)
	c.Assert(got, qt.Contains, "Force terminate PID 10")
}

func TestRenderFooterKeepsLongErrorVisible(t *testing.T) {
	c := qt.New(t)
	state := tableState{
		List:      live.NewConnectionList(time.Now(), []live.Connection{{PID: 10}}, live.SortByTransactionStart),
		HasList:   true,
		Selected:  -1,
		Width:     80,
		Height:    24,
		Interval:  time.Second,
		LastError: "list connections: server is warming up, please retry in a moment",
	}

	got := renderFooter(state)
	lines := strings.Split(stripANSI(got), "\n")

	c.Assert(lines[0], qt.Equals, "error: list connections: server is warming up, please retry in a moment")
	c.Assert(lipgloss.Width(lines[0]) <= 80, qt.IsTrue)
	c.Assert(got, qt.Contains, "up/down")
}

func TestRenderFooterShowsResumeWhenPaused(t *testing.T) {
	c := qt.New(t)
	state := tableState{
		List:    live.NewConnectionList(time.Now(), []live.Connection{{PID: 10}}, live.SortByTransactionStart),
		HasList: true,
		Paused:  true,
		Width:   120,
		Height:  24,
	}

	got := stripANSI(renderFooter(state))

	c.Assert(got, qt.Contains, "space resume")
	c.Assert(got, qt.Not(qt.Contains), "space pause")
	c.Assert(got, qt.Not(qt.Contains), "r refresh")
}

func TestRenderFooterClipsLongConfirmPrompt(t *testing.T) {
	c := qt.New(t)
	state := tableState{
		HasList:  true,
		Width:    72,
		Height:   24,
		Interval: time.Second,
		Confirm:  "Force terminate PID 39123 on hzi-4gomesickgvbywmt-cell1-1486496651-c9099829? (y/n)",
	}

	got := renderFooter(state)
	lines := strings.Split(stripANSI(got), "\n")

	c.Assert(lines[0], qt.Contains, "Force terminate PID 39123")
	c.Assert(lines[0], qt.Contains, "…")
	c.Assert(lipgloss.Width(lines[0]) <= 72, qt.IsTrue)
}

func selectedStatusLine(c *qt.C, rendered string) string {
	for _, line := range strings.Split(rendered, "\n") {
		if strings.HasPrefix(line, "selected pid ") {
			return line
		}
	}
	c.Fatalf("rendered output does not contain selected status:\n%s", rendered)
	return ""
}

// The refresh indicator is a fixed-width dot whose style (not width) changes
// with loading, so the header never reflows on refresh.
func TestRefreshIndicatorFixedWidth(t *testing.T) {
	c := qt.New(t)
	defer lipgloss.SetColorProfile(termenv.Ascii)
	lipgloss.SetColorProfile(termenv.ANSI256)

	pending := refreshIndicator(refreshDotPending)
	idle := refreshIndicator(refreshDotIdle)
	failing := refreshIndicator(refreshDotFailing)
	hidden := refreshIndicator(refreshDotHidden)

	// Every state renders one cell wide (constant width → no header reflow),
	// including replay's blank dot.
	for _, s := range []string{pending, idle, failing, hidden} {
		c.Assert(ansi.StringWidth(s), qt.Equals, 1)
	}
	c.Assert(pending, qt.Contains, "●")
	c.Assert(idle, qt.Contains, "●")
	c.Assert(failing, qt.Contains, "●")
	// ...but the styling differs so each live state is distinguishable.
	c.Assert(pending, qt.Not(qt.Equals), idle)
	c.Assert(failing, qt.Not(qt.Equals), pending)
	c.Assert(failing, qt.Not(qt.Equals), idle)
}

func TestRenderHelpVitessKeepsKillNames(t *testing.T) {
	c := qt.New(t)

	help := stripANSI(renderHelpModal(helpState{
		Target: Target{Database: "prod", Branch: "main"},
		Width:  120,
		Height: 40,
		Capabilities: ConnectionCapabilities{
			CancelQuery:         ActionTargetQueryID,
			TerminateConnection: ActionTargetConnectionID,
			configured:          true,
		},
	}))

	c.Assert(help, qt.Contains, "c  Kill the selected query (KILL QUERY)")
	c.Assert(help, qt.Contains, "K  Kill the selected connection (KILL)")
}

func TestRenderHelpModalReplayHidesRefreshAndCapture(t *testing.T) {
	c := qt.New(t)
	modal := stripANSI(renderHelpModal(helpState{
		Target: Target{Database: "prod", Branch: "main"},
		Width:  120,
		Height: 40,
		Replay: true,
	}))

	c.Assert(modal, qt.Not(qt.Contains), "r refresh")
	c.Assert(modal, qt.Not(qt.Contains), "C capture")
	c.Assert(modal, qt.Contains, "space pause")
}
