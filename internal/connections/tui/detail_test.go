package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	qt "github.com/frankban/quicktest"
	live "github.com/planetscale/cli/internal/connections"
)

func newDetailModel(t *testing.T, stub *clientStub, list live.ConnectionList) Model {
	t.Helper()
	model := NewModel(context.Background(), stub, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 30})
	model = updated.(Model)
	updated, _ = model.Update(listMsg{list: list})
	return updated.(Model)
}

func TestEnterFromTableOpensDetailOnQueryTab(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:       10,
		Instance:  "primary",
		QueryText: "SELECT * FROM widgets WHERE id = 7",
	}}, live.SortByTransactionStart)
	model := newDetailModel(t, &clientStub{}, list)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view := updated.(Model).View()

	c.Assert(view, qt.Contains, "pid 10")
	c.Assert(view, qt.Contains, "on primary")
	c.Assert(view, qt.Contains, "SELECT * FROM widgets WHERE id = 7")
	c.Assert(view, qt.Contains, "blockers")
	c.Assert(view, qt.Contains, "[query]")
	c.Assert(view, qt.Not(qt.Contains), "V Query")
	c.Assert(view, qt.Not(qt.Contains), "B Blockers")
}

func TestTableUppercaseVOpensDetailOnQueryTab(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:       10,
		Instance:  "primary",
		QueryText: "SELECT * FROM widgets WHERE id = 7",
	}}, live.SortByTransactionStart)
	model := newDetailModel(t, &clientStub{}, list)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	got := updated.(Model)

	c.Assert(got.detailOpen, qt.IsTrue)
	c.Assert(got.detailTab, qt.Equals, tabQuery)
	c.Assert(got.View(), qt.Contains, "SELECT * FROM widgets WHERE id = 7")
}

func TestTableLowercaseVOpensDetailOnQueryTab(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:       10,
		Instance:  "primary",
		QueryText: "SELECT * FROM widgets WHERE id = 7",
	}}, live.SortByTransactionStart)
	model := newDetailModel(t, &clientStub{}, list)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	got := updated.(Model)

	c.Assert(got.detailOpen, qt.IsTrue)
	c.Assert(got.detailTab, qt.Equals, tabQuery)
	c.Assert(got.View(), qt.Contains, "SELECT * FROM widgets WHERE id = 7")
}

func TestTableUppercaseBOpensDetailOnBlockersTab(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 10, Instance: "primary", BlockedBy: []int{20}, QueryText: "blocked"},
		{PID: 20, Instance: "primary", QueryText: "blocker holding lock"},
	}, live.SortByTransactionStart)
	model := newDetailModel(t, &clientStub{}, list)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("B")})
	got := updated.(Model)

	c.Assert(got.detailOpen, qt.IsTrue)
	c.Assert(got.detailTab, qt.Equals, tabBlockers)
	c.Assert(got.View(), qt.Contains, "BLOCKED BY")
}

func TestDetailTabKeysUseArrowsAndBlockersShortcut(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:       10,
		Instance:  "primary",
		QueryText: "SELECT 1",
	}}, live.SortByTransactionStart)
	model := newDetailModel(t, &clientStub{}, list)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	got := updated.(Model)
	c.Assert(got.detailTab, qt.Equals, tabBlockers)

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRight})
	got = updated.(Model)
	c.Assert(got.detailTab, qt.Equals, tabQuery)
}

func TestDetailVDoesNotSwitchTabs(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:       10,
		Instance:  "primary",
		QueryText: "SELECT 1",
	}}, live.SortByTransactionStart)
	model := newDetailModel(t, &clientStub{}, list)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	got := updated.(Model)
	c.Assert(got.detailTab, qt.Equals, tabBlockers)

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})

	c.Assert(updated.(Model).detailTab, qt.Equals, tabBlockers)
}

func TestEscapeReturnsToTable(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10}}, live.SortByTransactionStart)
	model := newDetailModel(t, &clientStub{}, list)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEsc})

	view := updated.(Model).View()
	c.Assert(view, qt.Contains, "connections 1")
}

