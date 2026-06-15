package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	live "github.com/planetscale/cli/internal/connections"
)

type tableState struct {
	List            live.ConnectionList
	HasList         bool
	Sort            live.SortMode
	CanSort         bool
	Selected        int
	ViewportStart   int
	Width           int
	Height          int
	Paused          bool
	Refresh         refreshDotState
	ReadOnlyActions bool
	Replay          bool
	LastError       string
	AccessDenied    bool
	CanStepHistory  bool
	Notice          string
	CaptureStopped  string // sticky reason when capture writer detaches on error; "" when capture is healthy or absent
	CaptureStatus   string
	Confirm         string
	Now             time.Time
	Interval        time.Duration
	StepPos         int // 1-based position when stepping; 0 means following live
	StepTotal       int // total samples held in history
	Target          Target
	Filter          string // active row filter chip, e.g. "filter: role=primary"; empty when none
	DisplayPreset   connectionDisplayPreset
	Capabilities    ConnectionCapabilities
}

type freshnessTier int

const (
	freshnessFresh freshnessTier = iota
	freshnessStale
	freshnessVeryStale
)

func freshnessTierFor(captured, now time.Time, interval time.Duration) freshnessTier {
	if interval <= 0 || captured.IsZero() {
		return freshnessFresh
	}
	age := now.Sub(captured)
	if age >= 10*interval {
		return freshnessVeryStale
	}
	if age >= 3*interval {
		return freshnessStale
	}
	return freshnessFresh
}

func instanceRoleMarker(role string) string {
	switch role {
	case "primary":
		return ""
	case "replica":
		return "R"
	default:
		return ""
	}
}

const (
	defaultTableWidth         = 120
	queryPreviewLimit         = 160
	viewportEdgeRowThreshold  = 2
	startColumnMinWidth       = 140
	processlistTabletMinWidth = 120
	// Width budget for the APP column on a 120-col terminal.
	appColumnTarget = 14
	// Minimum rendered width of the BLOCK column. Matches the "BLOCK" header so
	// the column never narrows below the header when downstream counts are
	// single-digit (Y, N, Y 3 are otherwise 1–3 chars wide and would let
	// lipgloss squeeze the column past the header).
	blockColumnMinWidth = 5
	// Stable minimum widths for the STATE and WAIT columns. lipgloss sizes a
	// column to its widest cell, so when a wide value (e.g. "idle/xact",
	// "Lock/transactionid") appears or disappears between refreshes the column —
	// and everything to its right — jitters horizontally. Padding each cell to a
	// fixed minimum that fits the common values pins the column width so the live
	// view stays stable across refreshes. Widths cover "idle/xact" (9)
	// and "Lock/transactionid" (18); the rare wider values still expand.
	stateColumnMinWidth = 9
	waitColumnMinWidth  = 18
)

const (
	connectionColState = 2
	connectionColBlock = 3
)

type bindings struct {
	Navigate      key.Binding
	Refresh       key.Binding
	Pause         key.Binding
	Capture       key.Binding
	Sort          key.Binding
	Step          key.Binding
	Detail        key.Binding
	Cancel        key.Binding
	TerminateTxn  key.Binding
	TerminateConn key.Binding
	Help          key.Binding
	Quit          key.Binding
}

type actionLabels struct {
	cancel        string
	terminateConn string
}

func actionHelpLabels(display connectionDisplayPreset) actionLabels {
	if display == connectionDisplayProcesslist {
		return actionLabels{cancel: "KILL QUERY", terminateConn: "KILL"}
	}
	return actionLabels{cancel: "cancel query", terminateConn: "force terminate"}
}

func (b bindings) ShortHelp() []key.Binding {
	return b.ShortHelpFor(false, false, connectionDisplayDefault, DefaultConnectionCapabilities(), true, true, true, true, false)
}

func (b bindings) ShortHelpFor(readOnlyActions, replay bool, display connectionDisplayPreset, support ConnectionCapabilities, canSort, hasList, canSelectRow, canStepHistory, paused bool) []key.Binding {
	b = b.withActionLabels(display)
	b = b.withPauseLabel(paused)
	if !hasList {
		help := []key.Binding{}
		if !replay && !paused {
			help = append(help, b.Refresh)
		}
		help = append(help, b.Pause)
		return append(help, b.Help, b.Quit)
	}

	help := []key.Binding{}
	if canSelectRow {
		help = append(help, b.Navigate)
	}
	if !replay && !paused {
		help = append(help, b.Refresh)
	}
	help = append(help, b.Pause)
	if !replay {
		help = append(help, b.Capture)
	}
	if canSort {
		help = append(help, b.Sort)
	}
	if canStepHistory {
		help = append(help, b.Step)
	}
	if canSelectRow {
		help = append(help, b.Detail)
	}
	if canSelectRow && !readOnlyActions {
		support = support.effective()
		if support.supports(actionCancelQuery) {
			help = append(help, b.Cancel)
		}
		if support.supports(actionTerminateTxn) {
			help = append(help, b.TerminateTxn)
		}
		if support.supports(actionTerminateConn) {
			help = append(help, b.TerminateConn)
		}
	}
	return append(help, b.Help, b.Quit)
}

