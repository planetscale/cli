package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	qt "github.com/frankban/quicktest"
	"github.com/muesli/termenv"
	live "github.com/planetscale/cli/internal/connections"
	"github.com/planetscale/cli/internal/connections/history"
)

func TestModelRendersListAndRows(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)
	list := live.NewConnectionList(time.Unix(100, 0), []live.Connection{{
		PID:             10,
		Instance:        "primary",
		Username:        "brett",
		ApplicationName: "psql",
		ClientAddr:      "127.0.0.1",
		State:           "active",
		Duration:        3 * time.Second,
		QueryText:       "SELECT * FROM widgets",
	}}, live.SortByTransactionStart)

	updated, _ = model.Update(listMsg{list: list})
	view := updated.(Model).View()

	c.Assert(view, qt.Contains, "connections 1")
	c.Assert(view, qt.Contains, "10")
	c.Assert(view, qt.Contains, "SELECT * FROM widgets")
	c.Assert(view, qt.Contains, "q quit")
}

func TestModelRendersTargetInTableHeader(t *testing.T) {
	tests := []struct {
		name  string
		model Model
		list  live.ConnectionList
		want  string
	}{
		{
			name: "postgres target",
			model: NewModel(context.Background(), &clientStub{}, time.Second, 0).
				WithTarget(Target{Database: "prod", Branch: "main"}),
			list: live.NewConnectionList(time.Unix(100, 0), []live.Connection{{PID: 10}}, live.SortByTransactionStart),
			want: "prod / main",
		},
		{
			name: "vitess target from configured target",
			model: NewModel(context.Background(), &clientStub{}, time.Second, 0).
				WithTarget(Target{Database: "shop", Branch: "main", Keyspace: "commerce", Shard: "-80"}).
				WithConnectionView(VitessConnectionView),
			list: live.NewConnectionList(time.Unix(100, 0), []live.Connection{{PID: 10}}, live.SortByDuration),
			want: "shop / main / commerce / -80",
		},
		{
			name: "vitess target from topology",
			model: NewModel(context.Background(), &clientStub{}, time.Second, 0).
				WithTarget(Target{Database: "shop", Branch: "main"}).
				WithConnectionView(VitessConnectionView),
			list: func() live.ConnectionList {
				list := live.NewConnectionList(time.Unix(100, 0), []live.Connection{{PID: 10}}, live.SortByDuration)
				list.Topology = &live.Topology{Keyspace: "commerce", Shard: "-80", Tablet: "zone1-1001"}
				return list
			}(),
			want: "shop / main / commerce / -80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			updated, _ := tt.model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
			updated, _ = updated.(Model).Update(listMsg{list: tt.list})

			view := updated.(Model).View()

			c.Assert(view, qt.Contains, tt.want)
			c.Assert(view, qt.Contains, "connections 1")
		})
	}
}

func TestModelRendersActiveFilterInTableHeader(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithFilter("filter: instance=db-1")

	view := model.View()

	c.Assert(view, qt.Contains, "filter: instance=db-1")
}

func TestModelClearsScreenOnResize(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)

	_, cmd := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	c.Assert(cmd, qt.Not(qt.IsNil))
	c.Assert(fmt.Sprintf("%T", cmd()), qt.Equals, "tea.clearScreenMsg")
}

func TestModelRendersTargetInDetailHeader(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithTarget(Target{Database: "prod", Branch: "main"})
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10, Instance: "primary", QueryText: "select 1"}}, live.SortByTransactionStart)
	updated, _ := model.Update(listMsg{list: list})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})

	view := updated.(Model).View()

	c.Assert(view, qt.Contains, "prod / main")
	c.Assert(view, qt.Contains, "pid 10")
}

func TestModelKeepsLastListWhenRefreshFails(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10}}, live.SortByTransactionStart)

	updated, _ := model.Update(listMsg{list: list})
	updated, _ = updated.(Model).Update(listMsg{err: errors.New("connection refused")})
	view := updated.(Model).View()

	c.Assert(view, qt.Contains, "10")
	c.Assert(view, qt.Contains, "error: connection refused")
}

func degradedList(captured time.Time, pid int) live.ConnectionList {
	list := live.NewConnectionList(captured, []live.Connection{{PID: pid, Instance: "primary"}}, live.SortByTransactionStart)
	list.Instances = []live.InstanceMeta{
		{ID: "primary", Role: "primary", Error: "remote service unavailable"},
		{ID: "replica-1", Role: "replica"},
	}
	return list
}

func TestModelHoldsLastGoodListOnPersistentPartial(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	good := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10, Instance: "primary"}}, live.SortByTransactionStart)

	updated, _ := model.Update(listMsg{list: good})
	// A degraded refresh (client retries already exhausted) must not replace the
	// good frame or show the unreachable banner — it holds and goes stale.
	updated, _ = updated.(Model).Update(listMsg{list: degradedList(time.Now(), 20)})
	got := updated.(Model)

	// Held the last good frame (PID 10); the degraded frame (PID 20) was not
	// swapped in. Assert on the held data, not the rendered string — the header
	// renders a wall-clock timestamp that can incidentally contain a PID.
	held := got.currentList().Connections
	c.Assert(len(held), qt.Equals, 1)
	c.Assert(held[0].PID, qt.Equals, 10)
	c.Assert(got.View(), qt.Not(qt.Contains), "unreachable")
	c.Assert(got.consecutiveErrors, qt.Equals, 1)
}

func TestModelShowsPartialOnFirstLoadWithNoPriorFrame(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)

	// No good frame to hold yet, so the first result is shown even if degraded.
	updated, _ := model.Update(listMsg{list: degradedList(time.Now(), 30)})
	got := updated.(Model)

	// No prior frame to hold, so the degraded result is shown (banner and all).
	c.Assert(got.currentList().Connections[0].PID, qt.Equals, 30)
	c.Assert(got.View(), qt.Contains, "unreachable")
}

func TestModelShowsInitialListErrorBeforeAnySuccessfulList(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)

	updated, _ := model.Update(listMsg{err: errors.New("list connections: server is warming up, please retry in a moment")})
	view := updated.(Model).View()

	c.Assert(view, qt.Not(qt.Contains), "loading live connections...")
	c.Assert(view, qt.Contains, "unable to load live connections")
	c.Assert(view, qt.Contains, "list connections: server is warming up, please retry in a moment")
}

func TestModelInitialPermissionDeniedShowsAccessDeniedState(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	err := fmt.Errorf("wrapped read failure: %w", &live.HTTPError{
		Op:         "list connections",
		StatusCode: http.StatusForbidden,
		Message:    "policy denied this request",
	})

	updated, _ := model.Update(listMsg{err: err})
	view := stripANSI(updated.(Model).View())

	c.Assert(view, qt.Contains, "connections —")
	c.Assert(view, qt.Contains, "you don't have permission to view live connections")
	c.Assert(view, qt.Contains, "production branches require the Analyst role or higher — ask an org admin, or use a development branch")
	c.Assert(view, qt.Contains, "r refresh | space pause | ? help | q quit")
	c.Assert(view, qt.Not(qt.Contains), "policy denied this request")
	c.Assert(view, qt.Not(qt.Contains), "enter detail")
	c.Assert(view, qt.Not(qt.Contains), "cancel query")
	c.Assert(view, qt.Not(qt.Contains), "kill transaction")
	c.Assert(view, qt.Not(qt.Contains), "force terminate")
}

func TestModelActionForbiddenAfterListKeepsTableWithErrorFooter(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:       10,
		QueryText: "SELECT pg_sleep(30)",
	}}, live.SortByTransactionStart)
	updated, _ := model.Update(listMsg{list: list})

	updated, _ = updated.(Model).Update(actionResultMsg{
		kind: actionTerminateConn,
		err: &live.HTTPError{
			Op:         "terminate connection",
			StatusCode: http.StatusForbidden,
			Message:    "denied by infra policy on tablet zone1-1001",
		},
	})
	view := stripANSI(updated.(Model).View())

	c.Assert(view, qt.Contains, "connections 1")
	c.Assert(view, qt.Contains, "10")
	c.Assert(view, qt.Contains, "SELECT pg_sleep(30)")
	c.Assert(view, qt.Contains, "error: permission denied: you don't have permission to modify live connections")
	c.Assert(view, qt.Not(qt.Contains), "zone1-1001")
	c.Assert(view, qt.Not(qt.Contains), "you don't have permission to view live connections")
}

func TestModelListForbiddenAfterSuccessKeepsTableWithErrorFooter(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:       10,
		QueryText: "SELECT pg_sleep(30)",
	}}, live.SortByTransactionStart)
	updated, _ := model.Update(listMsg{list: list})

	updated, _ = updated.(Model).Update(listMsg{
		err: &live.HTTPError{
			Op:         "list connections",
			StatusCode: http.StatusForbidden,
			Message:    "denied by infra policy on tablet zone1-1001",
		},
	})
	view := stripANSI(updated.(Model).View())

	c.Assert(view, qt.Contains, "connections 1")
	c.Assert(view, qt.Contains, "10")
	c.Assert(view, qt.Contains, "SELECT pg_sleep(30)")
	c.Assert(view, qt.Contains, "error: permission denied: you don't have permission to view live connections")
	c.Assert(view, qt.Not(qt.Contains), "zone1-1001")
	c.Assert(view, qt.Not(qt.Contains), "unable to load live connections")
}