func TestDetailSwitchesBetweenQueryAndBlockersTabs(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 10, Instance: "primary", BlockedBy: []int{20}, QueryText: "blocked"},
		{PID: 20, Instance: "primary", QueryText: "blocker holding lock"},
	}, live.SortByTransactionStart)
	model := newDetailModel(t, &clientStub{}, list)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	view := updated.(Model).View()

	c.Assert(view, qt.Contains, "BLOCKED BY")
	c.Assert(view, qt.Contains, "20")
	c.Assert(view, qt.Contains, "blocker holding lock")

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRight})
	view = updated.(Model).View()
	c.Assert(view, qt.Contains, "blocked")
	c.Assert(view, qt.Not(qt.Contains), "BLOCKED BY")
}

func TestDetailArrowTabSwitchResetsQueryOffset(t *testing.T) {
	c := qt.New(t)
	query := "select a, b, c\nfrom t\nwhere a = 1\nand b = 2\nand c = 3\nand d = 4\nand e = 5\nand f = 6\norder by a\nlimit 10"
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10, Instance: "primary", QueryText: query}}, live.SortByTransactionStart)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	updated, _ = updated.(Model).Update(listMsg{list: list})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	for i := 0; i < 4; i++ {
		updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	got := updated.(Model)
	c.Assert(got.queryOffset > 0, qt.IsTrue)
	got.detailTab = tabBlockers
	updated = got

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyLeft})

	c.Assert(updated.(Model).detailTab, qt.Equals, tabQuery)
	c.Assert(updated.(Model).queryOffset, qt.Equals, 0)
}

func TestDetailExplicitTabKeysResetQueryOffset(t *testing.T) {
	c := qt.New(t)
	query := "select a, b, c\nfrom t\nwhere a = 1\nand b = 2\nand c = 3\nand d = 4\nand e = 5\nand f = 6\norder by a\nlimit 10"
	list := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 10, Instance: "primary", BlockedBy: []int{20}, QueryText: query},
		{PID: 20, Instance: "primary", QueryText: "blocker holding lock"},
	}, live.SortByTransactionStart)

	for _, key := range []string{"b", "B"} {
		model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
		updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
		updated, _ = updated.(Model).Update(listMsg{list: list})
		updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
		for i := 0; i < 4; i++ {
			updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
		}
		got := updated.(Model)
		c.Assert(got.queryOffset > 0, qt.IsTrue)

		updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})

		got = updated.(Model)
		c.Assert(got.detailTab, qt.Equals, tabBlockers)
		c.Assert(got.queryOffset, qt.Equals, 0)
	}

	for _, key := range []tea.KeyType{tea.KeyLeft, tea.KeyRight} {
		model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
		updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
		updated, _ = updated.(Model).Update(listMsg{list: list})
		updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
		for i := 0; i < 4; i++ {
			updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
		}
		got := updated.(Model)
		c.Assert(got.queryOffset > 0, qt.IsTrue)
		got.detailTab = tabBlockers

		updated, _ = got.Update(tea.KeyMsg{Type: key})

		got = updated.(Model)
		c.Assert(got.detailTab, qt.Equals, tabQuery)
		c.Assert(got.queryOffset, qt.Equals, 0)
	}
}

func TestDetailBlockerTabClampsOutOfRangeSelection(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 10, Instance: "primary", BlockedBy: []int{20, 30}, QueryText: "blocked"},
		{PID: 20, Instance: "primary", QueryText: "first blocker"},
		{PID: 30, Instance: "primary", QueryText: "second blocker"},
	}, live.SortByTransactionStart)

	for _, key := range []string{"b", "B"} {
		model := newDetailModel(t, &clientStub{}, list)
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		got := updated.(Model)
		got.blockerRow = 99

		updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})

		got = updated.(Model)
		c.Assert(got.detailTab, qt.Equals, tabBlockers)
		c.Assert(got.blockerRow, qt.Equals, 1)
	}
}