func (b bindings) withActionLabels(display connectionDisplayPreset) bindings {
	labels := actionHelpLabels(display)
	b.Cancel = key.NewBinding(key.WithKeys("c"), key.WithHelp("c", labels.cancel))
	b.TerminateConn = key.NewBinding(key.WithKeys("K"), key.WithHelp("shift+K", labels.terminateConn))
	return b
}

func (b bindings) withPauseLabel(paused bool) bindings {
	label := "pause"
	if paused {
		label = "resume"
	}
	b.Pause = key.NewBinding(key.WithKeys("space"), key.WithHelp("space", label))
	return b
}

// renderTable returns the complete terminal frame for the table view. Sizes
// the connection-body section so total output never exceeds state.Height —
// otherwise an overflowing alt-screen frame pushes the header off the top.
func renderTable(state tableState) string {
	headerLines := []string{renderHeader(state), ""}
	var bannerLines []string
	if banner := renderInstanceFailureBanner(state.List.Instances); banner != "" {
		bannerLines = []string{banner, ""}
	}
	footerLines := strings.Split(renderFooter(state), "\n")
	bodyAvail := state.Height - len(headerLines) - len(bannerLines) - len(footerLines)

	var bodyLines []string
	switch {
	case !state.HasList && state.AccessDenied:
		bodyLines = []string{
			errorStyle.Render("you don't have permission to view live connections"),
			mutedStyle.Render("production branches require the Analyst role or higher — ask an org admin, or use a development branch"),
			"",
		}
	case !state.HasList && state.LastError != "":
		bodyLines = []string{
			errorStyle.Render("unable to load live connections"),
			errorStyle.Render(clipLine(sanitizeFooterText(state.LastError), tableWidth(state.Width))),
			"",
		}
	case !state.HasList:
		bodyLines = []string{mutedStyle.Render("loading live connections..."), ""}
	case len(state.List.Connections) == 0:
		bodyLines = []string{
			mutedStyle.Render("no live connections"),
			mutedStyle.Render("new connections appear on the next refresh"),
			"",
		}
	default:
		bodyLines = strings.Split(renderConnectionTable(state, bodyAvail), "\n")
	}

	if state.Height > 0 {
		for len(bodyLines) < bodyAvail {
			bodyLines = append(bodyLines, "")
		}
		if bodyAvail > 0 && len(bodyLines) > bodyAvail {
			bodyLines = bodyLines[:bodyAvail]
		} else if bodyAvail <= 0 {
			bodyLines = nil
		}
	}

	all := make([]string, 0, len(headerLines)+len(bannerLines)+len(bodyLines)+len(footerLines))
	all = append(all, headerLines...)
	all = append(all, bannerLines...)
	all = append(all, bodyLines...)
	all = append(all, footerLines...)
	if state.Height > 0 && len(all) > state.Height {
		all = all[:state.Height]
	}
	return strings.Join(all, "\n")
}

// headerPart is one ` | `-delimited header token in the fitted summary line.
type headerPart struct {
	text string
}

type headerOptions struct {
	showSort     bool
	showRecOff   bool
	compactCount bool
	targetMode   headerTargetMode
	compactFresh bool
}

type headerTargetMode int

const (
	headerTargetFull headerTargetMode = iota
	headerTargetCompactShard
	headerTargetEllipsizeWithShard
	headerTargetCompactFresh
	headerTargetNoShard
)

const minHeaderDatabaseWidth = 12

// renderHeader returns the summary line at the top of the table, fitted to the
// viewport width. The staged fallbacks compact the target, count, sort, and
// capture tokens before protected freshness, pause, and count context is lost.
func renderHeader(state tableState) string {
	width := tableWidth(state.Width)
	stages := []headerOptions{
		{showSort: true, showRecOff: true, targetMode: headerTargetFull},
		{showSort: true, targetMode: headerTargetFull},
		{targetMode: headerTargetFull},
		{compactCount: true, targetMode: headerTargetFull},
		{compactCount: true, targetMode: headerTargetCompactShard},
		{compactCount: true, targetMode: headerTargetEllipsizeWithShard},
		{compactCount: true, targetMode: headerTargetCompactFresh, compactFresh: true},
		{compactCount: true, targetMode: headerTargetNoShard, compactFresh: true},
	}
	if state.CaptureStatus != "rec off" {
		for i := range stages {
			stages[i].showRecOff = true
		}
	}
	for _, opts := range stages {
		line := renderHeaderWithOptions(state, width, opts)
		if width <= 0 || ansi.StringWidth(line) <= width {
			return clipLine(line, width)
		}
	}
	return clipLine(renderHeaderWithOptions(state, width, stages[len(stages)-1]), width)
}

func renderHeaderWithOptions(state tableState, width int, opts headerOptions) string {
	var parts []headerPart
	if target := renderHeaderTarget(displayTarget(state.Target, state.List), width, opts.targetMode, state, opts); target != "" {
		parts = append(parts, headerPart{text: target})
	}
	parts = append(parts,
		headerPart{text: connectionCountHeaderText(state, opts.compactCount)},
	)
	if state.Filter != "" {
		parts = append(parts, headerPart{text: headerFilterText(state.Filter, width)})
	}
	if opts.showSort {
		parts = append(parts, headerPart{text: sortHeaderText(state)})
	}
	if state.Paused {
		parts = append(parts, headerPart{text: pausedStyle.Render("paused")})
	}
	if state.StepPos > 0 {
		parts = append(parts, headerPart{text: fmt.Sprintf("step %d/%d", state.StepPos, state.StepTotal)})
	}
	if state.CaptureStopped != "" {
		parts = append(parts, headerPart{text: errorStyle.Render(state.CaptureStopped)})
	}
	if state.CaptureStatus != "" && (state.CaptureStatus != "rec off" || opts.showRecOff) {
		parts = append(parts, headerPart{text: state.CaptureStatus})
	}
	if token := renderCapturedHeaderToken(state, opts.compactFresh); token != "" {
		parts = append(parts, headerPart{text: token})
	}
	return joinHeaderParts(parts)
}