func TestModelInitialErrorDoesNotAdvertiseRowActions(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)

	updated, _ := model.Update(listMsg{err: errors.New("list connections: server is warming up")})
	view := stripANSI(updated.(Model).View())

	c.Assert(view, qt.Contains, "connections —")
	c.Assert(view, qt.Contains, "unable to load live connections")
	c.Assert(view, qt.Not(qt.Contains), "enter detail")
	c.Assert(view, qt.Not(qt.Contains), "cancel query")
	c.Assert(view, qt.Not(qt.Contains), "kill transaction")
	c.Assert(view, qt.Not(qt.Contains), "force terminate")
}

func TestModelReadOnlyActionsHidesFooterActions(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithReadOnlyActions("replay mode cannot run live actions")
	qid := "10-q"
	xid := "10-x"
	cid := "10-c"
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:           10,
		QueryID:       &qid,
		TransactionID: &xid,
		ConnectionID:  &cid,
		QueryText:     "SELECT 1",
	}}, live.SortByTransactionStart)

	updated, _ := model.Update(listMsg{list: list})
	tableView := stripANSI(updated.(Model).View())

	c.Assert(tableView, qt.Contains, "r refresh")
	c.Assert(tableView, qt.Not(qt.Contains), "cancel query")
	c.Assert(tableView, qt.Not(qt.Contains), "kill transaction")
	c.Assert(tableView, qt.Not(qt.Contains), "force terminate")

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	detailView := stripANSI(updated.(Model).View())

	c.Assert(detailView, qt.Contains, "q/esc back")
	c.Assert(detailView, qt.Not(qt.Contains), "cancel query")
	c.Assert(detailView, qt.Not(qt.Contains), "kill transaction")
	c.Assert(detailView, qt.Not(qt.Contains), "force terminate")
}

func TestModelNoticeSurvivesRefreshUntilTTL(t *testing.T) {
	c := qt.New(t)
	base := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	model.now = func() time.Time { return base }

	updated, _ := model.Update(actionResultMsg{kind: actionTerminateConn})
	got := updated.(Model)
	c.Assert(got.View(), qt.Contains, "force terminate sent")
	noticeID := got.notice.id

	got.now = func() time.Time { return base.Add(2 * time.Second) }
	updated, _ = got.Update(listMsg{list: live.NewConnectionList(base, []live.Connection{{PID: 10}}, live.SortByTransactionStart)})
	got = updated.(Model)
	c.Assert(got.View(), qt.Contains, "force terminate sent")

	got.now = func() time.Time { return base.Add(6 * time.Second) }
	updated, _ = got.Update(noticeTimeoutMsg{id: noticeID})
	got = updated.(Model)
	c.Assert(got.View(), qt.Not(qt.Contains), "force terminate sent")
}

func TestModelNoticeExpiresWhileReplayIsIdle(t *testing.T) {
	c := qt.New(t)
	base := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	h := history.NewCaptureHistory(3)
	h.Push(live.NewConnectionList(base, []live.Connection{{PID: 10}}, live.SortByTransactionStart))
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).WithCaptureHistory(h)
	model.now = func() time.Time { return base }

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	got := updated.(Model)
	c.Assert(cmd, qt.Not(qt.IsNil))
	c.Assert(got.liveRefresh, qt.IsFalse)
	c.Assert(got.notice.text, qt.Equals, "resumed")
	noticeID := got.notice.id

	updated, cmd = got.Update(noticeTimeoutMsg{id: noticeID})
	got = updated.(Model)

	c.Assert(cmd, qt.IsNil)
	c.Assert(got.notice.text, qt.Equals, "")
}

func TestModelNoticeExpiryIgnoresStaleTimer(t *testing.T) {
	c := qt.New(t)
	first := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	second := first.Add(time.Second)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	model.now = func() time.Time { return first }

	updated, cmd := model.Update(actionResultMsg{kind: actionCancelQuery})
	got := updated.(Model)
	c.Assert(cmd, qt.Not(qt.IsNil))
	c.Assert(got.notice.text, qt.Equals, "cancel query sent")
	firstNoticeID := got.notice.id

	got.now = func() time.Time { return second }
	updated, cmd = got.Update(actionResultMsg{kind: actionTerminateConn})
	got = updated.(Model)
	c.Assert(cmd, qt.Not(qt.IsNil))
	c.Assert(got.notice.text, qt.Equals, "force terminate sent")

	updated, cmd = got.Update(noticeTimeoutMsg{id: firstNoticeID})
	got = updated.(Model)

	c.Assert(cmd, qt.IsNil)
	c.Assert(got.notice.text, qt.Equals, "force terminate sent")
}

func TestModelKeepsEmptyListAndErrorDuringContinuedContention(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	list := live.NewConnectionList(time.Now(), nil, live.SortByTransactionStart)

	updated, _ := model.Update(listMsg{list: list})
	updated, _ = updated.(Model).Update(listMsg{err: errors.New("list connections: server is warming up, please retry in a moment")})
	updated, _ = updated.(Model).Update(tickMsg(time.Now()))
	view := updated.(Model).View()

	c.Assert(view, qt.Contains, "connections 0")
	c.Assert(view, qt.Contains, "●") // fixed-width refresh indicator (replaces the "refreshing" text token)
	c.Assert(view, qt.Contains, "no live connections")
	c.Assert(view, qt.Contains, "new connections appear on the next refresh")
	c.Assert(view, qt.Contains, "error: list connections: server is warming up, please retry in a moment")
}

func TestModelKeepsPopulatedListAndErrorDuringContinuedContention(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:       10,
		QueryText: "SELECT pg_sleep(120)",
	}}, live.SortByTransactionStart)

	updated, _ := model.Update(listMsg{list: list})
	updated, _ = updated.(Model).Update(listMsg{err: errors.New("list connections: server is warming up, please retry in a moment")})
	updated, _ = updated.(Model).Update(tickMsg(time.Now()))
	view := updated.(Model).View()

	c.Assert(view, qt.Contains, "connections 1")
	c.Assert(view, qt.Contains, "●") // fixed-width refresh indicator (replaces the "refreshing" text token)
	c.Assert(view, qt.Contains, "10")
	c.Assert(view, qt.Contains, "SELECT pg_sleep(120)")
	c.Assert(view, qt.Contains, "error: list connections: server is warming up, please retry in a moment")
}

func TestModelCyclesSortMode(t *testing.T) {
	c := qt.New(t)
	now := time.Now()
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	list := live.NewConnectionList(now, []live.Connection{
		{PID: 1, Duration: time.Second},
		{PID: 2, Duration: 5 * time.Second, BlockedBy: []int{3}, State: "active"},
		{PID: 3, Duration: 2 * time.Second},
	}, live.SortByTransactionStart)
	updated, _ := model.Update(listMsg{list: list})

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	got := updated.(Model)

	c.Assert(got.sort, qt.Equals, live.SortByDuration)
	c.Assert(got.lastSuccessfulList.Connections[0].PID, qt.Equals, 2)

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	got = updated.(Model)

	c.Assert(got.sort, qt.Equals, live.SortByBlocked)
	c.Assert(got.lastSuccessfulList.Connections[0].PID, qt.Equals, 3)

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	c.Assert(updated.(Model).sort, qt.Equals, live.SortByTransactionStart)
}

func TestModelVitessConnectionViewUsesOnlyDurationSort(t *testing.T) {
	c := qt.New(t)
	now := time.Now()
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithConnectionView(VitessConnectionView)
	list := live.NewConnectionList(now, []live.Connection{
		{PID: 1, Duration: time.Second},
		{PID: 2, Duration: 5 * time.Second},
	}, live.SortByDuration)
	updated, _ := model.Update(listMsg{list: list})

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	got := updated.(Model)

	c.Assert(got.sort, qt.Equals, live.SortByDuration)
	c.Assert(got.lastSuccessfulList.Connections[0].PID, qt.Equals, 2)
	c.Assert(stripANSI(got.View()), qt.Not(qt.Contains), "s sort")

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	c.Assert(stripANSI(updated.(Model).View()), qt.Not(qt.Contains), "s sort")
}

func TestModelPauseKeepsTickRefreshing(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	updated, cmd := updated.(Model).Update(tickMsg(time.Now()))

	got := updated.(Model)
	c.Assert(got.paused, qt.IsTrue)
	c.Assert(got.loading, qt.IsTrue)
	c.Assert(cmd, qt.Not(qt.IsNil))
	c.Assert(got.View(), qt.Contains, "paused")
}