func TestDetailDownArrowSelectsNextBlockerOnBlockersTab(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 10, BlockedBy: []int{20}},
		{PID: 20, BlockedBy: []int{30}},
		{PID: 30},
	}, live.SortByTransactionStart)
	model := newDetailModel(t, &clientStub{}, list)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	got := updated.(Model)
	c.Assert(got.detailTab, qt.Equals, tabBlockers)
	c.Assert(got.blockerRow, qt.Equals, 0)

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	c.Assert(updated.(Model).blockerRow, qt.Equals, 1)
}

func TestDetailCancelDispatchesActionForSubject(t *testing.T) {
	c := qt.New(t)
	xid := "10-1"
	qid := "10-2"
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:           10,
		Instance:      "primary",
		TransactionID: &xid,
		QueryID:       &qid,
	}}, live.SortByTransactionStart)
	stub := &clientStub{}
	model := newDetailModel(t, stub, list)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, cmd := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})

	got := updated.(Model)
	c.Assert(cmd, qt.IsNil)
	c.Assert(got.confirming, qt.IsTrue)
	c.Assert(got.pendingKind, qt.Equals, actionCancelQuery)
	c.Assert(stub.cancelCalls, qt.Equals, 0)

	updated, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	c.Assert(cmd, qt.IsNotNil)
	cmd()
	c.Assert(stub.cancelCalls, qt.Equals, 1)
	c.Assert(stub.lastTarget.PID, qt.Equals, 10)
	c.Assert(stub.lastTarget.Instance, qt.Equals, "primary")
	c.Assert(stub.lastTarget.QueryID, qt.Not(qt.IsNil))
	c.Assert(*stub.lastTarget.QueryID, qt.Equals, "10-2")
	_ = updated
}

func TestDetailCancelOnBlockersTabTargetsSelectedBlocker(t *testing.T) {
	c := qt.New(t)
	bxid := "20-99"
	bqid := "20-q"
	list := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 10, Instance: "primary", BlockedBy: []int{20}},
		{PID: 20, Instance: "primary", TransactionID: &bxid, QueryID: &bqid},
	}, live.SortByTransactionStart)
	stub := &clientStub{}
	model := newDetailModel(t, stub, list)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	updated, cmd := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})

	got := updated.(Model)
	c.Assert(cmd, qt.IsNil)
	c.Assert(got.confirming, qt.IsTrue)
	c.Assert(got.pendingKind, qt.Equals, actionCancelQuery)
	c.Assert(stub.cancelCalls, qt.Equals, 0)

	updated, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	c.Assert(cmd, qt.IsNotNil)
	cmd()
	c.Assert(stub.cancelCalls, qt.Equals, 1)
	c.Assert(stub.lastTarget.PID, qt.Equals, 20)
	c.Assert(*stub.lastTarget.TransactionID, qt.Equals, "20-99")
	_ = updated
}

func TestDetailKillTransactionPromptsConfirm(t *testing.T) {
	c := qt.New(t)
	xid := "10-x"
	list := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 10, Instance: "primary", TransactionID: &xid},
	}, live.SortByTransactionStart)
	stub := &clientStub{}
	model := newDetailModel(t, stub, list)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})

	got := updated.(Model)
	c.Assert(got.confirming, qt.IsTrue)
	c.Assert(got.pendingKind, qt.Equals, actionTerminateTxn)
	view := got.View()
	c.Assert(view, qt.Contains, "Terminate transaction on PID 10")
}

func TestDetailEnterOnBlockersTabReanchorsToBlocker(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 10, Instance: "primary", BlockedBy: []int{20}},
		{PID: 20, Instance: "primary"},
	}, live.SortByTransactionStart)
	model := newDetailModel(t, &clientStub{}, list)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})

	got := updated.(Model)
	c.Assert(got.detailPID, qt.Equals, 20)
}