func sortHeaderText(state tableState) string {
	if !state.CanSort {
		return fmt.Sprintf("sorted by %s", state.Sort)
	}
	return fmt.Sprintf("sort %s", state.Sort)
}

func connectionCountText(state tableState) string {
	if !state.HasList {
		return "connections —"
	}
	return fmt.Sprintf("connections %d", len(state.List.Connections))
}

func connectionCountHeaderText(state tableState, compact bool) string {
	if !compact {
		return refreshIndicator(state.Refresh) + " " + connectionCountText(state)
	}
	if !state.HasList {
		return refreshIndicator(state.Refresh) + " —"
	}
	return fmt.Sprintf("%s %d", refreshIndicator(state.Refresh), len(state.List.Connections))
}

// refreshDotState is the health of the live refresh, surfaced by the header dot.
type refreshDotState int

const (
	refreshDotIdle    refreshDotState = iota // dim: nothing pending, last refresh succeeded
	refreshDotPending                        // cyan: a fetch is in flight, or one recently failed but not for long
	refreshDotFailing                        // red: refresh is continuously failing (sustained 503s/timeouts)
	refreshDotHidden                         // replay: there is no live refresh, so show no dot
)

// refreshDotFailThreshold is the count of consecutive failed list fetches after
// which the indicator escalates from "recently retried" (cyan) to "continuously
// failing" (red). One failure still reads as a transient blip; two in a row
// reads as a real, ongoing outage.
const refreshDotFailThreshold = 2

// computeRefreshDot maps the live refresh state to a dot state. It depends only
// on the live fetch/error bookkeeping — never on the paused/stepped cursor — so
// the dot keeps reflecting the most recent refresh attempt while the operator
// holds or steps through history.
func computeRefreshDot(loading bool, consecutiveErrors int, replay bool) refreshDotState {
	if replay {
		return refreshDotHidden
	}
	if consecutiveErrors >= refreshDotFailThreshold {
		return refreshDotFailing
	}
	if loading || consecutiveErrors > 0 {
		return refreshDotPending
	}
	return refreshDotIdle
}

// refreshIndicator returns a fixed-width status dot: cyan while a fetch is in
// flight or recently retried, red when the refresh is continuously failing, dim
// when idle, and a blank (no dot) in replay mode. The glyph is always one cell
// wide, so toggling it never reflows the header.
func refreshIndicator(state refreshDotState) string {
	switch state {
	case refreshDotHidden:
		return " "
	case refreshDotFailing:
		return refreshErrorStyle.Render("●")
	case refreshDotPending:
		return refreshActiveStyle.Render("●")
	default:
		return refreshIdleStyle.Render("●")
	}
}

func headerFilterText(filter string, width int) string {
	maxWidth := width / 4
	if maxWidth < 12 {
		maxWidth = min(width, 12)
	}
	if maxWidth > 32 {
		maxWidth = 32
	}
	return clipLine(filter, maxWidth)
}

func joinHeaderParts(parts []headerPart) string {
	texts := make([]string, 0, len(parts))
	for _, p := range parts {
		texts = append(texts, p.text)
	}
	return strings.Join(texts, " | ")
}

func renderTarget(target Target) string {
	var parts []string
	if target.Database != "" {
		parts = append(parts, target.Database)
	}
	if target.Branch != "" {
		parts = append(parts, target.Branch)
	}
	if target.Keyspace != "" {
		parts = append(parts, target.Keyspace)
	}
	if target.Shard != "" {
		parts = append(parts, target.Shard)
	}
	return strings.Join(parts, " / ")
}

func renderHeaderTarget(target Target, width int, mode headerTargetMode, state tableState, opts headerOptions) string {
	if mode == headerTargetFull {
		return renderTarget(target)
	}
	rendered := renderHeaderTargetText(target, mode, 0)
	if mode != headerTargetEllipsizeWithShard && mode != headerTargetCompactFresh && mode != headerTargetNoShard {
		return rendered
	}
	budget := headerTargetBudget(rendered, state, opts, width)
	if budget <= 0 {
		return ""
	}
	return renderHeaderTargetText(target, mode, budget)
}

func renderHeaderTargetText(target Target, mode headerTargetMode, budget int) string {
	keyspaceShard := compactKeyspaceShard(target)
	includeKeyspaceShard := keyspaceShard != "" && mode != headerTargetNoShard
	database := target.Database
	if budget > 0 {
		suffix := targetSuffixText(target.Branch, keyspaceShard, includeKeyspaceShard)
		databaseBudget := budget
		if suffix != "" && database != "" {
			databaseBudget -= ansi.StringWidth(" / " + suffix)
		}
		if includeKeyspaceShard && database != "" && databaseBudget < minHeaderDatabaseWidth {
			return renderHeaderTargetText(target, headerTargetCompactShard, 0)
		}
		if databaseBudget <= 0 {
			if !includeKeyspaceShard && mode == headerTargetEllipsizeWithShard {
				return renderTarget(target)
			}
			database = ""
			if ansi.StringWidth(suffix) > budget {
				return ""
			}
		} else {
			database = clipLine(database, databaseBudget)
		}
	}
	parts := []string{}
	if database != "" {
		parts = append(parts, database)
	}
	if target.Branch != "" {
		parts = append(parts, target.Branch)
	}
	if includeKeyspaceShard {
		parts = append(parts, keyspaceShard)
	}
	return strings.Join(parts, " / ")
}