func TestModelClearsPauseNoticeAfterTTL(t *testing.T) {
	c := qt.New(t)
	base := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	model.now = func() time.Time { return base }

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeySpace})
	got := updated.(Model)
	noticeID := got.notice.id

	got.now = func() time.Time { return base.Add(2 * time.Second) }
	updated, _ = got.Update(listMsg{list: live.NewConnectionList(base, nil, live.SortByTransactionStart)})
	got = updated.(Model)
	c.Assert(got.View(), qt.Contains, "resumed")

	got.now = func() time.Time { return base.Add(6 * time.Second) }
	updated, _ = got.Update(noticeTimeoutMsg{id: noticeID})
	c.Assert(updated.(Model).View(), qt.Not(qt.Contains), "resumed")
}

func TestModelArrowKeysScrollVisibleRows(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 10})
	model = updated.(Model)
	list := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 101},
		{PID: 202},
		{PID: 303},
		{PID: 404},
		{PID: 505},
		{PID: 606},
	}, live.SortByTransactionStart)
	updated, _ = model.Update(listMsg{list: list})

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	view := updated.(Model).View()

	c.Assert(view, qt.Not(qt.Contains), "101")
	c.Assert(view, qt.Contains, "505")

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyUp})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyUp})
	view = updated.(Model).View()

	c.Assert(view, qt.Contains, "101")
	c.Assert(view, qt.Not(qt.Contains), "606")
}

func TestDetailQueryTabScrollsMultilineQuery(t *testing.T) {
	c := qt.New(t)
	query := "select a\nfrom t\nwhere a = 1\nand b = 2\nand c = 3\nand d = 4\norder by a\nlimit 10"
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10, Instance: "primary", QueryText: query}}, live.SortByTransactionStart)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	updated, _ = updated.(Model).Update(listMsg{list: list})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})

	firstView := updated.(Model).View()
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	secondView := updated.(Model).View()

	c.Assert(firstView, qt.Not(qt.Equals), secondView)
	c.Assert(secondView, qt.Contains, "lines ")
}

func TestDetailQueryTabRendersVerticalRecord(t *testing.T) {
	c := qt.New(t)
	conn := live.Connection{
		PID: 42, Instance: "cell-1", InstanceRole: "replica", State: "active",
		Username: "app", ApplicationName: "worker", QueryText: "SELECT 1",
	}
	lines := connectionRecordLines(conn, connectionDisplayDefault, 120)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"pid:", "instance:", "role:", "state:", "application:", "query_id:", "connection_id:", "query:", "SELECT 1"} {
		c.Assert(joined, qt.Contains, want)
	}
}

func TestDetailQueryTabScrollsToLastLine(t *testing.T) {
	c := qt.New(t)
	// An authored multiline query has more lines than a short viewport, forcing
	// the scroll path. The final line must be reachable at max scroll.
	query := "select a, b, c\nfrom t\nwhere a = 1\nand b = 2\nand c = 3\nand d = 4\nand e = 5\nand f = 6\norder by a\nlimit 10"
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10, Instance: "primary", QueryText: query}}, live.SortByTransactionStart)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	updated, _ = updated.(Model).Update(listMsg{list: list})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Scroll well past the end; the clamp must land on the true max offset.
	for i := 0; i < 50; i++ {
		updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	view := updated.(Model).View()

	// "limit 10" is the last line; it must be visible at max scroll.
	c.Assert(view, qt.Contains, "limit 10")
}

func TestModelManualRefreshFetchesWhenIdle(t *testing.T) {
	c := qt.New(t)
	source := &clientStub{list: live.NewConnectionList(time.Now(), []live.Connection{{PID: 10}}, live.SortByDuration)}
	model := NewModel(context.Background(), source, time.Second, 0)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	c.Assert(updated.(Model).loading, qt.IsTrue)
	c.Assert(cmd, qt.Not(qt.IsNil))
	msg := cmd().(listMsg)
	c.Assert(msg.err, qt.IsNil)
	c.Assert(msg.list.Connections[0].PID, qt.Equals, 10)
	c.Assert(source.calls, qt.Equals, 1)
}

func TestModelManualRefreshSkipsWhenLoading(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	model.loading = true

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	c.Assert(cmd, qt.IsNil)
}

func TestModelTickDoesNotStartFetchWhileLoading(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	model.loading = true

	updated, cmd := model.Update(tickMsg(time.Now()))

	c.Assert(updated.(Model).loading, qt.IsTrue)
	c.Assert(cmd, qt.Not(qt.IsNil))
}

func TestModelFetchUsesParentContext(t *testing.T) {
	c := qt.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	model := NewModel(ctx, &clientStub{}, time.Second, 0)

	msg := model.fetch()().(listMsg)

	c.Assert(errors.Is(msg.err, context.Canceled), qt.IsTrue)
}

// clientStub stands in for *live.Client in TUI tests. It records List calls
// and the three action calls, with separate error fields so a test can fail
// just the read path or just the action path.
type clientStub struct {
	list  live.ConnectionList
	err   error
	calls int

	cancelCalls   int
	terminateTxn  int
	terminateConn int
	lastTarget    live.ActionTarget
	actionErr     error
}

func (s *clientStub) List(ctx context.Context, sort live.SortMode) (live.ConnectionList, error) {
	s.calls++
	if err := ctx.Err(); err != nil {
		return live.ConnectionList{}, err
	}
	return s.list, s.err
}

func (s *clientStub) CancelQuery(ctx context.Context, target live.ActionTarget) error {
	s.cancelCalls++
	s.lastTarget = target
	return s.actionErr
}

func (s *clientStub) TerminateTransaction(ctx context.Context, target live.ActionTarget) error {
	s.terminateTxn++
	s.lastTarget = target
	return s.actionErr
}

func (s *clientStub) TerminateConnection(ctx context.Context, target live.ActionTarget) error {
	s.terminateConn++
	s.lastTarget = target
	return s.actionErr
}

func vitessModelWithConnection() (Model, *clientStub) {
	client := &clientStub{}
	connectionID := "zone1-2001-101"
	queryID := "zone1-2001-101"
	model := NewModel(context.Background(), client, time.Second, 0).
		WithConnectionView(VitessConnectionView)
	list := live.NewConnectionList(time.Unix(100, 0), []live.Connection{{
		PID:          101,
		Instance:     "zone1-2001",
		ConnectionID: &connectionID,
		QueryID:      &queryID,
		QueryText:    "SELECT 1",
	}}, live.SortByDuration)
	updated, _ := model.Update(listMsg{list: list})
	return updated.(Model), client
}

func TestModelRendersPartialFailureBanner(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)
	updated, _ = model.Update(listMsg{list: live.ConnectionList{
		Instances: []live.InstanceMeta{
			{ID: "primary", Role: "primary"},
			{ID: "replica-a", Role: "replica", Error: "timeout after 2s"},
			{ID: "replica-b", Role: "replica", Error: "connection refused"},
		},
	}})

	view := updated.(Model).View()

	c.Assert(view, qt.Contains, "2 of 3 instances unreachable")
	c.Assert(view, qt.Contains, "replica-a")
	c.Assert(view, qt.Contains, "replica-b")
}

func TestModelOmitsBannerWhenAllInstancesHealthy(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)
	updated, _ = model.Update(listMsg{list: live.ConnectionList{
		Instances: []live.InstanceMeta{
			{ID: "primary", Role: "primary"},
			{ID: "replica-a", Role: "replica"},
		},
	}})

	view := updated.(Model).View()

	c.Assert(view, qt.Not(qt.Contains), "instances unreachable")
}

func TestModelRendersFreshnessRelativeAge(t *testing.T) {
	c := qt.New(t)
	captured := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	clock := captured.Add(2 * time.Second)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	model.now = func() time.Time { return clock }
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)
	updated, _ = model.Update(listMsg{list: live.NewConnectionList(captured, []live.Connection{{PID: 1}}, live.SortByTransactionStart)})

	view := updated.(Model).View()

	c.Assert(view, qt.Contains, "(2s ago)")
}

func TestModelInteractiveWritesCaptureFile(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	writer := history.NewCaptureWriter(&buf)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).WithCaptureWriter(writer)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10}}, live.SortByTransactionStart)

	updated, _ = model.Update(listMsg{list: list})
	_ = updated

	c.Assert(buf.String(), qt.Contains, `"pid":10`)
}

func TestModelCaptureWriteErrorDetachesWriterAndSetsStickyError(t *testing.T) {
	c := qt.New(t)
	target := &countingFailingWriter{err: errors.New("disk full")}
	writer := history.NewCaptureWriter(target)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).WithCaptureWriter(writer)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10}}, live.SortByTransactionStart)

	updated, _ = model.Update(listMsg{list: list})
	got := updated.(Model)

	c.Assert(got.captureStopped, qt.Contains, "capture stopped")
	c.Assert(got.captureStopped, qt.Contains, "disk full")
	c.Assert(got.capture.Writer, qt.IsNil)
	c.Assert(target.writes, qt.Equals, 1)
	c.Assert(got.View(), qt.Contains, "capture stopped")

	// A subsequent successful list must not retry the detached writer and
	// must not clear the sticky indicator.
	updated, _ = got.Update(listMsg{list: list})
	got = updated.(Model)

	c.Assert(target.writes, qt.Equals, 1)
	c.Assert(got.captureStopped, qt.Contains, "capture stopped")
	c.Assert(got.View(), qt.Contains, "capture stopped")
}

