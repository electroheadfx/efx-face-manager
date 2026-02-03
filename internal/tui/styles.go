package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors - matching efx-face-manager theme
var (
	primary   = lipgloss.Color("#A855F7") // Light purple/violet (same as selection bg)
	secondary = lipgloss.Color("#10B981") // Green
	accent    = lipgloss.Color("#A855F7") // Same light purple as primary
	muted     = lipgloss.Color("#9CA3AF") // Lighter gray
	danger    = lipgloss.Color("#EF4444") // Red
	warning   = lipgloss.Color("#F59E0B") // Yellow
	white     = lipgloss.Color("#FFFFFF")
	dark      = lipgloss.Color("#1F2937")
)

// App frame
var appStyle = lipgloss.NewStyle().
	Padding(1, 2)

// Title - ASCII art header style
var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(primary).
	MarginBottom(1)

// ASCII Header box
var headerBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.DoubleBorder()).
	BorderForeground(primary).
	Padding(1, 2).
	Align(lipgloss.Center)

// Subtitle / section header
var subtitleStyle = lipgloss.NewStyle().
	Foreground(accent).
	Bold(true).
	MarginBottom(1)

// Section title
var sectionTitleStyle = lipgloss.NewStyle().
	Foreground(muted).
	MarginBottom(1)

// Menu item - normal
var menuItemStyle = lipgloss.NewStyle().
	PaddingLeft(2)

// Menu item - selected
var menuItemSelectedStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(white).
	Background(primary).
	PaddingLeft(2).
	PaddingRight(2)

// Status indicators
var statusOkStyle = lipgloss.NewStyle().
	Foreground(secondary)

var statusWarnStyle = lipgloss.NewStyle().
	Foreground(warning)

var statusErrorStyle = lipgloss.NewStyle().
	Foreground(danger)

var statusMutedStyle = lipgloss.NewStyle().
	Foreground(muted)

// Help bar
var helpStyle = lipgloss.NewStyle().
	Foreground(muted).
	MarginTop(1)

// Panel styles
var panelStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(muted).
	Padding(1, 1)

var panelFocusedStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(primary).
	Padding(1, 1)

var panelTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(accent).
	MarginBottom(1)

// Command preview box
var commandBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(primary).
	Padding(1, 2)

// Server status indicator
var serverRunningStyle = lipgloss.NewStyle().
	Foreground(secondary).
	Bold(true)

var serverStoppedStyle = lipgloss.NewStyle().
	Foreground(muted)

// Error message
var errorStyle = lipgloss.NewStyle().
	Foreground(danger).
	Bold(true)

// Success message
var successStyle = lipgloss.NewStyle().
	Foreground(secondary).
	Bold(true)

// Spinner style
var spinnerStyle = lipgloss.NewStyle().
	Foreground(primary)

// Info line style
var infoLineStyle = lipgloss.NewStyle().
	Foreground(muted).
	Italic(true)

// Option selected in list
var optionSelectedStyle = lipgloss.NewStyle().
	Foreground(primary).
	Bold(true)

// Option normal
var optionNormalStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#E5E7EB"))

// Value style (for displaying current values)
var valueStyle = lipgloss.NewStyle().
	Foreground(secondary)

// Disabled value
var valueDisabledStyle = lipgloss.NewStyle().
	Foreground(muted).
	Italic(true)

// Helper functions for dynamic panel sizing
func getPanelStyle(width, height int, focused bool) lipgloss.Style {
	style := panelStyle
	if focused {
		style = panelFocusedStyle
	}
	return style.Width(width).Height(height)
}

func getMenuItemStyle(selected bool, width int) lipgloss.Style {
	if selected {
		return menuItemSelectedStyle.Width(width)
	}
	return menuItemStyle.Width(width)
}

// ASCII Art header - compact version
const asciiHeader = `┌─┐┌─┐─┐ ┬   ┌─┐┌─┐┌─┐┌─┐
├┤ ├┤ ┌┴┬┘───├┤ ├─┤│  ├┤ 
└─┘└  ┴ └─   └  ┴ ┴└─┘└─┘`

// getContentWidth returns 80% of terminal width (for content area)
func getContentWidth(termWidth int) int {
	width := termWidth * 80 / 100
	if width < 60 {
		width = 60
	}
	if width > 100 {
		width = 100
	}
	return width
}

// renderHeader renders a compact header (just logo + version inline)
func renderHeader(version string, termWidth int) string {
	contentWidth := getContentWidth(termWidth)
	
	// Compact: just ASCII art with version on same line
	headerLine := titleStyle.Render(asciiHeader) + "  " + infoLineStyle.Render("v"+version)
	
	return lipgloss.NewStyle().Width(contentWidth).Render(headerLine)
}

// renderFooter renders a sticky footer with help and optional pagination
func renderFooter(helpText string, pagination string) string {
	if pagination != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			pagination,
			helpStyle.Render(helpText),
		)
	}
	return helpStyle.Render(helpText)
}

// calculatePadding calculates vertical padding to push footer to bottom
func calculatePadding(contentLines, footerLines, termHeight int) int {
	availableHeight := termHeight - 4
	totalNeeded := contentLines + footerLines
	if availableHeight > totalNeeded {
		return availableHeight - totalNeeded
	}
	return 0
}