func compactKeyspaceShard(target Target) string {
	switch {
	case target.Keyspace != "" && target.Shard != "":
		return target.Keyspace + "/" + target.Shard
	case target.Keyspace != "":
		return target.Keyspace
	default:
		return target.Shard
	}
}

func targetSuffixText(branch, keyspaceShard string, includeKeyspaceShard bool) string {
	var parts []string
	if branch != "" {
		parts = append(parts, branch)
	}
	if includeKeyspaceShard && keyspaceShard != "" {
		parts = append(parts, keyspaceShard)
	}
	return strings.Join(parts, " / ")
}

func headerTargetBudget(target string, state tableState, opts headerOptions, width int) int {
	if width <= 0 || target == "" {
		return width
	}
	placeholder := "\x00"
	line := renderHeaderWithTarget(state, opts, placeholder)
	withoutTarget := strings.Replace(line, placeholder, "", 1)
	return min(ansi.StringWidth(target), width-ansi.StringWidth(withoutTarget))
}

func renderHeaderWithTarget(state tableState, opts headerOptions, target string) string {
	var parts []headerPart
	if target != "" {
		parts = append(parts, headerPart{text: target})
	}
	parts = append(parts, headerPart{text: connectionCountHeaderText(state, opts.compactCount)})
	if state.Filter != "" {
		parts = append(parts, headerPart{text: headerFilterText(state.Filter, tableWidth(state.Width))})
	}
	if opts.showSort {
		parts = append(parts, headerPart{text: sortHeaderText(state)})
	}
	if state.Paused {
		parts = append(parts, headerPart{text: pausedStyle.Render("paused")})
	}
	if state.StepPos > 0 {
		parts = append(parts, headerPart{text: fmt.Sprintf("step %d/%d", state.StepPos, state.StepTotal)})
	}
	if state.CaptureStopped != "" {
		parts = append(parts, headerPart{text: errorStyle.Render(state.CaptureStopped)})
	}
	if state.CaptureStatus != "" && (state.CaptureStatus != "rec off" || opts.showRecOff) {
		parts = append(parts, headerPart{text: state.CaptureStatus})
	}
	if token := renderCapturedHeaderToken(state, opts.compactFresh); token != "" {
		parts = append(parts, headerPart{text: token})
	}
	return joinHeaderParts(parts)
}

func displayTarget(target Target, list live.ConnectionList) Target {
	if list.Topology == nil {
		return target
	}
	if target.Keyspace == "" {
		target.Keyspace = list.Topology.Keyspace
	}
	if target.Shard == "" {
		target.Shard = list.Topology.Shard
	}
	return target
}

func renderCapturedToken(state tableState) string {
	return capturedToken(state.List.CapturedAt, state.Now, state.Interval)
}

func renderCapturedHeaderToken(state tableState, compact bool) string {
	if compact {
		return capturedCompactToken(state.List.CapturedAt, state.Now, state.Interval)
	}
	return renderCapturedToken(state)
}

func capturedCompactToken(captured, now time.Time, interval time.Duration) string {
	if captured.IsZero() {
		return ""
	}
	if now.IsZero() {
		return "captured " + formatCapturedAbsolute(captured, now)
	}
	age := now.Sub(captured).Truncate(time.Second)
	if age < 0 {
		age = 0
	}
	relative := fmt.Sprintf("(%ds)", int(age.Seconds()))
	switch freshnessTierFor(captured, now, interval) {
	case freshnessVeryStale:
		relative = veryStaleStyle.Render(relative)
	case freshnessStale:
		relative = staleStyle.Render(relative)
	}
	return relative
}

// capturedToken renders the absolute capture time plus a relative-age suffix
// that tints as the sample goes stale. Age is shown in every mode, including
// paused; the separate "paused" chip in the header communicates the freeze.
func capturedToken(captured, now time.Time, interval time.Duration) string {
	if captured.IsZero() {
		return ""
	}
	// Absolute timestamp stays unstyled so operators have a stable wall-clock
	// reference; only the relative-age suffix tints to signal staleness.
	absolute := "captured " + formatCapturedAbsolute(captured, now)
	if now.IsZero() {
		return absolute
	}
	age := now.Sub(captured).Truncate(time.Second)
	if age < 0 {
		age = 0
	}
	relative := fmt.Sprintf("(%ds ago)", int(age.Seconds()))
	switch freshnessTierFor(captured, now, interval) {
	case freshnessVeryStale:
		relative = veryStaleStyle.Render(relative)
	case freshnessStale:
		relative = staleStyle.Render(relative)
	}
	return absolute + " " + relative
}

