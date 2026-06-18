package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	lgtree "github.com/charmbracelet/lipgloss/tree"
	"github.com/charmbracelet/x/ansi"
	live "github.com/planetscale/cli/internal/connections"
)

type detailTab string

const (
	tabQuery    detailTab = "query"
	tabBlockers detailTab = "blockers"

	detailLabelWidth = 16
)

type detailState struct {
	List             live.ConnectionList
	Subject          live.Connection
	SubjectFound     bool
	Tab              detailTab
	BlockerSelection int
	QueryOffset      int
	Width            int
	Height           int
	Paused           bool
	Refresh          refreshDotState
	ReadOnlyActions  bool
	Replay           bool
	LastError        string
	Notice           string
	CaptureStopped   string // sticky reason when capture writer detaches on error; "" when capture is healthy or absent
	CaptureStatus    string
	Confirm          string
	Now              time.Time
	Interval         time.Duration
	StepPos          int // 1-based position when stepping; 0 means following live
	StepTotal        int // total samples held in history
	Target           Target
	DisplayPreset    connectionDisplayPreset
	Capabilities     ConnectionCapabilities
}

func renderDetail(state detailState) string {
	state = normalizeDetailState(state)
	width := tableWidth(state.Width)
	height := state.Height
	if height <= 0 {
		height = 24
	}

	headerLines := []string{
		clipLine(renderDetailHeader(state), width),
		"",
		clipLine(renderDetailTabs(state.Tab, state.Capabilities), width),
		"",
	}

	footerLines := strings.Split(renderDetailFooter(state), "\n")
	bodyHeight := height - len(headerLines) - len(footerLines)
	if bodyHeight < 0 {
		bodyHeight = 0
	}
	var bodyLines []string
	if !state.SubjectFound {
		bodyLines = append(bodyLines,
			clipLine(mutedStyle.Render("This connection is no longer in the live snapshot — it closed or was terminated."), width),
			clipLine(mutedStyle.Render("Press q or esc to return to the connection list."), width),
		)
	} else {
		switch state.Tab {
		case tabBlockers:
			bodyLines = append(bodyLines, renderBlockers(state, width, bodyHeight)...)
		default:
			bodyLines = append(bodyLines, renderQuery(state.Subject, state.DisplayPreset, width, bodyHeight, state.QueryOffset)...)
		}
	}

	if len(bodyLines) > bodyHeight {
		bodyLines = bodyLines[:bodyHeight]
	}
	lines := append(headerLines, bodyLines...)
	for len(lines)+len(footerLines) < height {
		lines = append(lines, "")
	}
	lines = append(lines, footerLines...)
	return strings.Join(lines, "\n")
}

func normalizeDetailState(state detailState) detailState {
	state.Capabilities = state.Capabilities.effective()
	state.Tab = effectiveDetailTab(state.Tab, state.Capabilities)
	return state
}

func renderDetailHeader(state detailState) string {
	if !state.SubjectFound {
		if target := renderTarget(state.Target); target != "" {
			return target + " | " + mutedStyle.Render("connection ended")
		}
		return "live connection detail | " + mutedStyle.Render("connection ended")
	}
	// The header carries only identity + operational state. The per-connection
	// metadata (user, app, state, duration, wait, IDs) lives in the body record
	// below, so the header stays short instead of overflowing and clipping.
	parts := []string{}
	if target := renderTarget(state.Target); target != "" {
		parts = append(parts, target)
	}
	parts = append(parts, refreshIndicator(state.Refresh)+" "+fmt.Sprintf("pid %d", state.Subject.PID))
	if state.Subject.Instance != "" {
		parts = append(parts, "on "+state.Subject.Instance)
	}
	if state.Paused {
		parts = append(parts, pausedStyle.Render("paused"))
	}
	if state.StepPos > 0 {
		parts = append(parts, fmt.Sprintf("step %d/%d", state.StepPos, state.StepTotal))
	}
	if state.CaptureStopped != "" {
		parts = append(parts, errorStyle.Render(state.CaptureStopped))
	}
	if state.CaptureStatus != "" {
		parts = append(parts, state.CaptureStatus)
	}
	if token := capturedToken(state.List.CapturedAt, state.Now, state.Interval); token != "" {
		parts = append(parts, token)
	}
	return strings.Join(parts, " | ")
}

func renderDetailTabs(active detailTab, capabilities ConnectionCapabilities) string {
	if !capabilities.ShowBlockers {
		return tabActiveStyle.Render("[query]")
	}
	blockers, query := "blockers", "query"
	if active == tabBlockers {
		blockers = tabActiveStyle.Render("[blockers]")
	} else {
		query = tabActiveStyle.Render("[query]")
	}
	return blockers + "  " + query
}

func effectiveDetailTab(active detailTab, capabilities ConnectionCapabilities) detailTab {
	if active == tabBlockers && !capabilities.ShowBlockers {
		return tabQuery
	}
	return active
}

