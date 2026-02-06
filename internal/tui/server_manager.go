package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarques/efx-face-manager/internal/server"
)

// serverManagerModel handles the multi-server management view
type serverManagerModel struct {
	servers      *server.Manager
	width        int
	height       int
	selectedIdx  int
	selectedPort int
	viewport     viewport.Model
	focusOnLogs  bool
}

func newServerManagerModel(servers *server.Manager, width, height int) serverManagerModel {
	// Calculate proper viewport dimensions to prevent crash
	contentWidth := width - 4
	logPanelHeight := height - 8 - 10 // control height (8) + title/borders/footer/leading newlines (10)
	if logPanelHeight < 5 {
		logPanelHeight = 5
	}
	vp := viewport.New(contentWidth-4, logPanelHeight-3) // Account for padding and title

	m := serverManagerModel{
		servers:  servers,
		width:    width,
		height:   height,
		viewport: vp,
	}

	// Select first server if any
	list := servers.List()
	if len(list) > 0 {
		m.selectedPort = list[0].Port
		m.viewport.SetContent(servers.GetLogs(m.selectedPort))
	}

	return m
}

func (m serverManagerModel) Init() tea.Cmd {
	return nil
}

func (m serverManagerModel) Update(msg tea.Msg) (serverManagerModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case serverUpdateMsg:
		// Refresh logs if it's for the selected server
		if msg.Port == m.selectedPort {
			m.viewport.SetContent(m.servers.GetLogs(m.selectedPort))
			m.viewport.GotoBottom()
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.focusOnLogs {
				m.viewport, cmd = m.viewport.Update(msg)
			} else {
				list := m.servers.List()
				if m.selectedIdx > 0 {
					m.selectedIdx--
					if len(list) > m.selectedIdx {
						m.selectedPort = list[m.selectedIdx].Port
						m.viewport.SetContent(m.servers.GetLogs(m.selectedPort))
					}
				}
			}
		case "down", "j":
			if m.focusOnLogs {
				m.viewport, cmd = m.viewport.Update(msg)
			} else {
				list := m.servers.List()
				if m.selectedIdx < len(list)-1 {
					m.selectedIdx++
					m.selectedPort = list[m.selectedIdx].Port
					m.viewport.SetContent(m.servers.GetLogs(m.selectedPort))
				}
			}
		case "s":
			// Stop selected server
			if m.selectedPort > 0 {
				m.servers.Stop(m.selectedPort)
				// Select another server
				list := m.servers.List()
				if len(list) > 0 {
					m.selectedIdx = 0
					m.selectedPort = list[0].Port
					m.viewport.SetContent(m.servers.GetLogs(m.selectedPort))
				} else {
					m.selectedPort = 0
					m.viewport.SetContent("")
				}
			}
		case "S":
			// Stop all servers
			m.servers.StopAll()
			m.selectedPort = 0
			m.selectedIdx = 0
			m.viewport.SetContent("")
		case "n":
			// Open new server dialog
			return m, func() tea.Msg { return openNewServerMsg{} }
		case "c":
			// Open chat for selected server
			if m.selectedPort > 0 {
				return m, func() tea.Msg { return openChatMsg{port: m.selectedPort} }
			}
		case "x":
			// Clear logs
			if m.selectedPort > 0 {
				if inst := m.servers.Get(m.selectedPort); inst != nil {
					inst.Output.Clear()
					m.viewport.SetContent("")
				}
			}
		case "g":
			m.viewport.GotoTop()
		case "G":
			m.viewport.GotoBottom()
		case "tab":
			m.focusOnLogs = !m.focusOnLogs
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			// Quick select by number
			idx := int(msg.String()[0] - '1')
			list := m.servers.List()
			if idx < len(list) {
				m.selectedIdx = idx
				m.selectedPort = list[idx].Port
				m.viewport.SetContent(m.servers.GetLogs(m.selectedPort))
			}
		case "m", "esc":
			return m, func() tea.Msg { return goBackMsg{} }
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update viewport dimensions to match new layout
		contentWidth := msg.Width - 4
		logPanelHeight := msg.Height - 8 - 7 // control height (8) + title/borders/footer (7)
		if logPanelHeight < 5 {
			logPanelHeight = 5
		}
		m.viewport.Width = contentWidth - 4
		m.viewport.Height = logPanelHeight - 3 // Account for padding and title
	}

	// Update viewport for scroll
	if m.focusOnLogs {
		m.viewport, cmd = m.viewport.Update(msg)
	}

	return m, cmd
}