func TestDetailRendersConnectionEndedWhenSubjectMissing(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10, Instance: "primary"}}, live.SortByTransactionStart)
	model := newDetailModel(t, &clientStub{}, list)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	updated, _ = updated.(Model).Update(listMsg{list: live.ConnectionList{}})

	view := stripANSI(updated.(Model).View())
	c.Assert(view, qt.Contains, "connection ended")
	c.Assert(view, qt.Contains, "Press q or esc to return")

	endedFooter := stripANSI(renderDetailHelp(detailState{SubjectFound: false, Width: 120, Height: 24}))
	c.Assert(strings.Contains(endedFooter, "cancel"), qt.IsFalse)
	c.Assert(strings.Contains(endedFooter, "kill"), qt.IsFalse)
	c.Assert(strings.Contains(endedFooter, "terminate"), qt.IsFalse)
	c.Assert(strings.Contains(endedFooter, "q/esc back"), qt.IsTrue)

	liveFooter := stripANSI(renderDetailHelp(detailState{SubjectFound: true, Width: 200}))
	c.Assert(strings.Contains(liveFooter, "cancel query"), qt.IsTrue)
	c.Assert(strings.Contains(liveFooter, "kill transaction"), qt.IsTrue)
}

func TestRenderDetailFieldsShowNoneForEmptyValues(t *testing.T) {
	c := qt.New(t)
	view := stripANSI(renderDetail(detailState{
		List: live.NewConnectionList(time.Now(), nil, live.SortByDuration),
		Subject: live.Connection{
			PID:       101,
			Duration:  42 * time.Second,
			QueryText: "SELECT 1",
		},
		SubjectFound:  true,
		Tab:           tabQuery,
		DisplayPreset: connectionDisplayProcesslist,
		Capabilities:  DefaultConnectionCapabilities(),
		Width:         120,
		Height:        24,
	}))

	for _, field := range []string{
		"tablet:",
		"state:",
		"user:",
		"database:",
		"client_addr:",
		"connection_id:",
		"query_id:",
	} {
		c.Assert(view, qt.Contains, field)
		c.Assert(view, qt.Contains, field+" ")
	}
	c.Assert(strings.Count(view, "none"), qt.Equals, 7)
}

func TestRenderDetailWrapsLongIDFields(t *testing.T) {
	c := qt.New(t)
	queryID := "primary-1234567890abcdef-primary-1234567890abcdef-query"
	view := stripANSI(renderDetail(detailState{
		List: live.NewConnectionList(time.Now(), nil, live.SortByTransactionStart),
		Subject: live.Connection{
			PID:     101,
			QueryID: &queryID,
		},
		SubjectFound: true,
		Tab:          tabQuery,
		Capabilities: DefaultConnectionCapabilities(),
		Width:        44,
		Height:       24,
	}))

	c.Assert(view, qt.Contains, "query_id:")
	c.Assert(view, qt.Contains, "primary-1234567890abcdef-pr")
	c.Assert(view, qt.Contains, "imary-1234567890abcdef-quer")
	c.Assert(view, qt.Contains, "                 y")
	c.Assert(view, qt.Not(qt.Contains), "query_id:      primary-1234567890abcdef-prima…")
}

func TestDetailEndedStateKeepsTargetAndActionFeedback(t *testing.T) {
	c := qt.New(t)
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10, Instance: "primary"}}, live.SortByTransactionStart)
	model := newDetailModel(t, &clientStub{}, list).WithTarget(Target{
		Database: "prod",
		Branch:   "main",
	})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, _ = updated.(Model).Update(listMsg{list: live.ConnectionList{}})

	view := stripANSI(updated.(Model).View())
	c.Assert(view, qt.Contains, "prod / main | connection ended")

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	got := stripANSI(updated.(Model).View())
	c.Assert(got, qt.Contains, "connection ended — actions unavailable; esc to go back")

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	got = stripANSI(updated.(Model).View())
	c.Assert(got, qt.Contains, "connection ended — actions unavailable; esc to go back")

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	got = stripANSI(updated.(Model).View())
	c.Assert(got, qt.Contains, "connection ended — actions unavailable; esc to go back")
}

