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
	vp := viewport.New(width/2-4, height-10)
	
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
		m.viewport.Width = msg.Width/2 - 4
		m.viewport.Height = msg.Height - 10
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

	// Compact header - same as other pages
	b.WriteString(renderHeader(version, m.width))
	b.WriteString("\n")

	// Server count info
	if serverCount > 0 {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("Server Manager (%d running)", serverCount)))
	} else {
		b.WriteString(subtitleStyle.Render("Server Manager"))
	}
	b.WriteString("\n\n")

	// Calculate panel widths - full width
	leftWidth := contentWidth * 35 / 100
	rightWidth := contentWidth * 65 / 100 - 4
	panelHeight := m.height - 14

	// Left panel: Server list + controls
	leftContent := m.renderControlPanel(leftWidth, list)

	// Right panel: Log viewport
	rightContent := m.renderLogPanel(rightWidth)

	// Apply styles - align to top
	leftBorder := muted
	rightBorder := muted
	if !m.focusOnLogs {
		leftBorder = primary
	} else {
		rightBorder = primary
	}

	leftPanel := lipgloss.NewStyle().
		Width(leftWidth).
		Height(panelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(leftBorder).
		Padding(0, 1).
		Render(leftContent)

	rightPanel := lipgloss.NewStyle().
		Width(rightWidth).
		Height(panelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(rightBorder).
		Padding(0, 1).
		Render(rightContent)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	b.WriteString(panels)

	// Calculate padding to push footer to bottom
	content := b.String()
	contentLines := strings.Count(content, "\n") + 1
	padding := calculatePadding(contentLines, 1, m.height)
	b.WriteString(strings.Repeat("\n", padding))

	// Footer
	b.WriteString("\n" + helpStyle.Render("[↑/↓] select server  [s] stop  [S] stop all  [n] new  [c] clear  [tab] focus logs  [esc] menu"))

	return appStyle.Render(b.String())
}

func (m serverManagerModel) renderControlPanel(width int, list []*server.Instance) string {
	var b strings.Builder
	
	b.WriteString(panelTitleStyle.Render("Server Controls"))
	b.WriteString("\n\n")
	
	if len(list) == 0 {
		b.WriteString(statusMutedStyle.Render("No servers running"))
		b.WriteString("\n\n")
		b.WriteString(infoLineStyle.Render("Press [n] to start a new server"))
	} else {
		b.WriteString(sectionTitleStyle.Render(fmt.Sprintf("Running Servers (%d)", len(list))))
		b.WriteString("\n")
		
		// Server list
		for i, inst := range list {
			typeShort := string(inst.Type)
			if len(typeShort) > 5 {
				typeShort = typeShort[:5]
			}
			
			line := fmt.Sprintf("● %-25s :%d  %s", 
				truncateStr(inst.Model, 25), 
				inst.Port, 
				typeShort)
			
			if inst.Port == m.selectedPort {
				b.WriteString(optionSelectedStyle.Render(fmt.Sprintf("> %s", line)))
			} else {
				b.WriteString(optionNormalStyle.Render(fmt.Sprintf("  %s", line)))
			}
			b.WriteString("\n")
			
			// Show number shortcut
			if i < 9 {
				b.WriteString(statusMutedStyle.Render(fmt.Sprintf("    [%d]", i+1)))
				b.WriteString("\n")
			}
		}
		
		// Selected server details
		if m.selectedPort > 0 {
			if inst := m.servers.Get(m.selectedPort); inst != nil {
				b.WriteString("\n")
				b.WriteString(sectionTitleStyle.Render("Selected Server"))
				b.WriteString("\n")
				b.WriteString(fmt.Sprintf("Model: %s\n", inst.Model))
				b.WriteString(fmt.Sprintf("Type:  %s\n", inst.Type))
				b.WriteString(fmt.Sprintf("Port:  %d\n", inst.Port))
				b.WriteString(fmt.Sprintf("Host:  %s\n", inst.Host))
			}
		}
		
		// Actions
		b.WriteString("\n")
		b.WriteString(sectionTitleStyle.Render("Actions"))
		b.WriteString("\n")
		b.WriteString("  [s] Stop this server\n")
		if len(list) > 1 {
			b.WriteString("  [S] Stop ALL servers\n")
		}
		b.WriteString("  [n] Start new server...\n")
		b.WriteString("  [m] Back to main menu\n")
	}
	
	return b.String()
}

func (m serverManagerModel) renderLogPanel(width int) string {
	var b strings.Builder
	
	// Title with server info
	if m.selectedPort > 0 {
		if inst := m.servers.Get(m.selectedPort); inst != nil {
			b.WriteString(panelTitleStyle.Render(fmt.Sprintf("Server Output: %s :%d", truncateStr(inst.Model, 20), inst.Port)))
		} else {
			b.WriteString(panelTitleStyle.Render("Server Output"))
		}
	} else {
		b.WriteString(panelTitleStyle.Render("Server Output"))
	}
	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render(strings.Repeat("─", width-4)))
	b.WriteString("\n\n")
	
	// Viewport content
	if m.selectedPort > 0 {
		b.WriteString(m.viewport.View())
	} else {
		b.WriteString(statusMutedStyle.Render("No server selected"))
	}
	
	// Scroll indicator
	if m.selectedPort > 0 {
		b.WriteString("\n")
		b.WriteString(infoLineStyle.Render(fmt.Sprintf("%.0f%% [↑/↓] scroll  [g/G] top/bottom", m.viewport.ScrollPercent()*100)))
	}
	
	return b.String()
}