// formatCapturedAbsolute renders the absolute timestamp shown in the header.
// Same calendar day as `now`: time-only. Different day: include the date. The
// timezone is always the operator's local — operators looking at this header
// are debugging on their own clock.
func formatCapturedAbsolute(captured, now time.Time) string {
	captured = captured.Local()
	if !now.IsZero() && sameLocalDay(captured, now.Local()) {
		return captured.Format("15:04:05")
	}
	return captured.Format("2006-01-02 15:04:05")
}

func sameLocalDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func renderInstanceFailureBanner(instances []live.InstanceMeta) string {
	if len(instances) == 0 {
		return ""
	}
	var failed []string
	for _, inst := range instances {
		if inst.Error != "" {
			failed = append(failed, inst.ID)
		}
	}
	if len(failed) == 0 {
		return ""
	}
	return bannerStyle.Render(fmt.Sprintf(
		"%d of %d instances unreachable: %s",
		len(failed), len(instances), strings.Join(failed, ", "),
	))
}

// renderFooter returns the command/status line at the bottom of the table.
func renderFooter(state tableState) string {
	lines := []string{}
	if selectedStatus := renderSelectedStatus(state); selectedStatus != "" {
		lines = append(lines, selectedStatus)
	}

	switch {
	case state.Confirm != "":
		// Confirmation prompt replaces the help line: appending would push the
		// prompt past the terminal width on standard layouts, and only y/n/esc/
		// q/ctrl+c are valid keys while confirming anyway.
		lines = append(lines, errorStyle.Render(clipLine(state.Confirm, tableWidth(state.Width))))
	default:
		if state.LastError != "" && state.HasList {
			lines = append(lines, errorStyle.Render(clipLine("error: "+sanitizeFooterText(state.LastError), tableWidth(state.Width))))
		} else if state.Notice != "" {
			lines = append(lines, clipLine("status: "+state.Notice, tableWidth(state.Width)))
		}
		lines = append(lines, renderHelpFor(state.Width, state.ReadOnlyActions, state.Replay, state.DisplayPreset, state.Capabilities, state.CanSort, state.HasList, state.HasList && len(state.List.Connections) > 0, state.CanStepHistory, state.Paused))
	}
	return strings.Join(lines, "\n")
}

func renderSelectedStatus(state tableState) string {
	if !state.HasList || state.Selected < 0 || state.Selected >= len(state.List.Connections) {
		return ""
	}

	conn := state.List.Connections[state.Selected]
	status := fmt.Sprintf("selected pid %d", conn.PID)
	query := strings.Join(strings.Fields(conn.QueryText), " ")
	if query != "" {
		status += " | " + query
	}
	return clipLine(status, tableWidth(state.Width))
}

// queryPreview returns the collapsed query text shown in the final table column.
func queryPreview(query string) string {
	collapsed := strings.Join(strings.Fields(query), " ")
	return truncateRunes(collapsed, queryPreviewLimit)
}

// renderConnectionTable returns the visible slice of connection rows. bodyAvail
// is the terminal row count reserved for the table body (header row + rows).
func renderConnectionTable(state tableState, bodyAvail int) string {
	width := tableWidth(state.Width)
	connections := state.List.Connections
	visibleRows := visibleRowCount(len(connections), bodyAvail)
	start := viewportStartForSelection(state.ViewportStart, state.Selected, len(connections), visibleRows)
	visible := visibleConnections(connections, start, visibleRows)
	selectedInSlice := state.Selected - start
	counts := live.BlockingCounts(connections)
	headers, rows := buildConnectionRowsForDisplay(state.DisplayPreset, visible, counts, width, selectedInSlice)
	return renderConnectionTableTight(state.DisplayPreset, headers, rows, visible, counts, selectedInSlice, width)
}

// renderConnectionTableTight lays the table out with content-based column
// widths and a fixed inter-column gap, then clips to the terminal width. Unlike
// a width-filling table layout, this keeps columns packed at the left when the
// content (notably the trailing QUERY column) is short, so uniform/short rows
// don't get spread across the whole terminal.
func renderConnectionTableTight(display connectionDisplayPreset, headers []string, rows [][]string, connections []live.Connection, counts map[int]int, selectedInSlice, width int) string {
	if len(headers) == 0 {
		return ""
	}
	columnWidths := tightColumnWidths(headers, rows)
	lines := []string{renderTightRow(display, headers, headers, nil, 0, headerStyle, columnWidths, width)}
	for i, row := range rows {
		var conn *live.Connection
		blockCount := 0
		base := lipgloss.NewStyle()
		if i >= 0 && i < len(connections) {
			c := connections[i]
			conn = &c
			blockCount = counts[c.PID]
			base = connectionRowStyle(c, blockCount, i == selectedInSlice)
		}
		lines = append(lines, renderTightRow(display, row, headers, conn, blockCount, base, columnWidths, width))
	}
	return strings.Join(lines, "\n")
}

func tightColumnWidths(headers []string, rows [][]string) []int {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = ansi.StringWidth(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && i < len(row)-1 {
				widths[i] = max(widths[i], ansi.StringWidth(cell))
			}
		}
	}
	return widths
}

func renderTightRow(display connectionDisplayPreset, cells, headers []string, conn *live.Connection, blockCount int, base lipgloss.Style, widths []int, width int) string {
	var line strings.Builder
	for i, cell := range cells {
		style := base
		if conn != nil {
			style = connectionColumnStyleForDisplay(display, base, *conn, blockCount, i, headers)
		}
		if i < len(cells)-1 && i < len(widths) {
			cell = padCellToWidth(cell, widths[i])
			cell += tightCellPadding(i)
		}
		line.WriteString(style.Render(cell))
	}
	return clipLine(line.String(), width)
}

