package tui

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	live "github.com/planetscale/cli/internal/connections"
	"github.com/planetscale/cli/internal/connections/history"
)

const (
	// defaultHistoryCapacity bounds the in-memory ring of recent samples kept for
	// stepping. ~5 minutes at the default 1s interval.
	defaultHistoryCapacity = 300
	noticeTTL              = 5 * time.Second
)

type listMsg struct {
	list live.ConnectionList
	err  error
}

type tickMsg time.Time

type noticeTimeoutMsg struct {
	id uint64
}

type durationDoneMsg struct{}

type noticeState struct {
	id   uint64
	text string
}

type actionKind int

const (
	actionCancelQuery actionKind = iota + 1
	actionTerminateTxn
	actionTerminateConn
)

type actionResultMsg struct {
	kind actionKind
	err  error
}

type connectionDisplayPreset string

const (
	connectionDisplayDefault     connectionDisplayPreset = ""
	connectionDisplayProcesslist connectionDisplayPreset = "processlist"
)

type ConnectionViewProfile struct {
	displayPreset connectionDisplayPreset
	capabilities  ConnectionCapabilities
	defaultSort   live.SortMode
	sortModes     []live.SortMode
}

var (
	// PostgresConnectionView presents live connections with Postgres session semantics.
	PostgresConnectionView = ConnectionViewProfile{
		displayPreset: connectionDisplayDefault,
		capabilities:  DefaultConnectionCapabilities(),
		defaultSort:   live.SortByTransactionStart,
		sortModes:     postgresSortModes(),
	}
	// VitessConnectionView presents live connections with Vitess processlist semantics.
	VitessConnectionView = ConnectionViewProfile{
		displayPreset: connectionDisplayProcesslist,
		capabilities: ConnectionCapabilities{
			CancelQuery:         ActionTargetQueryID,
			TerminateConnection: ActionTargetConnectionID,
			ShowBlockers:        false,
			configured:          true,
		},
		defaultSort: live.SortByDuration,
		sortModes:   []live.SortMode{live.SortByDuration},
	}
)

func postgresSortModes() []live.SortMode {
	return []live.SortMode{
		live.SortByTransactionStart,
		live.SortByDuration,
		live.SortByBlocked,
	}
}

// DefaultSort is the sort order a connection source should use for this view.
func (p ConnectionViewProfile) DefaultSort() live.SortMode {
	return p.defaultSort
}

func (p ConnectionViewProfile) sortOptions() []live.SortMode {
	if len(p.sortModes) == 0 {
		return []live.SortMode{p.defaultSort}
	}
	return append([]live.SortMode(nil), p.sortModes...)
}

type Target struct {
	Database string
	Branch   string
	Keyspace string
	Shard    string
}

// ConnectionsClient is the live-connections data + action dependency the
// view model talks to. Production wires this to the wire-level *live.Client
// (directly or through a filtering wrapper); tests provide a stub that records
// calls.
type ConnectionsClient interface {
	List(context.Context, live.SortMode) (live.ConnectionList, error)
	CancelQuery(context.Context, live.ActionTarget) error
	TerminateTransaction(context.Context, live.ActionTarget) error
	TerminateConnection(context.Context, live.ActionTarget) error
}

// Model is the Bubble Tea view model for the interactive table.
type Model struct {
	client    ConnectionsClient
	ctx       context.Context
	interval  time.Duration
	duration  time.Duration
	sort      live.SortMode
	sortModes []live.SortMode
	now       func() time.Time
	capture   *CaptureControl
	target    Target
	filter    string

	displayPreset connectionDisplayPreset
	capabilities  ConnectionCapabilities

	samples     *history.CaptureHistory
	cursor      history.CaptureCursor
	following   bool
	liveRefresh bool

	// stepAnchorBase is the history base recorded when the cursor last moved.
	// The step-history numerator counts from this anchor rather than the live
	// oldest edge, so a held frame's position holds steady while paused even as
	// eviction advances the live base underneath it.
	stepAnchorBase history.CaptureCursor

	lastSuccessfulList  live.ConnectionList
	hasList             bool
	lastError           string
	initialAccessDenied bool
	// actionError holds the result of an explicit user action (cancel/kill, a
	// permission denial, or a "nothing to act on" guard). Unlike lastError —
	// which a successful auto-refresh clears — it survives refreshes and is
	// cleared only by the operator's next keystroke, so a destructive action's
	// outcome can't flash past unread on the ~1s refresh.
	actionError    string
	notice         noticeState
	captureStopped string
	paused         bool
	loading        bool
	// consecutiveErrors counts list fetches that have failed in a row; reset to
	// 0 on the next success. Drives the header refresh dot: 1 reads as a
	// transient blip (cyan), refreshDotFailThreshold+ reads as a sustained
	// outage (red).
	consecutiveErrors   int
	selected            int
	viewportStart       int
	width               int
	height              int
	confirming          bool
	pendingTarget       live.ActionTarget
	pendingKind         actionKind
	readOnlyActionError string
	helpOpen            bool
	helpOffset          int

	detailOpen     bool
	detailInstance string
	detailPID      int
	detailTab      detailTab
	blockerRow     int
	queryOffset    int
}

