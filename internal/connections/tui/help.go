package tui

import (
	"fmt"
	"strings"
)

type helpState struct {
	Target        Target
	Width         int
	Height        int
	Offset        int
	CanSort       bool
	Paused        bool
	Replay        bool
	DisplayPreset connectionDisplayPreset
	Capabilities  ConnectionCapabilities
}

func renderHelpModal(state helpState) string {
	width := tableWidth(state.Width)
	lines := clippedHelpLines(helpModalLines(state), width)
	if state.Height > 0 {
		if len(lines) > state.Height {
			return strings.Join(visibleHelpLines(lines, state.Height, state.Offset), "\n")
		}
		for len(lines) < state.Height {
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n")
}

// maxHelpOffset reports the largest useful scroll offset for the help modal;
// scrolling past it would not reveal any new line.
func maxHelpOffset(state helpState) int {
	if state.Height <= 1 {
		return 0
	}
	lines := helpModalLines(state)
	if len(lines) <= state.Height {
		return 0
	}
	return len(lines) - helpBodyHeight(state.Height)
}

func helpModalLines(state helpState) []string {
	support := state.Capabilities.effective()
	lines := []string{
		headerStyle.Render("Connections Help"),
		"",
		"Target: " + emptyDash(renderTarget(state.Target)),
		"",
		headerStyle.Render("Reading The Table"),
		"  Rows are live database sessions; the selected row is the action target.",
		"  Columns are abbreviated to fit the terminal; open detail for full query text.",
		"",
		headerStyle.Render("States And Status"),
		statusHelp(state.DisplayPreset),
		"  Paused keeps captured age visible; refreshing means fetching.",
		"",
	}
	if support.ShowBlockers {
		lines = append(lines,
			headerStyle.Render("Markers And Blocking"),
			"  R marks replica sessions. BLOCK digits show downstream sessions blocked.",
			"  W in BLOCK means the session is waiting on a lock.",
			"",
		)
	}
	if !support.supports(actionTerminateTxn) {
		lines = append(lines,
			headerStyle.Render("Actions"),
			"  c  Kill the selected query (KILL QUERY) using the query_id from the selected process.",
			"  K  Kill the selected connection (KILL) using the connection_id from the selected process.",
			"  c and K require confirmation. Replay mode blocks backend actions.",
			"",
		)
	} else {
		lines = append(lines,
			headerStyle.Render("Actions"),
			"  c  Cancel the selected query (pg_cancel_backend) only if it is the same active query we observed.",
			"  k  Kill the selected transaction (pg_terminate_backend) only if it is the same transaction we observed.",
			"  K  Force terminate the selected connection (pg_terminate_backend) only if backend start matches what we observed.",
			"  c, k, and K require confirmation. Replay mode blocks backend actions.",
			"",
		)
	}
	navigation := "  up/down select; enter/v detail"
	if support.ShowBlockers {
		navigation += "; left/right switch detail tabs; b blockers"
	}
	controls := []string{}
	if !state.Replay && !state.Paused {
		controls = append(controls, "r refresh")
	}
	if state.Paused {
		controls = append(controls, "space resume")
	} else {
		controls = append(controls, "space pause")
	}
	if !state.Replay {
		controls = append(controls, "C capture")
	}
	controls = append(controls, "[ ] { } step history")
	if state.CanSort {
		controls = append(controls, "s sort")
	}
	lines = append(lines,
		headerStyle.Render("Navigation"),
		navigation,
		"  "+strings.Join(controls, "; "),
		"  q/esc close or go back; ctrl+c quit; ?, esc, or q closes help",
	)
	return lines
}

func clippedHelpLines(lines []string, width int) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = clipLine(line, width)
	}
	return out
}

func visibleHelpLines(lines []string, height, offset int) []string {
	if height <= 1 {
		return lines[:1]
	}
	bodyHeight := height - 1
	offset = clampInt(offset, 0, len(lines)-bodyHeight)
	end := offset + bodyHeight
	visible := append([]string{}, lines[offset:end]...)
	visible = append(visible, mutedStyle.Render(fmt.Sprintf("lines %d-%d/%d", offset+1, end, len(lines))))
	return visible
}

func statusHelp(display connectionDisplayPreset) string {
	if display == connectionDisplayProcesslist {
		return "  Query is running; Sleep is quiet; other states come from the Vitess processlist."
	}
	return "  active is running; idle is quiet; idle/xact may hold locks."
}