func (m serverManagerModel) View() string {
	list := m.servers.List()
	serverCount := len(list)

	// Use full width for this complex page
	contentWidth := m.width - 4
	var b strings.Builder

	// Add leading newlines to prevent content from being cut off at top
	b.WriteString("\n\n\n")

	// Title line at top - appStyle provides padding
	if serverCount > 0 {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("Server Manager (%d running)", serverCount)))
	} else {
		b.WriteString(subtitleStyle.Render("Server Manager"))
	}
	b.WriteString("\n")

	// VERTICAL LAYOUT: Row 1 (Server Controls with 2 columns) + Row 2 (Logs full width)

	// Row 1: Server Controls - Minimal height to maximize log space
	controlPanelHeight := 8 // Compact height - content fills without gaps
	leftColWidth := contentWidth*50/100 - 2
	rightColWidth := contentWidth*50/100 - 2

	// Left column: Server list
	leftControlContent := m.renderServerList(list)
	// Right column: Selected server details
	rightControlContent := m.renderServerDetails()

	controlBorder := muted
	if !m.focusOnLogs {
		controlBorder = primary
	}

	leftControlPanel := lipgloss.NewStyle().
		Width(leftColWidth).
		Height(controlPanelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(controlBorder).
		Padding(0, 1).
		Render(leftControlContent)

	rightControlPanel := lipgloss.NewStyle().
		Width(rightColWidth).
		Height(controlPanelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(controlBorder).
		Padding(0, 1).
		Render(rightControlContent)

	controlRow := lipgloss.JoinHorizontal(lipgloss.Top, leftControlPanel, rightControlPanel)
	b.WriteString(controlRow)
	b.WriteString("\n")

	// Row 2: Log viewport - Width must match the combined control panels above
	// Each control panel rendered width = colWidth + 2 (border), so total = leftColWidth + rightColWidth + 4
	// Log panel rendered width = logPanelWidth + 2 (border), so we need logPanelWidth + 2 = leftColWidth + rightColWidth + 4
	logPanelWidth := leftColWidth + rightColWidth + 2
	logPanelHeight := m.height - controlPanelHeight - 10 // Account for title, borders, footer, leading newlines
	if logPanelHeight < 5 {
		logPanelHeight = 5 // Minimum height to prevent crash
	}

	logContent := m.renderLogPanel()

	logBorder := muted
	if m.focusOnLogs {
		logBorder = primary
	}

	logPanel := lipgloss.NewStyle().
		Width(logPanelWidth).
		Height(logPanelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(logBorder).
		Padding(0, 1).
		Render(logContent)

	b.WriteString(logPanel)

	// Footer with shortcuts
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("[↑/↓] select server  [s] stop  [S] stop all  [n] new  [c] chat  [x] clear  [tab] focus logs  [esc] menu"))

	return appStyle.Render(b.String())
}

// renderServerList renders the left column: running servers list
func (m serverManagerModel) renderServerList(list []*server.Instance) string {
	var b strings.Builder

	b.WriteString(panelTitleStyle.Render(fmt.Sprintf("Running Servers (%d)", len(list))))
	b.WriteString("\n")

	if len(list) == 0 {
		b.WriteString(statusMutedStyle.Render("No servers running"))
	} else {
		for i, inst := range list {
			typeShort := string(inst.Type)
			if len(typeShort) > 4 {
				typeShort = typeShort[:4]
			}
			shortcut := ""
			if i < 9 {
				shortcut = fmt.Sprintf(" [%d]", i+1)
			}
			line := fmt.Sprintf("● %-30s :%d %s%s", truncateStr(inst.Model, 30), inst.Port, typeShort, shortcut)
			if inst.Port == m.selectedPort {
				b.WriteString(optionSelectedStyle.Render(fmt.Sprintf("> %s", line)))
			} else {
				b.WriteString(optionNormalStyle.Render(fmt.Sprintf("  %s", line)))
			}
			if i < len(list)-1 {
				b.WriteString("\n")
			}
		}
	}
	return b.String()
}

// renderServerDetails renders the right column: selected server details & actions
func (m serverManagerModel) renderServerDetails() string {
	var b strings.Builder

	b.WriteString(panelTitleStyle.Render("Selected Server"))
	b.WriteString("\n")
	if m.selectedPort > 0 {
		if inst := m.servers.Get(m.selectedPort); inst != nil {
			b.WriteString(fmt.Sprintf("Model: %s\n", truncateStr(inst.Model, 35)))
			b.WriteString(fmt.Sprintf("Type: %s  Port: %d  Host: %s\n", inst.Type, inst.Port, inst.Host))
		}
	} else {
		b.WriteString(statusMutedStyle.Render("No server selected\n"))
	}
	b.WriteString(sectionTitleStyle.Render("Actions") + " [s]Stop [S]ALL [n]New [m]Menu")
	return b.String()
}

func (m serverManagerModel) renderLogPanel() string {
	var b strings.Builder

	// Title with server info - no truncation, no newlines, no separator
	if m.selectedPort > 0 {
		if inst := m.servers.Get(m.selectedPort); inst != nil {
			b.WriteString(panelTitleStyle.Render(fmt.Sprintf("Server Output: %s :%d", inst.Model, inst.Port)))
		} else {
			b.WriteString(panelTitleStyle.Render("Server Output"))
		}
	} else {
		b.WriteString(panelTitleStyle.Render("Server Output"))
	}
	b.WriteString("\n")

	// Viewport content
	if m.selectedPort > 0 {
		b.WriteString(m.viewport.View())
	} else {
		b.WriteString(statusMutedStyle.Render("No server selected"))
	}

	// Scroll indicator - no extra newline
	if m.selectedPort > 0 {
		b.WriteString(" ")
		b.WriteString(infoLineStyle.Render(fmt.Sprintf("%.0f%% [↑/↓] scroll  [g/G] top/bottom", m.viewport.ScrollPercent()*100)))
	}

	return b.String()
}