func NewModel(ctx context.Context, client ConnectionsClient, interval, duration time.Duration) Model {
	return Model{
		client:       client,
		ctx:          ctx,
		interval:     interval,
		duration:     duration,
		sort:         live.SortByTransactionStart,
		sortModes:    postgresSortModes(),
		now:          time.Now,
		width:        100,
		height:       24,
		detailTab:    tabQuery,
		capabilities: DefaultConnectionCapabilities(),
		samples:      history.NewCaptureHistory(defaultHistoryCapacity),
		following:    true,
		liveRefresh:  true,
	}
}

func (m Model) WithTarget(target Target) Model {
	m.target = target
	return m
}

// WithFilter records the active row filter (e.g. "filter: role=primary") so the
// header can show that the view is scoped. Empty when no filter is active.
func (m Model) WithFilter(filter string) Model {
	m.filter = filter
	return m
}

func (m Model) WithConnectionView(profile ConnectionViewProfile) Model {
	m.displayPreset = profile.displayPreset
	m.capabilities = profile.capabilities.effective()
	m.sort = profile.DefaultSort()
	m.sortModes = profile.sortOptions()
	return m
}

func (m Model) WithReadOnlyActions(message string) Model {
	m.readOnlyActionError = message
	return m
}

// WithCaptureWriter persists every successful list to the capture trace file
// while the TUI runs. Returns the model unchanged when w is nil.
func (m Model) WithCaptureWriter(w *history.CaptureWriter) Model {
	if w != nil {
		m.capture = &CaptureControl{Writer: w}
	}
	return m
}

func (m Model) WithCaptureControl(control *CaptureControl) Model {
	m.capture = control
	return m
}

// WithCaptureHistory replaces the default in-memory ring with one preloaded
// by the caller, used by replay mode to seed the model with every captured
// snapshot up front. The cursor points at the latest capture, rendering
// reflects it immediately, and the model starts paused so the operator
// controls advance via the step keybindings instead of an auto-tick chewing
// through the trace.
func (m Model) WithCaptureHistory(h *history.CaptureHistory) Model {
	m.samples = h
	m.liveRefresh = false
	if cursor, ok := h.Latest(); ok {
		m.cursor = cursor
		m.lastSuccessfulList = m.currentList()
		m.hasList = true
		m.paused = true
		m.recordStepPosition()
	}
	return m
}

// isReplay reports whether the model is driving a replayed trace rather than a
// live source.
func (m Model) isReplay() bool {
	return !m.liveRefresh
}

func (m Model) setNotice(text string) (Model, tea.Cmd) {
	m.notice.id++
	m.notice.text = text
	id := m.notice.id
	return m, tea.Tick(noticeTTL, func(time.Time) tea.Msg {
		return noticeTimeoutMsg{id: id}
	})
}

func (m Model) clearNotice() Model {
	m.notice.text = ""
	return m
}

func (m Model) rejectReadOnlyAction() (Model, bool) {
	if m.readOnlyActionError == "" {
		return m, false
	}
	m = m.setActionError(m.readOnlyActionError)
	return m, true
}