func TestModelToggleCaptureBackfillsHistoryAndTailsFutureSamples(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	openCalls := 0
	control := &CaptureControl{
		Open: func() (*history.CaptureWriter, string, error) {
			openCalls++
			return history.NewCaptureWriter(&buf), "trace.jsonl", nil
		},
	}
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).WithCaptureControl(control)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)

	for _, pid := range []int{10, 11} {
		next, _ := model.Update(listMsg{list: live.NewConnectionList(time.Unix(int64(pid), 0), []live.Connection{{
			PID: pid, Instance: "primary",
		}}, live.SortByTransactionStart)})
		model = next.(Model)
	}
	c.Assert(buf.String(), qt.Equals, "")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	model = updated.(Model)

	c.Assert(openCalls, qt.Equals, 1)
	c.Assert(buf.String(), qt.Contains, `"pid":10`)
	c.Assert(buf.String(), qt.Contains, `"pid":11`)
	c.Assert(model.View(), qt.Contains, "rec trace.jsonl")

	updated, _ = model.Update(listMsg{list: live.NewConnectionList(time.Unix(12, 0), []live.Connection{{
		PID: 12, Instance: "primary",
	}}, live.SortByTransactionStart)})
	model = updated.(Model)

	c.Assert(buf.String(), qt.Contains, `"pid":12`)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	model = updated.(Model)
	updated, _ = model.Update(listMsg{list: live.NewConnectionList(time.Unix(13, 0), []live.Connection{{
		PID: 13, Instance: "primary",
	}}, live.SortByTransactionStart)})
	model = updated.(Model)

	c.Assert(buf.String(), qt.Not(qt.Contains), `"pid":13`)
	c.Assert(model.View(), qt.Not(qt.Contains), "rec trace.jsonl")
}

func TestModelToggleCaptureBackfillsTailOnceButWritesFutureSamples(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	control := &CaptureControl{
		Open: func() (*history.CaptureWriter, string, error) {
			return history.NewCaptureWriter(&buf), "trace.jsonl", nil
		},
	}
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).WithCaptureControl(control)
	list := live.NewConnectionList(time.Unix(10, 0), []live.Connection{{
		PID: 10, Instance: "primary",
	}}, live.SortByTransactionStart)

	updated, _ := model.Update(listMsg{list: list})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	model = updated.(Model)

	c.Assert(strings.Count(buf.String(), `"pid":10`), qt.Equals, 1)

	updated, _ = model.Update(listMsg{list: list})
	_ = updated

	c.Assert(strings.Count(buf.String(), `"pid":10`), qt.Equals, 2)
}

func TestModelCaptureNoticeSurvivesRefreshUntilTTL(t *testing.T) {
	c := qt.New(t)
	base := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	control := &CaptureControl{
		Open: func() (*history.CaptureWriter, string, error) {
			return history.NewCaptureWriter(&buf), "trace.jsonl", nil
		},
	}
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).WithCaptureControl(control)
	model.now = func() time.Time { return base }

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	got := updated.(Model)
	c.Assert(got.View(), qt.Contains, "capturing to trace.jsonl")
	noticeID := got.notice.id

	got.now = func() time.Time { return base.Add(2 * time.Second) }
	updated, _ = got.Update(listMsg{list: live.NewConnectionList(base, []live.Connection{{PID: 10}}, live.SortByTransactionStart)})
	got = updated.(Model)
	c.Assert(got.View(), qt.Contains, "capturing to trace.jsonl")

	got.now = func() time.Time { return base.Add(6 * time.Second) }
	updated, _ = got.Update(noticeTimeoutMsg{id: noticeID})
	got = updated.(Model)
	c.Assert(got.View(), qt.Not(qt.Contains), "capturing to trace.jsonl")
}

type countingFailingWriter struct {
	err    error
	writes int
}

func (w *countingFailingWriter) Write([]byte) (int, error) {
	w.writes++
	return 0, w.err
}

func TestModelCancelQueryDispatchesAction(t *testing.T) {
	c := qt.New(t)
	client := &clientStub{}
	model := NewModel(context.Background(), client, time.Second, 0)

	qid := "10-7"
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID: 10, Instance: "primary", QueryID: &qid,
	}}, live.SortByTransactionStart)
	updated, _ := model.Update(listMsg{list: list})
	updated, cmd := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})

	got := updated.(Model)
	c.Assert(cmd, qt.IsNil)
	c.Assert(got.confirming, qt.IsTrue)
	c.Assert(got.pendingKind, qt.Equals, actionCancelQuery)
	c.Assert(got.View(), qt.Contains, "Cancel query")
	c.Assert(got.View(), qt.Contains, "[y/N]")
	c.Assert(client.cancelCalls, qt.Equals, 0)

	updated, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	c.Assert(cmd, qt.Not(qt.IsNil))
	c.Assert(updated.(Model).confirming, qt.IsFalse)
	_ = cmd()
	c.Assert(client.cancelCalls, qt.Equals, 1)
	c.Assert(client.lastTarget.Instance, qt.Equals, "primary")
	c.Assert(client.lastTarget.PID, qt.Equals, 10)
	c.Assert(client.lastTarget.QueryID, qt.Not(qt.IsNil))
	c.Assert(*client.lastTarget.QueryID, qt.Equals, "10-7")
}

func TestModelReadOnlyRejectsActionsBeforeConfirmation(t *testing.T) {
	c := qt.New(t)
	client := &clientStub{}
	model := NewModel(context.Background(), client, time.Second, 0).WithReadOnlyActions("not available in replay mode")
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10, Instance: "primary"}}, live.SortByTransactionStart)
	updated, _ := model.Update(listMsg{list: list})

	for _, key := range []string{"c", "k", "K"} {
		next, cmd := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		got := next.(Model)

		c.Assert(cmd, qt.IsNil)
		c.Assert(got.confirming, qt.IsFalse)
		c.Assert(got.actionError, qt.Equals, "not available in replay mode")
	}
	c.Assert(client.cancelCalls, qt.Equals, 0)
	c.Assert(client.terminateTxn, qt.Equals, 0)
	c.Assert(client.terminateConn, qt.Equals, 0)
}

func TestModelHelpOpensAndClosesFromTable(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithTarget(Target{Database: "prod", Branch: "main"})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	got := updated.(Model)

	c.Assert(got.View(), qt.Contains, "Reading The Table")
	c.Assert(got.View(), qt.Contains, "prod / main")

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyEsc})
	c.Assert(updated.(Model).View(), qt.Not(qt.Contains), "Reading The Table")
}

func TestModelHelpScrollsWhenClipped(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithTarget(Target{Database: "prod", Branch: "main"})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 8})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	first := stripANSI(updated.(Model).View())

	c.Assert(first, qt.Contains, "Reading The Table")
	c.Assert(first, qt.Not(qt.Contains), "Actions")

	for i := 0; i < 16; i++ {
		updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	second := stripANSI(updated.(Model).View())

	c.Assert(second, qt.Contains, "Actions")
	c.Assert(second, qt.Contains, "lines ")
}

func TestModelHelpUsesPostgresActionCopy(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	view := updated.(Model).View()

	c.Assert(view, qt.Contains, "Cancel the selected query")
	c.Assert(view, qt.Contains, "Kill the selected transaction")
	c.Assert(view, qt.Contains, "Force terminate the selected connection")
	c.Assert(view, qt.Not(qt.Contains), "pg_cancel_query")
}

func TestModelHelpUsesVitessActionCopy(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithConnectionView(VitessConnectionView)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	view := updated.(Model).View()

	c.Assert(view, qt.Contains, "Kill the selected query")
	c.Assert(view, qt.Contains, "Kill the selected connection")
	c.Assert(view, qt.Contains, "connection_id")
	c.Assert(view, qt.Not(qt.Contains), "idle/xact")
	c.Assert(view, qt.Not(qt.Contains), "pg_terminate_backend only if it is the same transaction")
}

func TestModelHelpBlocksActionKeys(t *testing.T) {
	c := qt.New(t)
	client := &clientStub{}
	model := NewModel(context.Background(), client, time.Second, 0)
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10, Instance: "primary"}}, live.SortByTransactionStart)
	updated, _ := model.Update(listMsg{list: list})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})

	updated, cmd := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	got := updated.(Model)

	c.Assert(cmd, qt.IsNil)
	c.Assert(got.confirming, qt.IsFalse)
	c.Assert(client.terminateConn, qt.Equals, 0)
}

func TestModelHelpIgnoredWhileConfirming(t *testing.T) {
	c := qt.New(t)
	client := &clientStub{}
	model := NewModel(context.Background(), client, time.Second, 0)
	xid := "10-42"
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10, Instance: "primary", TransactionID: &xid}}, live.SortByTransactionStart)
	updated, _ := model.Update(listMsg{list: list})
	// Open the terminate confirm prompt, then press ? — help must stay closed and
	// the confirm prompt must remain until resolved.
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	c.Assert(updated.(Model).confirming, qt.IsTrue)

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	got := updated.(Model)

	c.Assert(got.helpOpen, qt.IsFalse)
	c.Assert(got.confirming, qt.IsTrue)
	c.Assert(got.View(), qt.Not(qt.Contains), "Reading The Table")
}

