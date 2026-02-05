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
	white     = lipgloss.Color("#FFFFFF")
)

// App frame
var appStyle = lipgloss.NewStyle().
	Padding(1, 2)

// Title - ASCII art header style
var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(primary).
	MarginBottom(1)

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

// Status indicators
var statusMutedStyle = lipgloss.NewStyle().
	Foreground(muted)

// Help bar
var helpStyle = lipgloss.NewStyle().
	Foreground(muted).
	MarginTop(1)

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
	Foreground(white).
	Background(primary).
	Bold(true)

// Option normal
var optionNormalStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#E5E7EB"))

// ASCII Art header - mini version for other pages (option 4)
const asciiHeaderMini = `█▀▀ █▀▀ ▀▄▀   █▀▀ █▀█ █▀▀ █▀▀
██▄ █▀  ▄▀▄ ▬ █▀  █▀█ █▄▄ ██▄`

// Helper functions for dynamic panel sizing
func getPanelStyle(width, height int, focused bool) lipgloss.Style {
	style := panelStyle
	if focused {
		style = panelFocusedStyle
	}
	return style.Width(width).Height(height)
}

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

// renderHeader renders a compact header with mini logo + version
func renderHeader(version string, termWidth int) string {
	contentWidth := getContentWidth(termWidth)
	
	// Mini logo on first line
	logoLine := titleStyle.Render(asciiHeaderMini)
	
	// Full title and version on second line, left-aligned
	titleLine := infoLineStyle.Align(lipgloss.Left).Render("MLX Hugging Face Manager - ©efx v"+version)
	
	// Join vertically with left alignment
	header := lipgloss.JoinVertical(lipgloss.Left, logoLine, titleLine)
	
	return lipgloss.NewStyle().Width(contentWidth).Render(header)
}

// renderMenuHeader renders the main menu header with full title
func renderMenuHeader(version string, termWidth int) string {
	contentWidth := getContentWidth(termWidth)
	
	// Mini logo on first line
	logoLine := titleStyle.Render(asciiHeaderMini)
	
	// Full title and version on second line, left-aligned
	titleLine := infoLineStyle.Align(lipgloss.Left).Render("MLX Hugging Face Manager - ©efx v"+version)
	
	// Join vertically with left alignment
	header := lipgloss.JoinVertical(lipgloss.Left, logoLine, titleLine)
	
	return lipgloss.NewStyle().Width(contentWidth).Render(header)
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