func (m Model) setActionError(text string) Model {
	m.actionError = text
	return m.clearNotice()
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.liveRefresh && !m.hasList {
		cmds = append(cmds, m.fetch())
	}
	if m.liveRefresh {
		cmds = append(cmds, m.tick())
	}
	cmds = append(cmds, m.durationTimer())
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampViewport()
		m.clampBlockerSelection()
		m.clampQueryOffset()
		return m, tea.ClearScreen
	case tickMsg:
		if !m.liveRefresh {
			return m, nil
		}
		if m.loading {
			return m, m.tick()
		}
		m.loading = true
		return m, tea.Batch(m.fetch(), m.tick())
	case noticeTimeoutMsg:
		if msg.id == m.notice.id {
			m = m.clearNotice()
		}
		return m, nil
	case listMsg:
		m.loading = false
		if msg.err != nil {
			m.lastError = listErrorText(msg.err, m.hasList)
			m.initialAccessDenied = !m.hasList && isForbiddenHTTPError(msg.err)
			m.consecutiveErrors++
			return m, nil
		}
		m.consecutiveErrors = 0
		m.initialAccessDenied = false
		cursor := m.samples.Push(msg.list)
		prevCursor := m.cursor
		if !m.hasList {
			m.cursor = cursor
			m.following = !m.paused
		} else if m.paused {
			if _, ok := m.samples.At(m.cursor); !ok {
				m.cursor, _ = m.samples.Oldest()
			}
			m.following = false
		} else if m.following {
			m.cursor = cursor
		} else if _, ok := m.samples.At(m.cursor); !ok {
			m.cursor, _ = m.samples.Oldest()
		}
		// Re-anchor the step label only when the cursor actually moved (initial
		// load, follow-live advance, or an eviction reset). A held paused frame
		// keeps its label so it does not drift as the buffer tail grows.
		if m.cursor != prevCursor || !m.hasList {
			m.recordStepPosition()
		}
		prevConn, hadSelection := m.selectedConnection()
		prevIndex := m.selected
		m.lastSuccessfulList = m.currentList()
		m.hasList = true
		m.lastError = ""
		if hadSelection {
			m.reanchorSelection(prevConn, prevIndex)
		}
		m.clampViewport()
		m.clampBlockerSelection()
		m.clampQueryOffset()
		if m.capture != nil && m.capture.Writer != nil {
			if err := m.capture.Writer.Write(history.NewCapture(msg.list)); err != nil {
				// Detach the writer so we stop calling Write on a dead
				// destination (ENOSPC, EPIPE, EBADF won't recover), and
				// stash the reason in a sticky field that listMsg does
				// not clear — m.lastError gets reset on every successful
				// refresh, which would flash the warning instead of
				// holding it.
				m.captureStopped = "capture stopped: " + err.Error()
				_ = m.capture.Close()
			}
		}
		return m, nil
	case durationDoneMsg:
		return m, tea.Quit
	case actionResultMsg:
		if msg.err != nil {
			m = m.setActionError(live.UserFacingErrorText(msg.err, "modify"))
			return m, nil
		}
		var cmd tea.Cmd
		m, cmd = m.setNotice(actionNotice(msg.kind) + " sent")
		m.lastError = ""
		return m, cmd
	case tea.KeyMsg:
		// An action error (cancel/kill outcome, permission denial, "nothing to
		// act on" guard) is sticky until the operator's next keystroke, so it
		// can't be wiped by an auto-refresh before it's read. Clear it here, on
		// that next keystroke. A keystroke that starts a fresh action sets it
		// again below.
		m.actionError = ""
		if m.confirming {
			return m.handleConfirmKey(msg)
		}
		if m.helpOpen {
			return m.handleHelpKey(msg)
		}
		if m.detailOpen {
			return m.handleDetailKey(msg)
		}
		return m.handleTableKey(msg)
	}
	return m, nil
}