func TestModelVitessConnectionViewDisablesBlockersAndTransactionKill(t *testing.T) {
	c := qt.New(t)
	model, _ := vitessModelWithConnection()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	c.Assert(updated.(Model).detailOpen, qt.IsFalse)

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	c.Assert(updated.(Model).confirming, qt.IsFalse)
	c.Assert(updated.(Model).actionError, qt.Equals, "")
}

func TestModelVitessConnectionViewCancelTargetsQueryID(t *testing.T) {
	c := qt.New(t)
	model, _ := vitessModelWithConnection()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	got := updated.(Model)

	c.Assert(got.confirming, qt.IsTrue)
	c.Assert(got.pendingTarget.QueryID, qt.Not(qt.IsNil))
	c.Assert(*got.pendingTarget.QueryID, qt.Equals, "zone1-2001-101")
}

func TestModelVitessConnectionViewForceTerminateTargetsConnectionID(t *testing.T) {
	c := qt.New(t)
	model, _ := vitessModelWithConnection()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	got := updated.(Model)

	c.Assert(got.confirming, qt.IsTrue)
	c.Assert(got.pendingTarget.ConnectionID, qt.Not(qt.IsNil))
	c.Assert(*got.pendingTarget.ConnectionID, qt.Equals, "zone1-2001-101")
}

func TestModelVitessConnectionViewUsesProcesslistActionLabels(t *testing.T) {
	c := qt.New(t)
	connectionID := "zone1-2001-101"
	queryID := "zone1-2001-101"
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithConnectionView(VitessConnectionView)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID:          101,
		Instance:     "zone1-2001",
		ConnectionID: &connectionID,
		QueryID:      &queryID,
		QueryText:    "SELECT 1",
	}}, live.SortByDuration)
	updated, _ := model.Update(listMsg{list: list})
	tableView := stripANSI(updated.(Model).View())

	c.Assert(tableView, qt.Contains, "c KILL QUERY")
	c.Assert(tableView, qt.Contains, "shift+K KILL")
	c.Assert(tableView, qt.Not(qt.Contains), "cancel query")
	c.Assert(tableView, qt.Not(qt.Contains), "force terminate")

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	detailView := stripANSI(updated.(Model).View())

	c.Assert(detailView, qt.Contains, "c KILL QUERY")
	c.Assert(detailView, qt.Contains, "shift+K KILL")
	c.Assert(detailView, qt.Not(qt.Contains), "left/right tabs")
}

func TestModelTerminateTransactionConfirmationGate(t *testing.T) {
	c := qt.New(t)
	client := &clientStub{}
	xid := "10-42"
	model := NewModel(context.Background(), client, time.Second, 0)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID: 10, Instance: "primary", TransactionID: &xid,
	}}, live.SortByTransactionStart)
	updated, _ := model.Update(listMsg{list: list})

	// Press k — should enter confirming state without firing.
	updated, cmd := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	c.Assert(cmd, qt.IsNil)
	c.Assert(updated.(Model).confirming, qt.IsTrue)
	c.Assert(client.terminateTxn, qt.Equals, 0)

	// Press y — fires.
	updated, cmd = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	c.Assert(cmd, qt.Not(qt.IsNil))
	c.Assert(updated.(Model).confirming, qt.IsFalse)
	_ = cmd()
	c.Assert(client.terminateTxn, qt.Equals, 1)
	c.Assert(*client.lastTarget.TransactionID, qt.Equals, "10-42")
}

func TestModelForceTerminateConfirmationGate(t *testing.T) {
	c := qt.New(t)
	client := &clientStub{}
	cid := "10-1779113716123456"
	model := NewModel(context.Background(), client, time.Second, 0)
	list := live.NewConnectionList(time.Now(), []live.Connection{{
		PID: 10, Instance: "primary", ConnectionID: &cid,
	}}, live.SortByTransactionStart)
	updated, _ := model.Update(listMsg{list: list})

	// Press K — should enter confirming state without firing.
	updated, cmd := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	c.Assert(cmd, qt.IsNil)
	c.Assert(updated.(Model).confirming, qt.IsTrue)
	c.Assert(client.terminateConn, qt.Equals, 0)
	c.Assert(updated.(Model).View(), qt.Contains, "[y/N]")
	c.Assert(updated.(Model).View(), qt.Not(qt.Contains), "(y/n)")

	// A random key should be a no-op (no leak to other handlers).
	updated, cmd = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	c.Assert(cmd, qt.IsNil)
	c.Assert(updated.(Model).confirming, qt.IsTrue)
	c.Assert(client.terminateConn, qt.Equals, 0)

	// Press y — fires.
	updated, cmd = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	c.Assert(cmd, qt.Not(qt.IsNil))
	c.Assert(updated.(Model).confirming, qt.IsFalse)
	_ = cmd()
	c.Assert(client.terminateConn, qt.Equals, 1)
	c.Assert(client.lastTarget.ConnectionID, qt.Not(qt.IsNil))
	c.Assert(*client.lastTarget.ConnectionID, qt.Equals, "10-1779113716123456")
}

func TestModelForceTerminateConfirmationCancel(t *testing.T) {
	c := qt.New(t)
	client := &clientStub{}
	model := NewModel(context.Background(), client, time.Second, 0)
	cid := "10-c"
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10, ConnectionID: &cid}}, live.SortByTransactionStart)
	updated, _ := model.Update(listMsg{list: list})

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	updated, cmd := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})

	c.Assert(cmd, qt.Not(qt.IsNil))
	c.Assert(updated.(Model).confirming, qt.IsFalse)
	c.Assert(client.terminateConn, qt.Equals, 0)
	c.Assert(updated.(Model).notice.text, qt.Equals, "force terminate cancelled")
}

func TestModelCancelQueryConfirmationCancel(t *testing.T) {
	c := qt.New(t)
	client := &clientStub{}
	model := NewModel(context.Background(), client, time.Second, 0)
	qid := "10-q"
	list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10, QueryID: &qid}}, live.SortByTransactionStart)
	updated, _ := model.Update(listMsg{list: list})

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	updated, cmd := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})

	c.Assert(cmd, qt.Not(qt.IsNil))
	c.Assert(updated.(Model).confirming, qt.IsFalse)
	c.Assert(client.cancelCalls, qt.Equals, 0)
	c.Assert(updated.(Model).notice.text, qt.Equals, "cancel query cancelled")
}

// Universal-quit shortcuts: ctrl+c and q must still quit during confirmation
// so an operator can always escape the gate without knowing n/esc.
func TestModelForceTerminateConfirmationHonorsUniversalQuit(t *testing.T) {
	c := qt.New(t)
	client := &clientStub{}

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyCtrlC},
		{Type: tea.KeyRunes, Runes: []rune("q")},
	} {
		model := NewModel(context.Background(), client, time.Second, 0)
		cid := "10-c"
		list := live.NewConnectionList(time.Now(), []live.Connection{{PID: 10, ConnectionID: &cid}}, live.SortByTransactionStart)
		updated, _ := model.Update(listMsg{list: list})
		updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
		c.Assert(updated.(Model).confirming, qt.IsTrue)

		_, cmd := updated.(Model).Update(key)

		c.Assert(cmd, qt.Not(qt.IsNil), qt.Commentf("key %v should produce tea.Quit during confirmation", key))
		msg := cmd()
		_, ok := msg.(tea.QuitMsg)
		c.Assert(ok, qt.IsTrue, qt.Commentf("key %v should fire tea.QuitMsg, got %T", key, msg))
		c.Assert(client.terminateConn, qt.Equals, 0, qt.Commentf("key %v must not fire the destructive action", key))
	}
}

// startConfirm() returns silently when selectedConnection() reports no
// selection. Pin that branch so a future refactor can't accidentally enter
// confirming state on an empty list.
func TestModelForceTerminateWithNoSelectionIsNoOp(t *testing.T) {
	c := qt.New(t)
	client := &clientStub{}
	model := NewModel(context.Background(), client, time.Second, 0)
	// No listMsg dispatched: hasList is false.

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})

	c.Assert(cmd, qt.IsNil)
	c.Assert(updated.(Model).confirming, qt.IsFalse)
	c.Assert(client.terminateConn, qt.Equals, 0)
}

// The dispatch tests above verify that the right client method is called but
// don't round-trip the resulting actionResultMsg back through Update. These
// two tests pin the Update-side branches: success sets a notice, error sets
// actionError without preserving a stale notice.

func TestModelActionResultSuccessSetsNotice(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)

	cases := []struct {
		kind actionKind
		want string
	}{
		{actionCancelQuery, "cancel query sent"},
		{actionTerminateTxn, "terminate transaction sent"},
		{actionTerminateConn, "force terminate sent"},
	}

	for _, tc := range cases {
		updated, cmd := model.Update(actionResultMsg{kind: tc.kind})

		c.Assert(cmd, qt.Not(qt.IsNil), qt.Commentf("kind %d should schedule notice expiry", tc.kind))
		got := updated.(Model)
		c.Assert(got.notice.text, qt.Equals, tc.want)
		c.Assert(got.lastError, qt.Equals, "")
	}
}