// connectionRecordLines renders the full per-connection detail as a MySQL
// `\G`-style vertical record: one aligned field per line (from the shared
// live.Connection.HumanFields), then the wrapped query text. This fills the
// detail body — which otherwise showed only the query over a sea of whitespace
// — and reuses the exact field set the agent-cli `list --format human` emits.
func connectionRecordLines(conn live.Connection, preset connectionDisplayPreset, width int) []string {
	return renderConnectionRecord(connectionDetailFields(preset, conn), conn.QueryText, width)
}

func renderConnectionRecord(fields [][2]string, query string, width int) []string {
	var lines []string
	for _, field := range fields {
		label := mutedStyle.Render(fmt.Sprintf("%-*s", detailLabelWidth, field[0]+":"))
		values := detailFieldValueLines(field[1], width-detailLabelWidth-1)
		for i, value := range values {
			if i == 0 {
				lines = append(lines, clipLine(label+" "+value, width))
				continue
			}
			lines = append(lines, clipLine(strings.Repeat(" ", detailLabelWidth+1)+value, width))
		}
	}
	lines = append(lines, headerStyle.Render("query:"))
	queryLines := queryDisplayLines(query, width)
	if len(queryLines) == 0 {
		queryLines = []string{mutedStyle.Render("no query")}
	}
	lines = append(lines, queryLines...)
	return lines
}

func detailFieldValueLines(value string, width int) []string {
	rendered := detailFieldValue(value)
	if strings.TrimSpace(value) == "" || width <= 0 || ansi.StringWidth(rendered) <= width {
		return []string{rendered}
	}
	return wrapLines(value, width)
}

func detailFieldValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return mutedStyle.Render("none")
	}
	return value
}

func connectionDetailFields(preset connectionDisplayPreset, conn live.Connection) [][2]string {
	if preset != connectionDisplayProcesslist {
		return conn.HumanFields()
	}

	return [][2]string{
		{"pid", fmt.Sprint(conn.PID)},
		{"tablet", conn.Instance},
		{"state", conn.State},
		{"duration", processlistDetailDuration(conn)},
		{"user", conn.Username},
		{"database", conn.DatabaseName},
		{"client_addr", conn.ClientAddr},
		{"connection_id", live.DerefString(conn.ConnectionID)},
		{"query_id", live.DerefString(conn.QueryID)},
	}
}

func processlistDetailDuration(conn live.Connection) string {
	if conn.Duration <= 0 && !processlistConnectionHasWork(conn) {
		return ""
	}
	return conn.Duration.String()
}

func renderQuery(conn live.Connection, preset connectionDisplayPreset, width, height, offset int) []string {
	if height <= 0 {
		return nil
	}
	wrapped := connectionRecordLines(conn, preset, width)
	if len(wrapped) == 0 {
		return []string{mutedStyle.Render("no query")}
	}
	if len(wrapped) > height-1 {
		offset = clampInt(offset, 0, maxQueryOffset(len(wrapped), height))
		end := offset + height - 1
		visible := append([]string{}, wrapped[offset:end]...)
		endLine := end
		if endLine < offset+1 {
			endLine = offset + 1
		}
		visible = append(visible, mutedStyle.Render(fmt.Sprintf("lines %d-%d/%d", offset+1, endLine, len(wrapped))))
		return visible
	}
	return wrapped
}

func renderBlockers(state detailState, width, height int) []string {
	lines := []string{headerStyle.Render("BLOCKED BY")}
	selection := 0
	selectedLine := -1

	blockedBy := blockerRows(state.List, state.Subject)
	if len(blockedBy) == 0 {
		lines = append(lines, mutedStyle.Render("No upstream blocker"))
	} else {
		if state.BlockerSelection < len(blockedBy) {
			selectedLine = len(lines) + state.BlockerSelection
		}
		rendered := renderBlockerTreeRows(blockedBy, state.BlockerSelection, selection, width)
		lines = append(lines, rendered...)
		selection += len(blockedBy)
	}

	// Two blank lines separate the upstream and downstream trees so the
	// BLOCKING header reads as a section break, not just another tree row,
	// even when the upstream tree has dozens of entries.
	lines = append(lines, "", "", headerStyle.Render("BLOCKING"))
	blocking := blockingRows(state.List, state.Subject)
	if len(blocking) == 0 {
		lines = append(lines, mutedStyle.Render("Not blocking other connections"))
	} else {
		if state.BlockerSelection >= selection {
			selectedLine = len(lines) + (state.BlockerSelection - selection)
		}
		lines = append(lines, renderBlockerTreeRows(blocking, state.BlockerSelection, selection, width)...)
	}
	return visibleDetailBody(lines, selectedLine, height)
}

func visibleDetailBody(lines []string, selectedLine, height int) []string {
	if height <= 0 || len(lines) <= height {
		return lines
	}
	if selectedLine < 0 {
		return lines[:height]
	}
	start := centeredViewportStart(selectedLine, len(lines), height)
	return lines[start : start+height]
}