func (m Model) handleTableKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.helpOpen = true
		m.helpOffset = 0
		return m, nil
	case "r":
		if !m.liveRefresh {
			return m, nil
		}
		if m.loading {
			return m, nil
		}
		m.loading = true
		return m, m.fetch()
	case " ":
		return m.togglePause()
	case "C":
		if m.capture == nil {
			return m, nil
		}
		return m.toggleCapture()
	case "s":
		next, ok := nextSort(m.sort, m.sortModes)
		if !ok {
			return m, nil
		}
		m.sort = next
		if m.hasList {
			m.lastSuccessfulList = m.currentList()
			m.clampViewport()
		}
		return m, nil
	case "[", "]", "{", "}":
		return m.handleStepKey(msg.String()), nil
	case "c":
		if next, rejected := m.rejectReadOnlyAction(); rejected {
			return next, nil
		}
		conn, ok := m.selectedConnection()
		if !ok {
			return m, nil
		}
		return m.startConfirm(actionCancelQuery, conn)
	case "k", "K":
		if next, rejected := m.rejectReadOnlyAction(); rejected {
			return next, nil
		}
		conn, ok := m.selectedConnection()
		if !ok {
			return m, nil
		}
		kind := actionTerminateTxn
		if msg.String() == "K" {
			kind = actionTerminateConn
		}
		return m.startConfirm(kind, conn)
	case "enter", "v", "V", "b", "B":
		conn, ok := m.selectedConnection()
		if !ok {
			return m, nil
		}
		if strings.EqualFold(msg.String(), "b") && !m.capabilities.effective().ShowBlockers {
			return m, nil
		}
		m.detailOpen = true
		m.detailInstance = conn.Instance
		m.detailPID = conn.PID
		m.blockerRow = 0
		m.queryOffset = 0
		if strings.EqualFold(msg.String(), "b") {
			m.detailTab = tabBlockers
			m.clampBlockerSelection()
		} else {
			m.detailTab = tabQuery
		}
		return m, nil
	case "up":
		m.moveSelection(-1)
		return m, nil
	case "down":
		m.moveSelection(1)
		return m, nil
	}
	return m, nil
}

func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "?":
		m.helpOpen = true
		m.helpOffset = 0
		return m, nil
	case "q", "esc", "backspace":
		m.detailOpen = false
		m.blockerRow = 0
		m.queryOffset = 0
		return m, nil
	case "r":
		if !m.liveRefresh {
			return m, nil
		}
		if m.loading {
			return m, nil
		}
		m.loading = true
		return m, m.fetch()
	case " ":
		return m.togglePause()
	case "C":
		if m.capture == nil {
			return m, nil
		}
		return m.toggleCapture()
	case "b", "B":
		if !m.capabilities.effective().ShowBlockers {
			return m, nil
		}
		m.detailTab = tabBlockers
		m.queryOffset = 0
		m.clampBlockerSelection()
		return m, nil
	case "left", "right":
		if !m.capabilities.effective().ShowBlockers {
			m.detailTab = tabQuery
			m.queryOffset = 0
			m.clampQueryOffset()
			return m, nil
		}
		if m.detailTab == tabBlockers {
			m.detailTab = tabQuery
			m.queryOffset = 0
			m.clampQueryOffset()
		} else {
			m.detailTab = tabBlockers
			m.queryOffset = 0
			m.clampBlockerSelection()
			m.clampQueryOffset()
		}
		return m, nil
	case "up":
		if m.detailTab == tabQuery && m.queryOffset > 0 {
			m.queryOffset--
			return m, nil
		}
		if m.detailTab == tabBlockers && m.blockerRow > 0 {
			m.blockerRow--
		}
		return m, nil
	case "down":
		if m.detailTab == tabQuery {
			m.queryOffset++
			m.clampQueryOffset()
			return m, nil
		}
		if m.detailTab == tabBlockers {
			m.blockerRow++
			m.clampBlockerSelection()
		}
		return m, nil
	case "enter":
		if !m.capabilities.effective().ShowBlockers {
			return m, nil
		}
		if m.detailTab == tabBlockers {
			target, ok := m.actionTargetConnection()
			if ok {
				m.detailInstance = target.Instance
				m.detailPID = target.PID
				m.blockerRow = 0
				m.queryOffset = 0
				m.clampBlockerSelection()
				m.clampQueryOffset()
			}
		}
		return m, nil
	case "c":
		if next, rejected := m.rejectReadOnlyAction(); rejected {
			return next, nil
		}
		conn, ok := m.actionTargetConnection()
		if !ok {
			return m.rejectEndedDetailAction(), nil
		}
		return m.startConfirm(actionCancelQuery, conn)
	case "k", "K":
		if next, rejected := m.rejectReadOnlyAction(); rejected {
			return next, nil
		}
		conn, ok := m.actionTargetConnection()
		if !ok {
			return m.rejectEndedDetailAction(), nil
		}
		kind := actionTerminateTxn
		if msg.String() == "K" {
			kind = actionTerminateConn
		}
		return m.startConfirm(kind, conn)
	case "[", "]", "{", "}":
		return m.handleStepKey(msg.String()), nil
	}
	return m, nil
}