func padCellToWidth(text string, width int) string {
	if pad := width - ansi.StringWidth(text); pad > 0 {
		text += strings.Repeat(" ", pad)
	}
	return text
}

func tightCellPadding(index int) string {
	if index == 0 {
		return " "
	}
	return "  "
}

func connectionRowStyle(conn live.Connection, blockCount int, selected bool) lipgloss.Style {
	style := blockingRowStyle(blockCount)
	if conn.InstanceRole == "replica" {
		style = style.Inherit(replicaRowStyle)
	}
	if selected {
		return style.Inherit(selectedRowStyle)
	}
	return style
}

func connectionColumnStyleForDisplay(display connectionDisplayPreset, style lipgloss.Style, conn live.Connection, blockCount int, col int, headers []string) lipgloss.Style {
	if display == connectionDisplayProcesslist {
		if col >= 0 && col < len(headers) && headers[col] == "STATE" {
			return processlistStateStyleFor(style, conn.State)
		}
		return style
	}
	switch col {
	case connectionColState:
		return stateStyleFor(style, conn.State)
	case connectionColBlock:
		if blockCount > 0 {
			return style.Foreground(adaptiveColor(colorStaleYellowLight, colorStaleYellow)).Bold(true)
		}
		if len(conn.BlockedBy) == 0 {
			return style.Inherit(mutedStyle)
		}
	}
	return style
}

func stateStyleFor(style lipgloss.Style, state string) lipgloss.Style {
	switch stateText(state) {
	case "active":
		return style.Foreground(adaptiveColor(colorActiveGreenLight, colorActiveGreen)).Bold(true)
	case "idle":
		return style.Foreground(adaptiveColor(colorIdleGrayLight, colorIdleGray))
	case "idle/xact", "idle/xact (aborted)":
		return style.Foreground(adaptiveColor(colorXactYellowLight, colorXactYellow)).Bold(true)
	default:
		return style
	}
}

// buildConnectionRows returns the table headers and cell values for visible
// connections.
func buildConnectionRows(connections []live.Connection, counts map[int]int, width int, selectedInSlice int) ([]string, [][]string) {
	return buildConnectionRowsForDisplay(connectionDisplayDefault, connections, counts, width, selectedInSlice)
}

func buildConnectionRowsForDisplay(display connectionDisplayPreset, connections []live.Connection, counts map[int]int, width int, selectedInSlice int) ([]string, [][]string) {
	if display == connectionDisplayProcesslist {
		return buildProcesslistConnectionRows(connections, width, selectedInSlice)
	}

	includeStart := width >= startColumnMinWidth
	headers := []string{"", "PID", "STATE", "BLOCK", "WAIT", "DURATION", "APP"}
	if includeStart {
		headers = append(headers, "START")
	}
	headers = append(headers, "QUERY")

	rows := make([][]string, 0, len(connections))
	for i, conn := range connections {
		marker := rowMarker(conn.InstanceRole, i == selectedInSlice)
		row := []string{
			marker,
			fmt.Sprint(conn.PID),
			stateCell(conn.State),
			blockedText(conn, counts[conn.PID]),
			waitTextForWidth(conn, width),
			formatDuration(conn.Duration),
			appTextForWidth(conn.ApplicationName, width),
		}
		if includeStart {
			row = append(row, formatTime(startTime(conn)))
		}
		row = append(row, queryPreview(conn.QueryText))
		rows = append(rows, row)
	}
	return headers, rows
}

func buildProcesslistConnectionRows(connections []live.Connection, width int, selectedInSlice int) ([]string, [][]string) {
	includeTablet := width >= processlistTabletMinWidth
	headers := []string{"", "PID"}
	if includeTablet {
		headers = append(headers, "TABLET")
	}
	headers = append(headers, "STATE", "DURATION", "USER", "DB", "QUERY")
	rows := make([][]string, 0, len(connections))
	for i, conn := range connections {
		row := []string{
			processlistRowMarker(i == selectedInSlice),
			fmt.Sprint(conn.PID),
		}
		if includeTablet {
			row = append(row, emptyDash(conn.Instance))
		}
		row = append(row,
			emptyDash(processlistStateText(conn.State)),
			processlistDuration(conn),
			appText(conn.Username),
			emptyDash(conn.DatabaseName),
			queryPreview(conn.QueryText),
		)
		rows = append(rows, row)
	}
	return headers, rows
}

func rowMarker(role string, selected bool) string {
	cursor := " "
	if selected {
		cursor = "▶"
	}
	roleMarker := instanceRoleMarker(role)
	if roleMarker == "" {
		roleMarker = " "
	}
	return cursor + roleMarker
}

func processlistRowMarker(selected bool) string {
	if selected {
		return "▶"
	}
	return " "
}

// stateText returns the abbreviated render of a connection's Postgres state.
// "idle in transaction" is the operator-critical state (holds locks), so the
// abbreviation must remain distinguishable from plain "idle".
func stateText(state string) string {
	switch state {
	case "idle in transaction":
		return "idle/xact"
	case "idle in transaction (aborted)":
		return "idle/xact (aborted)"
	default:
		return state
	}
}

