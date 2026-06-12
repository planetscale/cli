package tui

import (
	"math"
	"strconv"
	"testing"

	"github.com/charmbracelet/lipgloss"
	qt "github.com/frankban/quicktest"
	"github.com/muesli/termenv"
)

func TestStylesAdaptToLightBackgrounds(t *testing.T) {
	prevProfile := lipgloss.ColorProfile()
	prevBackground := lipgloss.HasDarkBackground()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(prevProfile)
	defer lipgloss.SetHasDarkBackground(prevBackground)

	tests := []struct {
		name  string
		style lipgloss.Style
	}{
		{name: "header", style: headerStyle},
		{name: "muted", style: mutedStyle},
		{name: "replica row", style: replicaRowStyle},
		{name: "stale", style: staleStyle},
		{name: "active state", style: stateActiveStyle},
		{name: "idle state", style: stateIdleStyle},
		{name: "transaction state", style: stateXactStyle},
		{name: "paused badge", style: pausedStyle},
		{name: "one blocker row", style: rowBlocksOneSessionStyle},
		{name: "two blocker row", style: rowBlocksTwoSessionsStyle},
		{name: "many blocker row", style: rowBlocksManySessionsStyle},
		{name: "blocking count badge", style: blockingCountBadgeStyle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			lipgloss.SetHasDarkBackground(true)
			dark := tt.style.Render("sample")
			lipgloss.SetHasDarkBackground(false)
			light := tt.style.Render("sample")

			c.Assert(stripANSI(light), qt.Equals, "sample")
			c.Assert(light, qt.Not(qt.Equals), dark)
		})
	}
}

func TestSelectedRowHasBackgroundOnBothTerminalBackgrounds(t *testing.T) {
	c := qt.New(t)
	prevProfile := lipgloss.ColorProfile()
	prevBackground := lipgloss.HasDarkBackground()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(prevProfile)
	defer lipgloss.SetHasDarkBackground(prevBackground)

	lipgloss.SetHasDarkBackground(true)
	dark := selectedRowStyle.Render("sample")
	lipgloss.SetHasDarkBackground(false)
	light := selectedRowStyle.Render("sample")

	c.Assert(dark, qt.Contains, "48;5;")
	c.Assert(dark, qt.Contains, "38;5;")
	c.Assert(stripANSI(dark), qt.Equals, "sample")
	c.Assert(light, qt.Contains, "48;5;")
	c.Assert(light, qt.Contains, "38;5;")
	c.Assert(stripANSI(light), qt.Equals, "sample")
}

func TestDarkVariantHighlightsRemainStyledForLightFallback(t *testing.T) {
	c := qt.New(t)
	prevProfile := lipgloss.ColorProfile()
	prevBackground := lipgloss.HasDarkBackground()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(prevProfile)
	defer lipgloss.SetHasDarkBackground(prevBackground)

	lipgloss.SetHasDarkBackground(true)

	for _, tt := range []struct {
		name     string
		rendered string
		plain    string
	}{
		{name: "paused", rendered: pausedStyle.Render("paused"), plain: "paused"},
		{name: "blocking", rendered: rowBlocksTwoSessionsStyle.Render("sample"), plain: "sample"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c.Assert(tt.rendered, qt.Contains, "38;5;")
			c.Assert(tt.rendered, qt.Contains, "48;5;")
			c.Assert(stripANSI(tt.rendered), qt.Equals, tt.plain)
		})
	}
}

func TestBlockingRowsSetForegroundForLightFallback(t *testing.T) {
	c := qt.New(t)
	prevProfile := lipgloss.ColorProfile()
	prevBackground := lipgloss.HasDarkBackground()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(prevProfile)
	defer lipgloss.SetHasDarkBackground(prevBackground)

	lipgloss.SetHasDarkBackground(true)

	c.Assert(rowBlocksOneSessionStyle.Render("sample"), qt.Contains, "38;5;")
	c.Assert(rowBlocksTwoSessionsStyle.Render("sample"), qt.Contains, "38;5;")
	c.Assert(rowBlocksManySessionsStyle.Render("sample"), qt.Contains, "38;5;")
}

func TestLightBlockingBackgroundsContrastWithWhiteSurface(t *testing.T) {
	for _, tt := range []struct {
		name     string
		color    string
		contrast float64
	}{
		{name: "one blocker", color: colorBlocksOneLight, contrast: 1.8},
		{name: "two blockers", color: colorBlocksTwoLight, contrast: 2.1},
		{name: "many blockers", color: colorBlocksManyLight, contrast: 2.0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			c.Assert(contrastAgainstWhite(tt.color) >= tt.contrast, qt.IsTrue)
		})
	}
}

func TestOneSessionBlockerVisibleOnLightBackground(t *testing.T) {
	c := qt.New(t)
	prevProfile := lipgloss.ColorProfile()
	prevBackground := lipgloss.HasDarkBackground()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(prevProfile)
	defer lipgloss.SetHasDarkBackground(prevBackground)

	lipgloss.SetHasDarkBackground(false)

	one := rowBlocksOneSessionStyle.Render("sample")
	two := rowBlocksTwoSessionsStyle.Render("sample")

	c.Assert(stripANSI(one), qt.Equals, "sample")
	c.Assert(one, qt.Contains, "48;5;")
	c.Assert(one, qt.Not(qt.Equals), two)
}

func contrastAgainstWhite(code string) float64 {
	r, g, b := ansi256RGB(code)
	l := relativeLuminance(r, g, b)
	return 1.05 / (l + 0.05)
}

func ansi256RGB(code string) (float64, float64, float64) {
	n, err := strconv.Atoi(code)
	if err != nil {
		panic(err)
	}
	if n >= 232 {
		gray := float64(8 + (n-232)*10)
		return gray, gray, gray
	}
	if n >= 16 {
		levels := []float64{0, 95, 135, 175, 215, 255}
		n -= 16
		return levels[n/36], levels[(n/6)%6], levels[n%6]
	}
	basic := [16][3]float64{
		{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0},
		{0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192},
		{128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
		{0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
	}
	return basic[n][0], basic[n][1], basic[n][2]
}

func relativeLuminance(r, g, b float64) float64 {
	return 0.2126*linearizedColor(r) + 0.7152*linearizedColor(g) + 0.0722*linearizedColor(b)
}

func linearizedColor(v float64) float64 {
	v /= 255
	if v <= 0.03928 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}