func (m Model) rejectEndedDetailAction() Model {
	if m.detailOpen {
		m.actionError = "connection ended — actions unavailable; esc to go back"
		m = m.clearNotice()
	}
	return m
}

func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", "esc", "q":
		m.helpOpen = false
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "up":
		m.helpOffset = clampInt(m.helpOffset-1, 0, maxHelpOffset(m.helpState()))
		return m, nil
	case "down":
		m.helpOffset = clampInt(m.helpOffset+1, 0, maxHelpOffset(m.helpState()))
		return m, nil
	case "pgup":
		m.helpOffset = clampInt(m.helpOffset-helpBodyHeight(m.height), 0, maxHelpOffset(m.helpState()))
		return m, nil
	case "pgdown":
		m.helpOffset = clampInt(m.helpOffset+helpBodyHeight(m.height), 0, maxHelpOffset(m.helpState()))
		return m, nil
	}
	return m, nil
}

func helpBodyHeight(height int) int {
	if height <= 1 {
		return 1
	}
	return height - 1
}

func (m Model) helpState() helpState {
	return helpState{Target: m.target, Width: m.width, Height: m.height, Offset: m.helpOffset, CanSort: m.canChangeSort(), Paused: m.paused, Replay: m.isReplay(), DisplayPreset: m.displayPreset, Capabilities: m.capabilities}
}

func (m Model) handleStepKey(key string) Model {
	var (
		cursor history.CaptureCursor
		ok     bool
	)
	switch key {
	case "[":
		cursor, ok = m.samples.Step(m.cursor, -1)
	case "]":
		cursor, ok = m.samples.Step(m.cursor, 1)
	case "{":
		cursor, ok = m.samples.Oldest()
		ok = ok && cursor != m.cursor
	case "}":
		cursor, ok = m.samples.Latest()
		ok = ok && (cursor != m.cursor || m.canResumeLiveFollow())
	}
	if !ok {
		return m
	}
	m.cursor = cursor
	latest, hasLatest := m.samples.Latest()
	if hasLatest && cursor == latest && m.canResumeLiveFollow() {
		m.following = true
		m.paused = false
	} else {
		m.following = false
	}
	m.recordStepPosition()
	m.lastSuccessfulList = m.currentList()
	m.clampViewport()
	m.clampBlockerSelection()
	m.clampQueryOffset()
	return m
}

func (m Model) togglePause() (Model, tea.Cmd) {
	m.paused = !m.paused
	if m.paused {
		m.following = false
		m.recordStepPosition()
		return m.setNotice("paused")
	}
	var cmd tea.Cmd
	m, cmd = m.setNotice("resumed")
	if cursor, ok := m.samples.Latest(); ok {
		m.cursor = cursor
		m.following = true
		m.recordStepPosition()
		m.lastSuccessfulList = m.currentList()
		m.clampViewport()
		m.clampBlockerSelection()
		m.clampQueryOffset()
	}
	return m, cmd
}

func (m Model) View() string {
	if m.helpOpen {
		return renderHelpModal(m.helpState())
	}
	if m.detailOpen {
		return renderDetail(m.detailState())
	}
	return renderTable(m.tableState())
}

func (m Model) tableState() tableState {
	pos, total := m.stepPosition()
	return tableState{
		List:            m.lastSuccessfulList,
		HasList:         m.hasList,
		Sort:            m.sort,
		CanSort:         m.canChangeSort(),
		Selected:        m.selected,
		ViewportStart:   m.viewportStart,
		Width:           m.width,
		Height:          m.height,
		Paused:          m.paused,
		Refresh:         computeRefreshDot(m.loading, m.consecutiveErrors, m.isReplay()),
		ReadOnlyActions: m.readOnlyActionError != "",
		Replay:          m.isReplay(),
		LastError:       firstNonEmpty(m.actionError, m.lastError),
		AccessDenied:    m.initialAccessDenied,
		CanStepHistory:  m.samples.Len() > 0,
		Notice:          m.notice.text,
		CaptureStopped:  m.captureStopped,
		CaptureStatus:   m.captureStatusText(),
		Confirm:         m.confirmPrompt(),
		Now:             m.now(),
		Interval:        m.interval,
		StepPos:         pos,
		StepTotal:       total,
		Target:          m.target,
		Filter:          m.filter,
		DisplayPreset:   m.displayPreset,
		Capabilities:    m.capabilities,
	}
}