func TestModelActionResultErrorSetsActionError(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)

	updated, cmd := model.Update(actionResultMsg{kind: actionCancelQuery, err: errors.New("server boom")})

	c.Assert(cmd, qt.IsNil)
	got := updated.(Model)
	// Action failures go to the sticky actionError (survives refresh), not the
	// refresh-scoped lastError.
	c.Assert(got.actionError, qt.Equals, "server boom")
	c.Assert(got.lastError, qt.Equals, "")
	c.Assert(got.notice.text, qt.Equals, "")

	// A successful auto-refresh must NOT clear the action error...
	updated, _ = got.Update(listMsg{list: live.NewConnectionList(time.Now(), nil, live.SortByTransactionStart)})
	c.Assert(updated.(Model).actionError, qt.Equals, "server boom")
	// ...but the operator's next keystroke does.
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	c.Assert(updated.(Model).actionError, qt.Equals, "")
}

func TestModelActionErrorClearsStaleNotice(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(actionResultMsg{kind: actionCancelQuery})
	model = updated.(Model)
	c.Assert(model.notice.text, qt.Equals, "cancel query sent")

	updated, _ = model.Update(actionResultMsg{kind: actionCancelQuery, err: errors.New("server boom")})
	got := updated.(Model)

	c.Assert(got.actionError, qt.Equals, "server boom")
	c.Assert(got.notice.text, qt.Equals, "")
}

// Pressing a destructive action on a row missing the required ID surfaces an
// immediate error instead of prompting y/N.
func TestMissingActionIDSkipsConfirm(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	// No QueryID on the connection → cancel-query has nothing to act on.
	conn := live.Connection{PID: 7, Instance: "primary"}

	updated, cmd := model.startConfirm(actionCancelQuery, conn)
	got := updated.(Model)

	c.Assert(cmd, qt.IsNil)
	c.Assert(got.confirming, qt.IsFalse) // no destructive prompt raised
	c.Assert(got.actionError, qt.Contains, "no active query to cancel")
}

func TestDefaultConnectionCapabilitiesKeepsBlockersAndMissingIDs(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	list := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 10, Instance: "primary", BlockedBy: []int{20}, QueryText: "blocked"},
		{PID: 20, Instance: "primary", QueryText: "blocker"},
	}, live.SortByTransactionStart)
	updated, _ := model.Update(listMsg{list: list})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	got := updated.(Model)

	c.Assert(got.detailOpen, qt.IsTrue)
	c.Assert(got.detailTab, qt.Equals, tabBlockers)
	c.Assert(got.View(), qt.Contains, "BLOCKED BY")
	c.Assert(got.View(), qt.Contains, "blockers")

	updated, cmd := model.startConfirm(actionTerminateTxn, live.Connection{PID: 7, Instance: "primary"})
	got = updated.(Model)
	c.Assert(cmd, qt.IsNil)
	c.Assert(got.confirming, qt.IsFalse)
	c.Assert(got.actionError, qt.Contains, "no open transaction to terminate")

	support := DefaultConnectionCapabilities()
	c.Assert(support.missingActionID(actionCancelQuery, live.ActionTarget{}), qt.Contains, "no active query to cancel")
	c.Assert(support.missingActionID(actionTerminateTxn, live.ActionTarget{}), qt.Contains, "no open transaction to terminate")
}

func TestModelStepStepsBackThroughHistory(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)

	for i, pid := range []int{10, 11, 12} {
		list := live.NewConnectionList(time.Unix(int64(100+i), 0), []live.Connection{{
			PID: pid, Instance: "primary",
		}}, live.SortByTransactionStart)
		next, _ := model.Update(listMsg{list: list})
		model = next.(Model)
	}

	c.Assert(model.lastSuccessfulList.Connections[0].PID, qt.Equals, 12)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	got := updated.(Model)

	c.Assert(got.following, qt.IsFalse)
	c.Assert(got.lastSuccessfulList.Connections[0].PID, qt.Equals, 11)
	c.Assert(got.View(), qt.Contains, "step 2/3")
}

func TestModelStepJumpKeysWalkToOldestAndLatest(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)

	for i, pid := range []int{10, 11, 12} {
		list := live.NewConnectionList(time.Unix(int64(100+i), 0), []live.Connection{{
			PID: pid, Instance: "primary",
		}}, live.SortByTransactionStart)
		next, _ := model.Update(listMsg{list: list})
		model = next.(Model)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("{")})
	got := updated.(Model)
	c.Assert(got.following, qt.IsFalse)
	c.Assert(got.lastSuccessfulList.Connections[0].PID, qt.Equals, 10)
	c.Assert(got.View(), qt.Contains, "step 1/3")

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("}")})
	got = updated.(Model)
	c.Assert(got.following, qt.IsTrue)
	c.Assert(got.lastSuccessfulList.Connections[0].PID, qt.Equals, 12)
	pos, total := got.stepPosition()
	c.Assert(pos, qt.Equals, 0)
	c.Assert(total, qt.Equals, 0)
}

func TestModelStepHoldsViewWhenNewSamplesArrive(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)

	for i, pid := range []int{10, 11} {
		list := live.NewConnectionList(time.Unix(int64(100+i), 0), []live.Connection{{
			PID: pid, Instance: "primary",
		}}, live.SortByTransactionStart)
		next, _ := model.Update(listMsg{list: list})
		model = next.(Model)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	got := updated.(Model)
	c.Assert(got.lastSuccessfulList.Connections[0].PID, qt.Equals, 10)

	newest := live.NewConnectionList(time.Unix(102, 0), []live.Connection{{
		PID: 12, Instance: "primary",
	}}, live.SortByTransactionStart)
	updated, _ = got.Update(listMsg{list: newest})
	got = updated.(Model)

	c.Assert(got.following, qt.IsFalse)
	c.Assert(got.lastSuccessfulList.Connections[0].PID, qt.Equals, 10)
	c.Assert(got.View(), qt.Contains, "step 1/3")
}

func TestModelStepPositionHoldsUnderEviction(t *testing.T) {
	c := qt.New(t)
	h := history.NewCaptureHistory(3)
	for _, pid := range []int{10, 11, 12} {
		h.Push(live.NewConnectionList(time.Unix(int64(pid), 0), []live.Connection{{
			PID: pid, Instance: "primary",
		}}, live.SortByTransactionStart))
	}

	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).WithCaptureHistory(h)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	model = updated.(Model)
	c.Assert(model.lastSuccessfulList.Connections[0].PID, qt.Equals, 11)
	pos, total := model.stepPosition()
	c.Assert(pos, qt.Equals, 2)
	c.Assert(total, qt.Equals, 3)

	updated, _ = model.Update(listMsg{list: live.NewConnectionList(time.Unix(13, 0), []live.Connection{{
		PID: 13, Instance: "primary",
	}}, live.SortByTransactionStart)})
	model = updated.(Model)

	c.Assert(model.lastSuccessfulList.Connections[0].PID, qt.Equals, 11)
	pos, total = model.stepPosition()
	c.Assert(pos, qt.Equals, 2)
	c.Assert(total, qt.Equals, 3)
}

func TestModelJumpToLatestCatchesUpAfterPausedSamples(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)

	initial := live.NewConnectionList(time.Unix(100, 0), []live.Connection{{
		PID: 10, Instance: "primary",
	}}, live.SortByTransactionStart)
	updated, _ = model.Update(listMsg{list: initial})
	model = updated.(Model)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)

	newest := live.NewConnectionList(time.Unix(101, 0), []live.Connection{{
		PID: 11, Instance: "primary",
	}}, live.SortByTransactionStart)
	updated, _ = model.Update(listMsg{list: newest})
	model = updated.(Model)

	c.Assert(model.lastSuccessfulList.Connections[0].PID, qt.Equals, 10)
	c.Assert(model.View(), qt.Contains, "step 1/2")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("}")})
	got := updated.(Model)

	c.Assert(got.following, qt.IsTrue)
	c.Assert(got.lastSuccessfulList.Connections[0].PID, qt.Equals, 11)
	pos, total := got.stepPosition()
	c.Assert(pos, qt.Equals, 0)
	c.Assert(total, qt.Equals, 0)
}

func TestModelJumpLatestResumesLiveFollowOutsideReplay(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)

	for i, pid := range []int{10, 11} {
		updated, _ := model.Update(listMsg{list: live.NewConnectionList(time.Unix(int64(100+i), 0), []live.Connection{{
			PID: pid, Instance: "primary",
		}}, live.SortByTransactionStart)})
		model = updated.(Model)
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	c.Assert(model.paused, qt.IsTrue)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	model = updated.(Model)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("}")})
	got := updated.(Model)

	c.Assert(got.paused, qt.IsFalse)
	c.Assert(got.following, qt.IsTrue)
	c.Assert(got.liveRefresh, qt.IsTrue)
	c.Assert(got.lastSuccessfulList.Connections[0].PID, qt.Equals, 11)
}