func stateCell(state string) string {
	text := emptyDash(stateText(state))
	return padCell(text, stateColumnMinWidth)
}

func processlistStateText(state string) string {
	command, _, ok := strings.Cut(state, "/")
	if ok {
		return command
	}
	return state
}

func processlistDuration(conn live.Connection) string {
	if conn.Duration <= 0 && processlistConnectionHasWork(conn) {
		return "00:00"
	}
	return formatDuration(conn.Duration)
}

func processlistConnectionHasWork(conn live.Connection) bool {
	state := strings.ToLower(strings.TrimSpace(processlistStateText(conn.State)))
	if state == "sleep" || state == "idle" {
		return false
	}
	return state != "" || strings.TrimSpace(conn.QueryText) != ""
}

func processlistStateStyleFor(style lipgloss.Style, state string) lipgloss.Style {
	switch strings.ToLower(processlistStateText(state)) {
	case "query":
		return style.Foreground(adaptiveColor(colorActiveGreenLight, colorActiveGreen)).Bold(true)
	case "sleep":
		return style.Foreground(adaptiveColor(colorIdleGrayLight, colorIdleGray))
	default:
		return style
	}
}

// padCell right-pads a cell to a minimum display width, pinning the column so it
// doesn't jitter as values change between refreshes. lipgloss sizes columns to
// the widest cell, so a constant trailing pad is an effective minimum.
func padCell(text string, minWidth int) string {
	if pad := minWidth - lipgloss.Width(text); pad > 0 {
		text += strings.Repeat(" ", pad)
	}
	return text
}

// waitText returns the combined wait-cause shown in the WAIT column. When the
// type and event are both populated, render "type/event" so operators see the
// discriminating suffix (e.g. "Lock/transactionid"). When only one half is
// populated, fall back to that half.
func waitText(conn live.Connection) string {
	switch {
	case conn.WaitEventType != "" && conn.WaitEvent != "":
		return conn.WaitEventType + "/" + conn.WaitEvent
	case conn.WaitEvent != "":
		return conn.WaitEvent
	case conn.WaitEventType != "":
		return conn.WaitEventType
	default:
		return "-"
	}
}

func waitTextForWidth(conn live.Connection, width int) string {
	wait := waitText(conn)
	if width < 100 {
		return clipLine(wait, 10)
	}
	return padCell(wait, waitColumnMinWidth)
}

func appText(name string) string {
	return appTextForWidth(name, 0)
}

func appTextForWidth(name string, width int) string {
	if name == "" {
		return "-"
	}
	limit := appColumnTarget
	if width >= 220 {
		limit = 24
	}
	runes := []rune(name)
	if len(runes) <= limit {
		return name
	}
	keep := limit - 1
	if keep > 0 && runes[keep-1] == '_' {
		keep++
	}
	return string(runes[:keep]) + "…"
}

// blockedText renders the BLOCK column. The leading character carries the alarm
// signal: a digit if this connection is blocking N downstream sessions (the
// operationally critical case — root blockers show "3" not "N 3"), "W" if it's
// waiting on a lock with no downstream, "-" if quiet. A trailing " W" suffix
// indicates the connection is both blocking and waiting (chain victim).
// Right-padded so the rendered column never narrows below the BLOCK header
// width — lipgloss sizes columns to max cell width, so trailing whitespace
// here effectively sets a minimum.
//
//	3 W — blocking 3 downstream AND waiting on a lock
//	3   — blocking 3 downstream (root blocker)
//	W   — waiting on a lock, blocking nothing
//	-   — quiet
func blockedText(conn live.Connection, count int) string {
	waiting := len(conn.BlockedBy) > 0
	var text string
	switch {
	case count > 0 && waiting:
		text = fmt.Sprintf("%d W", count)
	case count > 0:
		text = fmt.Sprint(count)
	case waiting:
		text = "W"
	default:
		text = "-"
	}
	return padCell(text, blockColumnMinWidth)
}

// blockingRowStyle shades rows that block downstream connections.
func blockingRowStyle(depth int) lipgloss.Style {
	switch {
	case depth >= 3:
		return rowBlocksManySessionsStyle
	case depth == 2:
		return rowBlocksTwoSessionsStyle
	case depth == 1:
		return rowBlocksOneSessionStyle
	default:
		return lipgloss.NewStyle()
	}
}