func TestDetailFooterWrapsAtNarrowWidth(t *testing.T) {
	c := qt.New(t)

	footer := stripANSI(renderDetailHelp(detailState{
		SubjectFound: true,
		Tab:          tabBlockers,
		Width:        80,
	}))

	c.Assert(strings.Contains(footer, "ctrl+c quit"), qt.IsTrue)
	c.Assert(strings.Contains(footer, "? help"), qt.IsTrue)
	c.Assert(strings.Contains(footer, "q/esc back"), qt.IsTrue)

	for _, line := range strings.Split(footer, "\n") {
		c.Assert(ansi.StringWidth(line) <= 80, qt.IsTrue)
	}
	c.Assert(strings.Count(footer, "\n") >= 1, qt.IsTrue)
}

func TestDetailFooterShowsShiftKForForceTerminate(t *testing.T) {
	c := qt.New(t)

	footer := stripANSI(renderDetailHelp(detailState{
		SubjectFound: true,
		Width:        160,
	}))

	c.Assert(footer, qt.Contains, "shift+K force terminate")
}

// The Blockers-tab contextual footer includes both navigation and destructive
// action hints, so it must stay within the default 200-column recording width.
func TestDetailBlockersFooterFitsOneLineAtDefaultRecordingWidth(t *testing.T) {
	c := qt.New(t)

	footer := stripANSI(renderDetailHelp(detailState{
		SubjectFound: true,
		Tab:          tabBlockers,
		Width:        200,
	}))

	c.Assert(strings.Contains(footer, "\n"), qt.IsFalse)
	c.Assert(ansi.StringWidth(footer) <= tableWidth(200), qt.IsTrue)
	c.Assert(strings.Contains(footer, "ctrl+c quit"), qt.IsTrue)
}

func TestRenderBlockersShowsBlockedByAndBlockingHeadings(t *testing.T) {
	c := qt.New(t)
	subject := live.Connection{PID: 10, BlockedBy: []int{20}}
	list := live.ConnectionList{Connections: []live.Connection{
		subject,
		{PID: 20, BlockedBy: []int{30}},
		{PID: 30},
		{PID: 40, BlockedBy: []int{10}},
	}}

	state := detailState{
		List:         list,
		Subject:      subject,
		SubjectFound: true,
		Tab:          tabBlockers,
		Width:        120,
		Height:       30,
	}
	out := renderDetail(state)
	c.Assert(out, qt.Contains, "BLOCKED BY")
	c.Assert(out, qt.Contains, "BLOCKING")
	c.Assert(out, qt.Contains, "20")
	c.Assert(out, qt.Contains, "40")
}

func TestRenderBlockersUsesFriendlyEmptyStates(t *testing.T) {
	c := qt.New(t)
	conn := live.Connection{PID: 10}
	lines := renderBlockers(detailState{
		List:         live.ConnectionList{Connections: []live.Connection{conn}},
		Subject:      conn,
		SubjectFound: true,
	}, 120, 20)
	got := stripANSI(strings.Join(lines, "\n"))

	c.Assert(got, qt.Contains, "No upstream blocker")
	c.Assert(got, qt.Contains, "Not blocking other connections")
}

func TestRenderDetailCoercesUnavailableBlockersToQuery(t *testing.T) {
	c := qt.New(t)
	subject := live.Connection{PID: 10, BlockedBy: []int{20}, QueryText: "SELECT 1"}
	view := stripANSI(renderDetail(detailState{
		List: live.ConnectionList{Connections: []live.Connection{
			subject,
			{PID: 20, QueryText: "blocker"},
		}},
		Subject:      subject,
		SubjectFound: true,
		Tab:          tabBlockers,
		Capabilities: ConnectionCapabilities{
			CancelQuery:         ActionTargetQueryID,
			TerminateConnection: ActionTargetConnectionID,
			ShowBlockers:        false,
		},
		Width:  120,
		Height: 24,
	}))

	c.Assert(view, qt.Contains, "[query]")
	c.Assert(view, qt.Contains, "SELECT 1")
	c.Assert(view, qt.Not(qt.Contains), "BLOCKED BY")
	c.Assert(view, qt.Not(qt.Contains), "BLOCKING")
	c.Assert(view, qt.Not(qt.Contains), "selected blocker")
}