func TestModelJumpLatestInReplayStaysReadOnlyAndPaused(t *testing.T) {
	c := qt.New(t)
	h := history.NewCaptureHistory(3)
	for i, pid := range []int{10, 11} {
		h.Push(live.NewConnectionList(time.Unix(int64(100+i), 0), []live.Connection{{
			PID: pid, Instance: "primary",
		}}, live.SortByTransactionStart))
	}
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithCaptureHistory(h).
		WithReadOnlyActions("not available in replay mode")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	model = updated.(Model)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("}")})
	got := updated.(Model)

	c.Assert(got.lastSuccessfulList.Connections[0].PID, qt.Equals, 11)
	c.Assert(got.paused, qt.IsTrue)
	c.Assert(got.following, qt.IsFalse)
	c.Assert(got.liveRefresh, qt.IsFalse)
	c.Assert(got.readOnlyActionError, qt.Equals, "not available in replay mode")
}

func TestModelStepForwardUsesSamplesCollectedWhilePaused(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)

	initial := live.NewConnectionList(time.Unix(100, 0), []live.Connection{{
		PID: 10, Instance: "primary",
	}}, live.SortByTransactionStart)
	updated, _ = model.Update(listMsg{list: initial})
	model = updated.(Model)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)

	for i, pid := range []int{11, 12} {
		list := live.NewConnectionList(time.Unix(int64(101+i), 0), []live.Connection{{
			PID: pid, Instance: "primary",
		}}, live.SortByTransactionStart)
		next, _ := model.Update(listMsg{list: list})
		model = next.(Model)
	}

	c.Assert(model.lastSuccessfulList.Connections[0].PID, qt.Equals, 10)
	c.Assert(model.View(), qt.Contains, "step 1/3")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	got := updated.(Model)

	c.Assert(got.following, qt.IsFalse)
	c.Assert(got.lastSuccessfulList.Connections[0].PID, qt.Equals, 11)
	c.Assert(got.View(), qt.Contains, "step 2/3")
}

func TestModelWithCaptureHistoryDoesNotFetchOnInit(t *testing.T) {
	c := qt.New(t)
	h := history.NewCaptureHistory(3)
	for _, pid := range []int{10, 11, 12} {
		h.Push(live.NewConnectionList(time.Unix(int64(pid), 0), []live.Connection{{
			PID: pid, Instance: "primary",
		}}, live.SortByTransactionStart))
	}
	client := &clientStub{list: live.NewConnectionList(time.Unix(99, 0), []live.Connection{{
		PID: 99, Instance: "primary",
	}}, live.SortByTransactionStart)}

	model := NewModel(context.Background(), client, time.Second, 0).WithCaptureHistory(h)

	c.Assert(model.Init(), qt.IsNil)
	c.Assert(client.calls, qt.Equals, 0)
}

func TestModelWithCaptureHistoryDoesNotRefreshOnTick(t *testing.T) {
	c := qt.New(t)
	h := history.NewCaptureHistory(3)
	for _, pid := range []int{10, 11, 12} {
		h.Push(live.NewConnectionList(time.Unix(int64(pid), 0), []live.Connection{{
			PID: pid, Instance: "primary",
		}}, live.SortByTransactionStart))
	}
	client := &clientStub{list: live.NewConnectionList(time.Unix(99, 0), []live.Connection{{
		PID: 99, Instance: "primary",
	}}, live.SortByTransactionStart)}
	model := NewModel(context.Background(), client, time.Second, 0).WithCaptureHistory(h)

	updated, cmd := model.Update(tickMsg(time.Now()))
	got := updated.(Model)

	c.Assert(cmd, qt.IsNil)
	c.Assert(got.loading, qt.IsFalse)
	c.Assert(got.lastSuccessfulList.Connections[0].PID, qt.Equals, 12)
	c.Assert(client.calls, qt.Equals, 0)
}

func TestModelReplayFooterHidesRefreshAndCaptureActions(t *testing.T) {
	c := qt.New(t)
	h := history.NewCaptureHistory(3)
	h.Push(live.NewConnectionList(time.Unix(10, 0), []live.Connection{{
		PID: 10, Instance: "primary", QueryText: "SELECT 1",
	}}, live.SortByTransactionStart))
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithCaptureHistory(h).
		WithReadOnlyActions("not available in replay mode")

	tableView := stripANSI(model.View())

	c.Assert(tableView, qt.Contains, "[ ] { } step history")
	c.Assert(tableView, qt.Contains, "enter/v detail")
	c.Assert(tableView, qt.Not(qt.Contains), "r refresh")
	c.Assert(tableView, qt.Not(qt.Contains), "C capture")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	detailView := stripANSI(updated.(Model).View())

	c.Assert(detailView, qt.Contains, "q/esc back")
	c.Assert(detailView, qt.Not(qt.Contains), "r refresh")
	c.Assert(detailView, qt.Not(qt.Contains), "C capture")
}

func TestModelEmptyReplayFrameAdvertisesStepHistoryWithoutRowActions(t *testing.T) {
	c := qt.New(t)
	h := history.NewCaptureHistory(3)
	h.Push(live.NewConnectionList(time.Unix(10, 0), []live.Connection{{
		PID: 10, Instance: "primary", QueryText: "SELECT 1",
	}}, live.SortByTransactionStart))
	h.Push(live.NewConnectionList(time.Unix(11, 0), nil, live.SortByTransactionStart))
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithCaptureHistory(h).
		WithReadOnlyActions("not available in replay mode")

	tableView := stripANSI(model.View())

	c.Assert(tableView, qt.Contains, "connections 0")
	c.Assert(tableView, qt.Contains, "[ ] { } step history")
	c.Assert(tableView, qt.Not(qt.Contains), "enter detail")
	c.Assert(tableView, qt.Not(qt.Contains), "cancel query")
	c.Assert(tableView, qt.Not(qt.Contains), "kill transaction")
	c.Assert(tableView, qt.Not(qt.Contains), "force terminate")
}

func TestModelStepHelpIsAdvertised(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	list := live.NewConnectionList(time.Unix(100, 0), []live.Connection{{
		PID: 10, Instance: "primary",
	}}, live.SortByTransactionStart)
	updated, _ = updated.(Model).Update(listMsg{list: list})
	view := updated.(Model).View()

	c.Assert(view, qt.Contains, "step history")
}

// Selection must follow the connection (by PID+instance) across a refresh that
// reorders/changes the list, not stay on a positional index.
func TestSelectionReanchorsByIdentityAcrossRefresh(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	// Sort by duration so order is deterministic by Duration desc.
	first := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 10, Instance: "primary", Duration: 30 * time.Second},
		{PID: 20, Instance: "primary", Duration: 20 * time.Second},
		{PID: 30, Instance: "primary", Duration: 10 * time.Second},
	}, live.SortByDuration)
	updated, _ := model.Update(listMsg{list: first})
	// Select the middle row (PID 20).
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	got := updated.(Model)
	sel, ok := got.selectedConnection()
	c.Assert(ok, qt.IsTrue)
	c.Assert(sel.PID, qt.Equals, 20)

	// New snapshot reorders: PID 20 is now at the TOP (longest duration).
	second := live.NewConnectionList(time.Now(), []live.Connection{
		{PID: 20, Instance: "primary", Duration: 40 * time.Second},
		{PID: 10, Instance: "primary", Duration: 30 * time.Second},
		{PID: 30, Instance: "primary", Duration: 10 * time.Second},
	}, live.SortByDuration)
	updated, _ = got.Update(listMsg{list: second})
	got = updated.(Model)

	sel, ok = got.selectedConnection()
	c.Assert(ok, qt.IsTrue)
	c.Assert(sel.PID, qt.Equals, 20) // still PID 20, even though it moved from row 1 to row 0
}