// formatDuration returns the compact duration text shown in the DURATION column.
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	d = d.Truncate(time.Second)
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	if minutes >= 60 {
		hours := minutes / 60
		minutes = minutes % 60
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// startTime returns the transaction start or query start shown in the START column.
func startTime(conn live.Connection) *time.Time {
	if conn.XactStart != nil {
		return conn.XactStart
	}
	return conn.QueryStart
}

func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("15:04:05")
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

// visibleRowCount returns how many connection rows fit in bodyAvail terminal
// rows after reserving one row for the table header. Floors at 2 so navigation
// keeps some context even in tiny terminals.
func visibleRowCount(totalRows, bodyAvail int) int {
	rows := bodyAvail - 1
	if rows < 2 {
		rows = 2
	}
	if totalRows > 0 && rows > totalRows {
		rows = totalRows
	}
	return rows
}

// bodyHeight returns the terminal rows available for the connection-body
// section (table header + rows). Mirrors the chrome accounting in renderTable
// so the model can clamp viewport scrolling to the same visible-row count the
// renderer uses.
func bodyHeight(state tableState) int {
	headerLines := 2
	bannerLines := 0
	if renderInstanceFailureBanner(state.List.Instances) != "" {
		bannerLines = 2
	}
	footerLines := strings.Count(renderFooter(state), "\n") + 1
	return state.Height - headerLines - bannerLines - footerLines
}

func visibleConnections(connections []live.Connection, start, height int) []live.Connection {
	start = clampViewportStart(start, len(connections), height)
	end := start + height
	if end > len(connections) {
		end = len(connections)
	}
	return connections[start:end]
}

func viewportStartForSelection(currentStart, selected, totalRows, visibleRows int) int {
	currentStart = clampViewportStart(currentStart, totalRows, visibleRows)
	if totalRows <= visibleRows {
		return 0
	}
	selected = clampInt(selected, 0, totalRows-1)
	row := selected - currentStart
	if selected < currentStart || selected >= currentStart+visibleRows {
		return centeredViewportStart(selected, totalRows, visibleRows)
	}
	if row < viewportEdgeRowThreshold && currentStart > 0 {
		return centeredViewportStart(selected, totalRows, visibleRows)
	}
	if row >= visibleRows-viewportEdgeRowThreshold && currentStart < totalRows-visibleRows {
		return centeredViewportStart(selected, totalRows, visibleRows)
	}
	return currentStart
}

func centeredViewportStart(selected, totalRows, visibleRows int) int {
	return clampViewportStart(selected-(visibleRows/2), totalRows, visibleRows)
}

func clampViewportStart(start, totalRows, visibleRows int) int {
	if totalRows <= 0 || visibleRows <= 0 || totalRows <= visibleRows {
		return 0
	}
	return clampInt(start, 0, totalRows-visibleRows)
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func tableWidth(width int) int {
	if width <= 0 {
		return defaultTableWidth
	}
	return width
}

func renderHelp(width int) string {
	return renderHelpFor(width, false, false, connectionDisplayDefault, DefaultConnectionCapabilities(), true, true, true, true, false)
}

func renderHelpFor(width int, readOnlyActions, replay bool, display connectionDisplayPreset, support ConnectionCapabilities, canSort, hasList, canSelectRow, canStepHistory, paused bool) string {
	return renderWrappedHelp(defaultBindings().ShortHelpFor(readOnlyActions, replay, display, support, canSort, hasList, canSelectRow, canStepHistory, paused), " | ", tableWidth(width))
}

// renderWrappedHelp lays out key hints across as many lines as the width
// requires, wrapping rather than truncating so no hint vanishes off the right
// edge at narrow widths.
func renderWrappedHelp(bindings []key.Binding, separator string, width int) string {
	helpModel := help.New()
	helpModel.Width = width
	helpModel.ShortSeparator = separator
	helpModel.Styles.ShortKey = helpKeyStyle
	helpModel.Styles.ShortDesc = helpDescStyle
	helpModel.Styles.ShortSeparator = helpSeparatorStyle
	rows := wrapHelpBindings(helpModel, bindings, width)
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, helpModel.ShortHelpView(row))
	}
	return strings.Join(lines, "\n")
}

func wrapHelpBindings(helpModel help.Model, bindings []key.Binding, width int) [][]key.Binding {
	rows := [][]key.Binding{}
	row := []key.Binding{}

	for _, binding := range bindings {
		if !binding.Enabled() {
			continue
		}

		candidate := make([]key.Binding, 0, len(row)+1)
		candidate = append(candidate, row...)
		candidate = append(candidate, binding)
		if len(row) > 0 && lipgloss.Width(renderUnwrappedHelp(helpModel, candidate)) > width {
			rows = append(rows, row)
			row = []key.Binding{binding}
			continue
		}
		row = candidate
	}

	if len(row) > 0 {
		rows = append(rows, row)
	}
	return rows
}

func renderUnwrappedHelp(helpModel help.Model, bindings []key.Binding) string {
	helpModel.Width = 0
	return helpModel.ShortHelpView(bindings)
}

func defaultBindings() bindings {
	return bindings{
		Navigate:      key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("up/down", "select")),
		Refresh:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Pause:         key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "pause")),
		Capture:       key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "capture")),
		Sort:          key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		Step:          key.NewBinding(key.WithKeys("[", "]", "{", "}"), key.WithHelp("[ ] { }", "step history")),
		Detail:        key.NewBinding(key.WithKeys("enter", "v"), key.WithHelp("enter/v", "detail")),
		Cancel:        key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "cancel query")),
		TerminateTxn:  key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "kill transaction")),
		TerminateConn: key.NewBinding(key.WithKeys("K"), key.WithHelp("shift+K", "force terminate")),
		Help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:          key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

// clipLine clips a rendered terminal line to the current viewport width.
// Width is measured in visible cells, so ANSI styling (e.g. a bold/underlined
// active tab) does not count against the budget or get truncated mid-escape.
func clipLine(line string, width int) string {
	if width <= 0 {
		return line
	}
	if ansi.StringWidth(line) <= width {
		return line
	}
	if width <= 1 {
		return ansi.Truncate(line, width, "")
	}
	return ansi.Truncate(line, width, "…")
}

// truncateRunes clips user-controlled text without splitting UTF-8 runes.
func truncateRunes(value string, limit int) string {
	if limit < 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func sanitizeFooterText(value string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
}
