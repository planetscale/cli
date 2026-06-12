package tui

import "github.com/charmbracelet/lipgloss"

const (
	colorHeaderBlue              = "39"
	colorHeaderBlueLight         = "25"
	colorSelectedForeground      = "236"
	colorSelectedForegroundLight = "236"
	colorSelectedBackground      = "253"
	colorSelectedBackgroundLight = "254"
	colorErrorRed                = "196"
	colorErrorRedLight           = "160"
	colorMutedGray               = "245"
	colorMutedGrayLight          = "240"
	colorStaleYellow             = "214"
	colorStaleYellowLight        = "130"
	colorActiveGreen             = "40"
	colorActiveGreenLight        = "28"
	colorIdleGray                = "245"
	colorIdleGrayLight           = "240"
	colorXactYellow              = "214"
	colorXactYellowLight         = "130"
	colorReplicaBlue             = "117" // pastel blue; readable beside muted blocking-tier rows
	colorReplicaBlueLight        = "25"
	colorPausedBackground        = "229"
	colorPausedBackgroundLight   = "230"
	colorPausedForeground        = "236"
	colorPausedForegroundLight   = "236"
	colorBlocksOneRowBackground  = "254" // subtle hint
	colorBlocksOneLight          = "250" // subtle but visible on a white terminal
	colorBlocksTwoRowBackground  = "253" // noticeable step up
	colorBlocksTwoLight          = "248" // darker than the one-session light tier so the step-up reads on white
	colorBlocksManyRowBackground = "222" // amber highlight distinct from gray tiers without dominating the screen during real incidents
	colorBlocksManyLight         = "179"
	colorBlocksText              = "236"
	colorBlocksTextLight         = "236"
	colorBlockingCountBackground = "63"
	colorBlockingCountLight      = "230"
	colorRefreshActive           = "39" // cyan dot: a fetch is in flight or recently retried
	colorRefreshActiveLight      = "25"
	colorRefreshIdle             = "240" // dim dot when idle — fixed width, no header reflow
	colorRefreshIdleLight        = "250"
	colorRefreshError            = "196" // red dot: refresh is continuously failing (sustained 503s/timeouts)
	colorRefreshErrorLight       = "160"
	colorHelpKey                 = "245"
	colorHelpKeyLight            = "240"
	colorHelpDesc                = "245"
	colorHelpDescLight           = "242"
	colorHelpSeparator           = "238"
	colorHelpSeparatorLight      = "247"
)

var (
	headerStyle      = lipgloss.NewStyle().Bold(true).Foreground(adaptiveColor(colorHeaderBlueLight, colorHeaderBlue))
	errorStyle       = lipgloss.NewStyle().Foreground(adaptiveColor(colorErrorRedLight, colorErrorRed))
	mutedStyle       = lipgloss.NewStyle().Foreground(adaptiveColor(colorMutedGrayLight, colorMutedGray))
	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(adaptiveColor(colorSelectedForegroundLight, colorSelectedForeground)).
				Background(adaptiveColor(colorSelectedBackgroundLight, colorSelectedBackground))
	replicaRowStyle  = lipgloss.NewStyle().Foreground(adaptiveColor(colorReplicaBlueLight, colorReplicaBlue))
	staleStyle       = lipgloss.NewStyle().Foreground(adaptiveColor(colorStaleYellowLight, colorStaleYellow))
	veryStaleStyle   = lipgloss.NewStyle().Foreground(adaptiveColor(colorErrorRedLight, colorErrorRed)).Bold(true)
	stateActiveStyle = lipgloss.NewStyle().Foreground(adaptiveColor(colorActiveGreenLight, colorActiveGreen)).Bold(true)
	stateIdleStyle   = lipgloss.NewStyle().Foreground(adaptiveColor(colorIdleGrayLight, colorIdleGray))
	stateXactStyle   = lipgloss.NewStyle().Foreground(adaptiveColor(colorXactYellowLight, colorXactYellow)).Bold(true)
	bannerStyle      = lipgloss.NewStyle().Foreground(adaptiveColor(colorErrorRedLight, colorErrorRed)).Bold(true)
	tabActiveStyle   = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(adaptiveColor(colorHeaderBlueLight, colorHeaderBlue))
	pausedStyle      = lipgloss.NewStyle().
				Background(adaptiveColor(colorPausedBackgroundLight, colorPausedBackground)).
				Foreground(adaptiveColor(colorPausedForegroundLight, colorPausedForeground)).
				Bold(true)

	rowBlocksOneSessionStyle = lipgloss.NewStyle().
					Foreground(adaptiveColor(colorBlocksTextLight, colorBlocksText)).
					Background(adaptiveColor(colorBlocksOneLight, colorBlocksOneRowBackground))
	rowBlocksTwoSessionsStyle = lipgloss.NewStyle().
					Foreground(adaptiveColor(colorBlocksTextLight, colorBlocksText)).
					Background(adaptiveColor(colorBlocksTwoLight, colorBlocksTwoRowBackground)).
					Bold(true)
	rowBlocksManySessionsStyle = lipgloss.NewStyle().
					Foreground(adaptiveColor(colorBlocksTextLight, colorBlocksText)).
					Background(adaptiveColor(colorBlocksManyLight, colorBlocksManyRowBackground)).
					Bold(true)
	// The blocking-count is a bold amber foreground digit (an alarm-family hue),
	// not a filled background badge. A filled cell read as a terminal
	// text-selection rectangle and the violet hue didn't map to "alarm"
	// Amber-on-default keeps it legible on both terminal backgrounds.
	blockingCountBadgeStyle = lipgloss.NewStyle().
				Foreground(adaptiveColor(colorStaleYellowLight, colorStaleYellow)).
				Bold(true)
	refreshActiveStyle = lipgloss.NewStyle().Foreground(adaptiveColor(colorRefreshActiveLight, colorRefreshActive)).Bold(true)
	refreshIdleStyle   = lipgloss.NewStyle().Foreground(adaptiveColor(colorRefreshIdleLight, colorRefreshIdle))
	refreshErrorStyle  = lipgloss.NewStyle().Foreground(adaptiveColor(colorRefreshErrorLight, colorRefreshError)).Bold(true)
	helpKeyStyle       = lipgloss.NewStyle().Foreground(adaptiveColor(colorHelpKeyLight, colorHelpKey))
	helpDescStyle      = lipgloss.NewStyle().Foreground(adaptiveColor(colorHelpDescLight, colorHelpDesc))
	helpSeparatorStyle = lipgloss.NewStyle().Foreground(adaptiveColor(colorHelpSeparatorLight, colorHelpSeparator))
)

func adaptiveColor(light, dark string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: light, Dark: dark}
}