// currentList returns the capture at the model's cursor, sorted by m.sort.
// SortConnections runs in place on the history-owned slice; row order drifts
// across sort changes but values are unchanged.
func (m Model) currentList() live.ConnectionList {
	list, ok := m.samples.At(m.cursor)
	if !ok {
		return live.ConnectionList{}
	}
	live.SortConnections(list.Connections, m.sort)
	list.Sort = m.sort
	return list
}

// stepPosition returns the 1-based position of the displayed capture and the
// total captures held in history, but only while the user is stepping back
// from latest. When following live, both are zero so the header omits the
// indicator. The numerator counts from stepAnchorBase (the base when the
// cursor last moved), so a held frame's position does not drift backward while
// paused as eviction advances the live base.
func (m Model) stepPosition() (pos, total int) {
	if m.following {
		return 0, 0
	}
	total = m.samples.Len()
	if total == 0 {
		return 0, 0
	}
	pos = int(m.cursor-m.stepAnchorBase) + 1
	if pos < 1 {
		pos = 1
	}
	if pos >= total {
		return 0, 0
	}
	return pos, total
}

// recordStepPosition re-anchors the step numerator to the current live base.
// Call it only when the cursor actually moves (step keys, pause/resume,
// eviction reset) — never on a routine push that leaves the held cursor in
// place, or the numerator would drift backward as eviction advances the base.
func (m *Model) recordStepPosition() {
	if oldest, ok := m.samples.Oldest(); ok {
		m.stepAnchorBase = oldest
	}
}