func vitessModelWithSelectedIndex(t *testing.T, selected int, conns []live.Connection) Model {
	t.Helper()
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithConnectionView(VitessConnectionView)
	updated, _ := model.Update(listMsg{list: live.NewConnectionList(time.Unix(100, 0), conns, live.SortByDuration)})
	for i := 0; i < selected; i++ {
		updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	return updated.(Model)
}

func updateModelWithVitessList(t *testing.T, model Model, conns []live.Connection) Model {
	t.Helper()
	updated, _ := model.Update(listMsg{list: live.NewConnectionList(time.Unix(101, 0), conns, live.SortByDuration)})
	return updated.(Model)
}

func TestVitessSelectionStopsFollowingQueryRowThatBecomesSleep(t *testing.T) {
	c := qt.New(t)
	model := vitessModelWithSelectedIndex(t, 2, []live.Connection{
		{PID: 10, Instance: "tablet", State: "Query/update", Duration: 4 * time.Second, QueryText: "INSERT 1"},
		{PID: 20, Instance: "tablet", State: "Query/update", Duration: 3 * time.Second, QueryText: "INSERT 2"},
		{PID: 30, Instance: "tablet", State: "Query/update", Duration: 2 * time.Second, QueryText: "INSERT 3"},
		{PID: 40, Instance: "tablet", State: "Sleep"},
	})
	sel, ok := model.selectedConnection()
	c.Assert(ok, qt.IsTrue)
	c.Assert(sel.PID, qt.Equals, 30)

	model = updateModelWithVitessList(t, model, []live.Connection{
		{PID: 50, Instance: "tablet", State: "Query/update", Duration: 5 * time.Second, QueryText: "INSERT 4"},
		{PID: 60, Instance: "tablet", State: "Query/update", Duration: 1 * time.Second, QueryText: "INSERT 5"},
		{PID: 30, Instance: "tablet", State: "Sleep"},
		{PID: 40, Instance: "tablet", State: "Sleep"},
	})

	sel, ok = model.selectedConnection()
	c.Assert(ok, qt.IsTrue)
	c.Assert(sel.PID, qt.Equals, 60)
}

func TestVitessSelectionKeepsSleepingIdentityThatBecomesActive(t *testing.T) {
	c := qt.New(t)
	model := vitessModelWithSelectedIndex(t, 2, []live.Connection{
		{PID: 10, Instance: "tablet", State: "Query/update", Duration: 2 * time.Second, QueryText: "INSERT 1"},
		{PID: 20, Instance: "tablet", State: "Sleep"},
		{PID: 30, Instance: "tablet", State: "Sleep"},
	})

	model = updateModelWithVitessList(t, model, []live.Connection{
		{PID: 30, Instance: "tablet", State: "Query/update", Duration: 5 * time.Second, QueryText: "INSERT 2"},
		{PID: 20, Instance: "tablet", State: "Sleep"},
	})

	sel, ok := model.selectedConnection()
	c.Assert(ok, qt.IsTrue)
	c.Assert(sel.PID, qt.Equals, 30)
}

func TestVitessSelectionKeepsSleepingIdentityThatStaysSleep(t *testing.T) {
	c := qt.New(t)
	model := vitessModelWithSelectedIndex(t, 2, []live.Connection{
		{PID: 10, Instance: "tablet", State: "Query/update", Duration: 2 * time.Second, QueryText: "INSERT 1"},
		{PID: 20, Instance: "tablet", State: "Sleep"},
		{PID: 30, Instance: "tablet", State: "Sleep"},
	})

	model = updateModelWithVitessList(t, model, []live.Connection{
		{PID: 40, Instance: "tablet", State: "Query/update", Duration: 3 * time.Second, QueryText: "INSERT 2"},
		{PID: 20, Instance: "tablet", State: "Sleep"},
		{PID: 30, Instance: "tablet", State: "Sleep"},
	})

	sel, ok := model.selectedConnection()
	c.Assert(ok, qt.IsTrue)
	c.Assert(sel.PID, qt.Equals, 30)
}

func TestVitessSelectionFallsBackToNearestActiveRowWhenSelectedConnectionEnds(t *testing.T) {
	c := qt.New(t)
	model := vitessModelWithSelectedIndex(t, 2, []live.Connection{
		{PID: 10, Instance: "tablet", State: "Query/update", Duration: 4 * time.Second, QueryText: "INSERT 1"},
		{PID: 20, Instance: "tablet", State: "Query/update", Duration: 3 * time.Second, QueryText: "INSERT 2"},
		{PID: 30, Instance: "tablet", State: "Query/update", Duration: 2 * time.Second, QueryText: "INSERT 3"},
		{PID: 40, Instance: "tablet", State: "Sleep"},
	})

	model = updateModelWithVitessList(t, model, []live.Connection{
		{PID: 50, Instance: "tablet", State: "Query/update", Duration: 5 * time.Second, QueryText: "INSERT 4"},
		{PID: 60, Instance: "tablet", State: "Query/update", Duration: 1 * time.Second, QueryText: "INSERT 5"},
		{PID: 40, Instance: "tablet", State: "Sleep"},
		{PID: 70, Instance: "tablet", State: "Sleep"},
	})

	sel, ok := model.selectedConnection()
	c.Assert(ok, qt.IsTrue)
	c.Assert(sel.PID, qt.Equals, 60)
}

func TestVitessSelectionKeepsIndexWhenOnlySleepingRowsRemain(t *testing.T) {
	c := qt.New(t)
	model := vitessModelWithSelectedIndex(t, 2, []live.Connection{
		{PID: 10, Instance: "tablet", State: "Query/update", Duration: 4 * time.Second, QueryText: "INSERT 1"},
		{PID: 20, Instance: "tablet", State: "Query/update", Duration: 3 * time.Second, QueryText: "INSERT 2"},
		{PID: 30, Instance: "tablet", State: "Query/update", Duration: 2 * time.Second, QueryText: "INSERT 3"},
	})

	model = updateModelWithVitessList(t, model, []live.Connection{
		{PID: 40, Instance: "tablet", State: "Sleep"},
		{PID: 50, Instance: "tablet", State: "Sleep"},
		{PID: 60, Instance: "tablet", State: "Sleep"},
		{PID: 70, Instance: "tablet", State: "Sleep"},
	})

	sel, ok := model.selectedConnection()
	c.Assert(ok, qt.IsTrue)
	c.Assert(sel.PID, qt.Equals, 60)
}

func TestRefreshDotReflectsLiveStateWhilePaused(t *testing.T) {
	c := qt.New(t)
	defer lipgloss.SetColorProfile(termenv.Ascii)
	lipgloss.SetColorProfile(termenv.ANSI256)
	prevBG := lipgloss.HasDarkBackground()
	defer lipgloss.SetHasDarkBackground(prevBG)
	lipgloss.SetHasDarkBackground(true)

	const (
		dimDot = "\x1b[38;5;240m●"
		redDot = "\x1b[1;38;5;196m●"
	)

	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	model = updated.(Model)
	// Two successful samples → history to step through, and a healthy dim dot.
	for _, ts := range []int64{100, 101} {
		list := live.NewConnectionList(time.Unix(ts, 0), []live.Connection{{PID: 10, State: "active"}}, live.SortByTransactionStart)
		updated, _ = model.Update(listMsg{list: list})
		model = updated.(Model)
	}
	c.Assert(model.View(), qt.Contains, dimDot)

	// Two consecutive failed fetches → sustained failure → red dot.
	updated, _ = model.Update(listMsg{err: errors.New("503 from api-bb")})
	model = updated.(Model)
	updated, _ = model.Update(listMsg{err: errors.New("503 from api-bb")})
	model = updated.(Model)
	c.Assert(model.View(), qt.Contains, redDot)

	// Pause and step back to a healthy historical frame; the dot must stay red.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	model = updated.(Model)
	view := model.View()
	c.Assert(view, qt.Contains, "paused")
	c.Assert(view, qt.Contains, "step")
	c.Assert(view, qt.Contains, redDot)

	// A successful refresh clears the streak → back to the dim idle dot.
	list := live.NewConnectionList(time.Unix(102, 0), []live.Connection{{PID: 10, State: "active"}}, live.SortByTransactionStart)
	updated, _ = model.Update(listMsg{list: list})
	c.Assert(updated.(Model).View(), qt.Contains, dimDot)
}

func TestDetailRefreshIgnoredInReplay(t *testing.T) {
	c := qt.New(t)
	h := history.NewCaptureHistory(3)
	h.Push(live.NewConnectionList(time.Unix(100, 0), []live.Connection{{
		PID: 10, Instance: "primary",
	}}, live.SortByTransactionStart))
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithCaptureHistory(h).
		WithReadOnlyActions("not available in replay mode")
	model.detailOpen = true
	model.detailInstance = "primary"
	model.detailPID = 10

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	got := updated.(Model)

	c.Assert(cmd, qt.IsNil)
	c.Assert(got.loading, qt.IsFalse)
	c.Assert(got.samples.Len(), qt.Equals, 1)
}

func TestModelInstanceGoneMidSessionRewordsError(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ := model.Update(listMsg{list: live.NewConnectionList(time.Unix(100, 0), []live.Connection{{
		PID: 10, Instance: "primary",
	}}, live.SortByTransactionStart)})
	model = updated.(Model)

	updated, _ = model.Update(listMsg{err: &live.UnknownInstanceError{Instance: "replica-1", Valid: []string{"primary"}}})
	got := updated.(Model)
	c.Assert(got.lastError, qt.Equals, `instance "replica-1" is no longer in the branch's instance set`)

	fresh := NewModel(context.Background(), &clientStub{}, time.Second, 0)
	updated, _ = fresh.Update(listMsg{err: &live.UnknownInstanceError{Instance: "replica-1", Valid: []string{"primary"}}})
	c.Assert(updated.(Model).lastError, qt.Contains, "unknown instance")
}

func TestModelPreListPausedFooterHidesRefresh(t *testing.T) {
	c := qt.New(t)
	model := NewModel(context.Background(), &clientStub{}, time.Second, 0).
		WithTarget(Target{Database: "prod", Branch: "main"})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	view := stripANSI(updated.(Model).View())

	c.Assert(view, qt.Contains, "space resume")
	c.Assert(view, qt.Not(qt.Contains), "r refresh")
}