func renderDetailFooter(state detailState) string {
	var lines []string
	if state.Confirm != "" {
		lines = append(lines, errorStyle.Render(clipLine(state.Confirm, tableWidth(state.Width))))
	} else if state.LastError != "" {
		lines = append(lines, errorStyle.Render(clipLine("error: "+state.LastError, tableWidth(state.Width))))
	} else if state.Notice != "" {
		lines = append(lines, clipLine("status: "+state.Notice, tableWidth(state.Width)))
	}
	if status := renderDetailSelectedStatus(state); status != "" {
		lines = append(lines, status)
	}
	lines = append(lines, renderDetailHelp(state))
	return strings.Join(lines, "\n")
}

func renderDetailSelectedStatus(state detailState) string {
	if !state.SubjectFound {
		return ""
	}
	width := tableWidth(state.Width)
	if state.Tab == tabBlockers {
		rows := detailBlockerRows(state.List, state.Subject)
		if idx := state.BlockerSelection; idx >= 0 && idx < len(rows) {
			row := rows[idx]
			if row.Present {
				return renderConnectionStatus("→ selected blocker", row.Connection, width)
			}
			return clipLine(fmt.Sprintf("→ selected blocker pid %d | %s", row.PID, mutedStyle.Render("connection ended")), width)
		}
	}
	return renderConnectionStatus("→ selected connection", state.Subject, width)
}

func renderConnectionStatus(label string, conn live.Connection, width int) string {
	status := fmt.Sprintf("%s pid %d", label, conn.PID)
	query := sanitizeFooterText(strings.Join(strings.Fields(conn.QueryText), " "))
	if query != "" {
		status += " | " + query
	}
	return clipLine(status, width)
}

func renderDetailHelp(state detailState) string {
	labels := actionHelpLabels(state.DisplayPreset)
	cancelHelp := labels.cancel
	killTxnHelp := "kill transaction"
	terminateHelp := labels.terminateConn
	support := state.Capabilities.effective()

	bindings := []key.Binding{}
	if support.ShowBlockers {
		bindings = append(bindings, key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("left/right", "tabs")))
	}
	if state.Tab == tabBlockers {
		bindings = append(bindings, key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open selected")))
	}
	if !state.Replay && !state.Paused {
		bindings = append(bindings, key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")))
	}
	pauseLabel := "pause"
	if state.Paused {
		pauseLabel = "resume"
	}
	bindings = append(bindings, key.NewBinding(key.WithKeys("space"), key.WithHelp("space", pauseLabel)))
	if !state.Replay {
		bindings = append(bindings, key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "capture")))
	}
	bindings = append(bindings, key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q/esc", "back")))
	// An ended connection has nothing to act on, so suppress the destructive
	// action hints rather than advertise keys that no-op.
	if state.SubjectFound && !state.ReadOnlyActions && !state.Replay {
		if support.supports(actionCancelQuery) {
			bindings = append(bindings, key.NewBinding(key.WithKeys("c"), key.WithHelp("c", cancelHelp)))
		}
		if support.supports(actionTerminateTxn) {
			bindings = append(bindings, key.NewBinding(key.WithKeys("k"), key.WithHelp("k", killTxnHelp)))
		}
		if support.supports(actionTerminateConn) {
			bindings = append(bindings, key.NewBinding(key.WithKeys("K"), key.WithHelp("shift+K", terminateHelp)))
		}
	}
	bindings = append(bindings,
		key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	)
	return renderWrappedHelp(bindings, " | ", tableWidth(state.Width))
}

func renderBlockerTreeRows(rows []blockerRow, selected, selectionOffset, width int) []string {
	if isSingleLevelBlockerTree(rows) {
		lines := make([]string, 0, len(rows))
		for i, row := range rows {
			lines = append(lines, clipLine("• "+blockerTreeLabel(row, selectionOffset+i == selected), width))
		}
		return lines
	}

	tree := lgtree.New()
	stack := []*lgtree.Tree{tree}
	for i, row := range rows {
		depth := row.Depth
		if depth < 0 {
			depth = 0
		}
		if depth >= len(stack) {
			depth = len(stack) - 1
		}
		node := lgtree.Root(blockerTreeLabel(row, selectionOffset+i == selected))
		stack[depth].Child(node)
		stack = append(stack[:depth+1], node)
	}
	rendered := tree.String()
	if rendered == "" {
		return nil
	}
	lines := strings.Split(rendered, "\n")
	for i, line := range lines {
		lines[i] = clipLine(line, width)
	}
	return lines
}

func isSingleLevelBlockerTree(rows []blockerRow) bool {
	if len(rows) == 0 {
		return false
	}
	for _, row := range rows {
		if row.Depth > 0 {
			return false
		}
	}
	return true
}

func blockerTreeLabel(row blockerRow, selected bool) string {
	label := blockerLabel(row)
	if selected {
		// Use the same ▶ cursor as the table so the selection affordance is
		// consistent across views.
		return selectedRowStyle.Render("▶ " + label)
	}
	// Reserve the caret's width on unselected rows so moving the selection
	// doesn't shift the row text, matching the table view's fixed-width marker.
	return "  " + label
}

func wrapLines(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	runes := []rune(text)
	var lines []string
	for len(runes) > width {
		lines = append(lines, string(runes[:width]))
		runes = runes[width:]
	}
	if len(runes) > 0 {
		lines = append(lines, string(runes))
	}
	return lines
}