func TestDetailQueryTabRendersProcesslistRecord(t *testing.T) {
	c := qt.New(t)
	connectionID := "zone1-2001-101"
	queryID := "zone1-2001-101"
	view := renderDetail(detailState{
		List: live.NewConnectionList(time.Now(), nil, live.SortByDuration),
		Subject: live.Connection{
			PID:          101,
			Instance:     "zone1-2001",
			State:        "Query/executing",
			Duration:     42 * time.Second,
			Username:     "vt_app",
			DatabaseName: "checkout",
			ClientAddr:   "10.0.0.1:1234",
			ConnectionID: &connectionID,
			QueryID:      &queryID,
			QueryText:    "SELECT 1",
		},
		SubjectFound:  true,
		Tab:           tabQuery,
		DisplayPreset: connectionDisplayProcesslist,
		Capabilities:  DefaultConnectionCapabilities(),
		Width:         120,
		Height:        24,
	})

	stripped := stripANSI(view)
	c.Assert(stripped, qt.Contains, "pid:")
	c.Assert(stripped, qt.Contains, "tablet:")
	c.Assert(stripped, qt.Contains, "zone1-2001")
	c.Assert(stripped, qt.Contains, "state:")
	c.Assert(stripped, qt.Contains, "Query/executing")
	c.Assert(stripped, qt.Contains, "duration:")
	c.Assert(stripped, qt.Contains, "user:")
	c.Assert(stripped, qt.Contains, "database:")
	c.Assert(stripped, qt.Contains, "checkout")
	c.Assert(stripped, qt.Contains, "client_addr:")
	c.Assert(stripped, qt.Contains, "connection_id:")
	c.Assert(stripped, qt.Contains, "query_id:")
	c.Assert(stripped, qt.Contains, "SELECT 1")
	c.Assert(stripped, qt.Not(qt.Contains), "blocked_by:")
	c.Assert(stripped, qt.Not(qt.Contains), "wait:")
	c.Assert(stripped, qt.Not(qt.Contains), "transaction_id:")
}

func TestDetailProcesslistZeroDurationShowsNone(t *testing.T) {
	c := qt.New(t)
	view := stripANSI(renderDetail(detailState{
		List: live.NewConnectionList(time.Now(), nil, live.SortByDuration),
		Subject: live.Connection{
			PID:      101,
			Instance: "zone1-2001",
			State:    "Sleep",
		},
		SubjectFound:  true,
		Tab:           tabQuery,
		DisplayPreset: connectionDisplayProcesslist,
		Capabilities:  DefaultConnectionCapabilities(),
		Width:         120,
		Height:        24,
	}))

	c.Assert(view, qt.Contains, "duration:        none")
	c.Assert(view, qt.Not(qt.Contains), "duration:        0s")
}

func TestRenderQueryShowsEmptyMessageWhenTextBlank(t *testing.T) {
	c := qt.New(t)
	subject := live.Connection{PID: 10}
	state := detailState{
		List:         live.ConnectionList{Connections: []live.Connection{subject}},
		Subject:      subject,
		SubjectFound: true,
		Tab:          tabQuery,
		Width:        80,
		Height:       24,
	}
	view := stripANSI(renderDetail(state))
	c.Assert(view, qt.Contains, "no query")
	c.Assert(view, qt.Not(qt.Contains), "query empty")
}

func TestRenderQueryShowsOverflowIndicatorWhenViewportIsOneLine(t *testing.T) {
	c := qt.New(t)
	conn := live.Connection{
		QueryText: "select a from t where a = 1 and b = 2 and c = 3 order by a limit 10",
	}

	lines := renderQuery(conn, connectionDisplayDefault, 80, 1, 0)

	c.Assert(lines, qt.HasLen, 1)
	c.Assert(lines[0], qt.Contains, "lines ")
	c.Assert(lines[0], qt.Not(qt.Contains), "SELECT")
}

func TestDetailHelpHidesRefreshWhilePaused(t *testing.T) {
	c := qt.New(t)
	footer := stripANSI(renderDetailHelp(detailState{SubjectFound: true, Width: 200, Paused: true}))

	c.Assert(footer, qt.Not(qt.Contains), "r refresh")
	c.Assert(footer, qt.Contains, "space resume")
}