func (m Model) startConfirm(kind actionKind, conn live.Connection) (tea.Model, tea.Cmd) {
	target := actionTargetFor(conn)
	if !m.capabilities.supports(kind) {
		return m, nil
	}
	// Don't raise a destructive confirmation for an action that can't run: if
	// the required server-issued ID is absent (e.g. an idle backend with no
	// active query, or a partial row), surface that immediately instead of
	// prompting y/N and only revealing "X is required" after the operator
	// confirms.
	if reason := m.capabilities.missingActionID(kind, target); reason != "" {
		m = m.setActionError(reason)
		return m, nil
	}
	m.confirming = true
	m.pendingKind = kind
	m.pendingTarget = target
	return m, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// listErrorText renders a refresh failure. An instance filter that stops
// matching mid-session means the instance left the branch's instance set, not
// that the operator mistyped it — reword so the live view stays honest.
func listErrorText(err error, hasList bool) string {
	var unknownInstance *live.UnknownInstanceError
	if hasList && errors.As(err, &unknownInstance) {
		return fmt.Sprintf("instance %q is no longer in the branch's instance set", unknownInstance.Instance)
	}
	return live.UserFacingErrorText(err, "view")
}

func isForbiddenHTTPError(err error) bool {
	var httpErr *live.HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusForbidden
}

func (m Model) canResumeLiveFollow() bool {
	return m.liveRefresh
}

func actionTargetFor(conn live.Connection) live.ActionTarget {
	return live.ActionTarget{
		Instance:      conn.Instance,
		PID:           conn.PID,
		ConnectionID:  conn.ConnectionID,
		TransactionID: conn.TransactionID,
		QueryID:       conn.QueryID,
	}
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		target := m.pendingTarget
		kind := m.pendingKind
		m.confirming = false
		m.pendingTarget = live.ActionTarget{}
		m.pendingKind = 0
		return m, m.fireAction(kind, target)
	case "n", "esc", "enter":
		kind := m.pendingKind
		m.confirming = false
		m.pendingTarget = live.ActionTarget{}
		m.pendingKind = 0
		return m.setNotice(actionNotice(kind) + " cancelled")
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) confirmPrompt() string {
	if !m.confirming {
		return ""
	}
	var verb string
	switch m.pendingKind {
	case actionCancelQuery:
		verb = "Cancel query on"
	case actionTerminateTxn:
		verb = "Terminate transaction on"
	case actionTerminateConn:
		verb = "Force terminate"
	default:
		return ""
	}
	return fmt.Sprintf("%s PID %d on %s? [y/N]", verb, m.pendingTarget.PID, m.pendingTarget.Instance)
}

func (m Model) fetch() tea.Cmd {
	return func() tea.Msg {
		list, err := m.client.List(m.ctx, m.sort)
		return listMsg{list: list, err: err}
	}
}

func (m Model) tick() tea.Cmd {
	if m.interval <= 0 {
		return nil
	}
	return tea.Tick(m.interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) durationTimer() tea.Cmd {
	if m.duration <= 0 {
		return nil
	}
	return tea.Tick(m.duration, func(time.Time) tea.Msg {
		return durationDoneMsg{}
	})
}

func (m *Model) moveSelection(delta int) {
	count := len(m.lastSuccessfulList.Connections)
	if count == 0 {
		m.selected = 0
		m.viewportStart = 0
		return
	}
	m.selected = clampInt(m.selected+delta, 0, count-1)
	m.clampViewport()
}

func (m *Model) clampViewport() {
	count := len(m.lastSuccessfulList.Connections)
	if count == 0 {
		m.selected = 0
		m.viewportStart = 0
		return
	}
	visibleRows := visibleRowCount(count, bodyHeight(m.tableState()))
	m.selected = clampInt(m.selected, 0, count-1)
	m.viewportStart = viewportStartForSelection(m.viewportStart, m.selected, count, visibleRows)
}

func (m *Model) clampBlockerSelection() {
	if !m.detailOpen || m.detailTab != tabBlockers {
		return
	}
	subject, ok := m.detailSubject()
	if !ok {
		m.blockerRow = 0
		return
	}
	rows := detailBlockerRows(m.lastSuccessfulList, subject)
	if len(rows) == 0 {
		m.blockerRow = 0
		return
	}
	m.blockerRow = clampInt(m.blockerRow, 0, len(rows)-1)
}

func (m *Model) clampQueryOffset() {
	if !m.detailOpen || m.detailTab != tabQuery {
		m.queryOffset = 0
		return
	}
	subject, ok := m.detailSubject()
	if !ok {
		m.queryOffset = 0
		return
	}
	bodyHeight := queryBodyHeight(m.height, m.detailFooterLineCount())
	// The Query tab renders the full \G record (fields + query), so the scroll
	// clamp must span the whole record, not just the query lines — otherwise the
	// tail of the query is unreachable.
	total := len(connectionRecordLines(subject, m.displayPreset, tableWidth(m.width)))
	m.queryOffset = clampInt(m.queryOffset, 0, maxQueryOffset(total, bodyHeight))
}

func queryBodyHeight(height, footerLines int) int {
	if height <= 0 {
		height = 24
	}
	headerLines := 4
	bodyHeight := height - headerLines - footerLines
	if bodyHeight < 0 {
		return 0
	}
	return bodyHeight
}

// detailState assembles the detail-view snapshot. renderDetail and
// detailFooterLineCount both derive from this single builder so the body-height
// math they each perform reads an identical footer and can never diverge.
func (m Model) detailState() detailState {
	subject, ok := m.detailSubject()
	pos, total := m.stepPosition()
	return detailState{
		List:             m.lastSuccessfulList,
		Subject:          subject,
		SubjectFound:     ok,
		Tab:              m.detailTab,
		BlockerSelection: m.blockerRow,
		QueryOffset:      m.queryOffset,
		Width:            m.width,
		Height:           m.height,
		Paused:           m.paused,
		Refresh:          computeRefreshDot(m.loading, m.consecutiveErrors, m.isReplay()),
		ReadOnlyActions:  m.readOnlyActionError != "",
		Replay:           m.isReplay(),
		LastError:        firstNonEmpty(m.actionError, m.lastError),
		Notice:           m.notice.text,
		CaptureStopped:   m.captureStopped,
		CaptureStatus:    m.captureStatusText(),
		Confirm:          m.confirmPrompt(),
		Now:              m.now(),
		Interval:         m.interval,
		StepPos:          pos,
		StepTotal:        total,
		Target:           m.target,
		DisplayPreset:    m.displayPreset,
		Capabilities:     m.capabilities,
	}
}

func (m Model) detailFooterLineCount() int {
	return strings.Count(renderDetailFooter(m.detailState()), "\n") + 1
}

func (m Model) canChangeSort() bool {
	return len(m.sortModes) > 1
}

func nextSort(sort live.SortMode, options []live.SortMode) (live.SortMode, bool) {
	if len(options) <= 1 {
		return sort, false
	}
	for i, option := range options {
		if option == sort {
			return options[(i+1)%len(options)], true
		}
	}
	return options[0], true
}

func (m Model) selectedConnection() (live.Connection, bool) {
	if !m.hasList || len(m.lastSuccessfulList.Connections) == 0 {
		return live.Connection{}, false
	}
	if m.selected < 0 || m.selected >= len(m.lastSuccessfulList.Connections) {
		return live.Connection{}, false
	}
	return m.lastSuccessfulList.Connections[m.selected], true
}

// reanchorSelection keeps the highlight on the same connection (by PID+instance)
// after a refresh instead of leaving the positional index pointing at whatever
// connection now occupies that row. Vitess processlist rows can recycle into
// Sleep quickly, so a selected work row that becomes idle falls back to nearby
// active work instead of following the stale identity into the idle pool.
func (m *Model) reanchorSelection(previous live.Connection, previousIndex int) {
	if m.displayPreset == connectionDisplayProcesslist {
		m.reanchorProcesslistSelection(previous, previousIndex)
		return
	}
	for i, conn := range m.lastSuccessfulList.Connections {
		if sameConnection(conn, previous) {
			m.selected = i
			return
		}
	}
}

func (m *Model) reanchorProcesslistSelection(previous live.Connection, previousIndex int) {
	for i, conn := range m.lastSuccessfulList.Connections {
		if !sameConnection(conn, previous) {
			continue
		}
		if processlistConnectionHasWork(conn) || !processlistConnectionHasWork(previous) {
			m.selected = i
			return
		}
		break
	}
	m.selected = nearestProcesslistWorkIndex(m.lastSuccessfulList.Connections, previousIndex)
}

func sameConnection(a, b live.Connection) bool {
	return a.PID == b.PID && a.Instance == b.Instance
}

func nearestProcesslistWorkIndex(connections []live.Connection, index int) int {
	if len(connections) == 0 {
		return 0
	}
	index = clampInt(index, 0, len(connections)-1)
	if processlistConnectionHasWork(connections[index]) {
		return index
	}
	for offset := 1; offset < len(connections); offset++ {
		before := index - offset
		if before >= 0 && processlistConnectionHasWork(connections[before]) {
			return before
		}
		after := index + offset
		if after < len(connections) && processlistConnectionHasWork(connections[after]) {
			return after
		}
	}
	return index
}

func (m Model) detailSubject() (live.Connection, bool) {
	if m.detailPID == 0 {
		return live.Connection{}, false
	}
	for _, conn := range m.lastSuccessfulList.Connections {
		if conn.PID == m.detailPID && conn.Instance == m.detailInstance {
			return conn, true
		}
	}
	return live.Connection{}, false
}

// actionTargetConnection returns the connection that c/k/K should act on
// while the detail view is active. On the Blockers tab it resolves the
// highlighted blocker row; otherwise it returns the detail subject.
func (m Model) actionTargetConnection() (live.Connection, bool) {
	subject, ok := m.detailSubject()
	if !ok {
		return live.Connection{}, false
	}
	if m.detailTab != tabBlockers {
		return subject, true
	}
	rows := detailBlockerRows(m.lastSuccessfulList, subject)
	if m.blockerRow < 0 || m.blockerRow >= len(rows) {
		return subject, true
	}
	row := rows[m.blockerRow]
	if !row.Present {
		return live.Connection{}, false
	}
	return row.Connection, true
}

func (m Model) fireAction(kind actionKind, target live.ActionTarget) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch kind {
		case actionCancelQuery:
			err = m.client.CancelQuery(m.ctx, target)
		case actionTerminateTxn:
			err = m.client.TerminateTransaction(m.ctx, target)
		case actionTerminateConn:
			err = m.client.TerminateConnection(m.ctx, target)
		}
		return actionResultMsg{kind: kind, err: err}
	}
}

func actionNotice(kind actionKind) string {
	switch kind {
	case actionCancelQuery:
		return "cancel query"
	case actionTerminateTxn:
		return "terminate transaction"
	case actionTerminateConn:
		return "force terminate"
	}
	return "action"
}
