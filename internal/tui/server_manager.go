package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarques/efx-face-manager/internal/opencode"
	"github.com/lmarques/efx-face-manager/internal/server"
	"golang.design/x/clipboard"
)

// Message types for server manager
type clearStatusMsg struct{}
type opencodeExitMsg struct{ err error }
type opencodeWebExitMsg struct{ err error }

// serverManagerModel handles the multi-server management view
type serverManagerModel struct {
	servers           *server.Manager
	width             int
	height            int
	selectedIdx       int
	selectedPort      int
	viewport          viewport.Model
	focusOnLogs       bool
	statusMsg         string
	opencodeInstalled bool
	// Opencode web server state
	opencodeWebCmd     *exec.Cmd
	opencodeWebPort    int
	opencodeWebRunning bool
}

func newServerManagerModel(servers *server.Manager, width, height int) serverManagerModel {
	// Calculate proper viewport dimensions to prevent crash
	contentWidth := width - 4
	controlPanelHeight := 8
	logPanelHeight := height - controlPanelHeight - 10 // Account for title, borders, footer, leading newlines
	if logPanelHeight < 5 {
		logPanelHeight = 5
	}
	// Viewport height: logPanelHeight - 2 (border) - 1 (title+newline) = maximize log space
	vpHeight := logPanelHeight - 3
	if vpHeight < 3 {
		vpHeight = 3
	}
	vp := viewport.New(contentWidth-6, vpHeight)

	// Check if opencode is installed
	_, opencodeErr := exec.LookPath("opencode")

	m := serverManagerModel{
		servers:           servers,
		width:             width,
		height:            height,
		viewport:          vp,
		opencodeInstalled: opencodeErr == nil,
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
	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case opencodeExitMsg:
		// Opencode exited, clear env var and refresh
		os.Unsetenv("OPENCODE_CONFIG_CONTENT")
		if msg.err != nil {
			m.statusMsg = "opencode error"
		}
		return m, nil

	case opencodeWebExitMsg:
		// Opencode web server exited
		m.opencodeWebRunning = false
		m.opencodeWebCmd = nil
		m.opencodeWebPort = 0
		os.Unsetenv("OPENCODE_CONFIG_CONTENT")
		if msg.err != nil {
			m.statusMsg = "opencode web stopped"
		} else {
			m.statusMsg = "opencode web stopped"
		}
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} })

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
		case "o":
			// Copy opencode command to clipboard (only if opencode is installed)
			if m.opencodeInstalled && m.selectedPort > 0 {
				if inst := m.servers.Get(m.selectedPort); inst != nil {
					workDir, _ := os.Getwd()
					// Create runner script with merged config
					scriptPath, err := opencode.CreateRunnerScript(inst.Host, inst.Port, inst.Model)
					if err != nil {
						m.statusMsg = "Failed to create runner script"
						return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} })
					}
					// Command references the runner script
					cmdStr := fmt.Sprintf("cd %s && %s", workDir, scriptPath)
					clipboard.Write(clipboard.FmtText, []byte(cmdStr))
					m.statusMsg = "Opencode runner copied to clipboard!"
					return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} })
				}
			}
		case "O":
			// Run opencode inline (suspends TUI) - only if opencode is installed
			if m.opencodeInstalled && m.selectedPort > 0 {
				if inst := m.servers.Get(m.selectedPort); inst != nil {
					// Load and merge configs like the script generation does
					userConfig := opencode.LoadUserConfig()
					providerConfig := opencode.GenerateProviderConfig(inst.Host, inst.Port, inst.Model)
					mergedConfig := opencode.MergeConfigs(userConfig, providerConfig)
					configJSON, _ := opencode.ConfigToJSON(mergedConfig)

					cmd := tea.ExecProcess(
						exec.Command("opencode", "--model", fmt.Sprintf("mlx-community/%s", inst.Model)),
						func(err error) tea.Msg {
							return opencodeExitMsg{err: err}
						},
					)
					// Set environment variable for the command
					os.Setenv("OPENCODE_CONFIG_CONTENT", configJSON)
					return m, cmd
				}
			}
		case "w":
			// Toggle opencode web server - only if opencode is installed and server selected
			if m.opencodeInstalled && m.selectedPort > 0 {
				if m.opencodeWebRunning {
					// Stop the web server - kill entire process group
					if m.opencodeWebCmd != nil && m.opencodeWebCmd.Process != nil {
						killProcessGroup(m.opencodeWebCmd)
					}
					m.opencodeWebRunning = false
					m.opencodeWebCmd = nil
					m.opencodeWebPort = 0
					os.Unsetenv("OPENCODE_CONFIG_CONTENT")
					m.statusMsg = "opencode web stopped"
					return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} })
				} else {
					// Start the web server
					if inst := m.servers.Get(m.selectedPort); inst != nil {
						// Load and merge configs with model field set
						userConfig := opencode.LoadUserConfig()
						providerConfig := opencode.GenerateProviderConfig(inst.Host, inst.Port, inst.Model)
						mergedConfig := opencode.MergeConfigsWithModel(userConfig, providerConfig, inst.Model)
						configJSON, _ := opencode.ConfigToJSON(mergedConfig)

						// Find next available port starting from 4100 (check system-wide, not just internal)
						webPort := m.servers.NextFreeSystemPort(4100)
						workDir, _ := os.Getwd()

						// Build the command string for clipboard (debug)
						cmdStr := fmt.Sprintf("cd %s && OPENCODE_CONFIG_CONTENT='%s' opencode web --port %d", workDir, configJSON, webPort)
						clipboard.Write(clipboard.FmtText, []byte(cmdStr))

						webCmd := exec.Command("opencode", "web", "--port", fmt.Sprintf("%d", webPort))
						webCmd.Env = append(os.Environ(), "OPENCODE_CONFIG_CONTENT="+configJSON)
						// Set working directory to current directory
						webCmd.Dir = workDir
						// Create new process group so we can kill all children
						setupProcessGroup(webCmd)
						// Start the process in background
						if err := webCmd.Start(); err != nil {
							m.statusMsg = "Failed to start opencode web"
							return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} })
						}

						m.opencodeWebCmd = webCmd
						m.opencodeWebPort = webPort
						m.opencodeWebRunning = true
						m.statusMsg = fmt.Sprintf("opencode web :%d (cmd copied)", webPort)

						// Wait for process exit in background
						return m, tea.Batch(
							tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} }),
							func() tea.Msg {
								err := webCmd.Wait()
								return opencodeWebExitMsg{err: err}
							},
						)
					}
				}
			}
		case "c":
			// Open chat with selected server (only if opencode is installed)
			if m.opencodeInstalled && m.selectedPort > 0 {
				if inst := m.servers.Get(m.selectedPort); inst != nil {
					return m, func() tea.Msg {
						return openChatMsg{host: inst.Host, port: inst.Port, modelName: inst.Model}
					}
				}
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
		controlPanelHeight := 8
		logPanelHeight := msg.Height - controlPanelHeight - 10 // Account for title, borders, footer, leading newlines
		if logPanelHeight < 5 {
			logPanelHeight = 5
		}
		// Viewport height: logPanelHeight - 2 (border) - 1 (title+newline) = maximize log space
		vpHeight := logPanelHeight - 3
		if vpHeight < 3 {
			vpHeight = 3
		}
		m.viewport.Width = contentWidth - 6
		m.viewport.Height = vpHeight
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

	// Title line at top
	titleText := "Server Manager"
	if serverCount > 0 {
		titleText = fmt.Sprintf("Server Manager (%d running)", serverCount)
	}
	b.WriteString(subtitleStyle.Render(titleText))
	// Status message on next line (in green, left-aligned)
	if m.statusMsg != "" {
		b.WriteString("\n" + successStyle.Render(m.statusMsg))
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

	// Footer with shortcuts (directly after log panel - no extra newline)
	var shortcuts string
	if m.opencodeInstalled {
		shortcuts = "[o] copy cmd  [O] opencode  [w] web  [c] chat  [s] stop  [S] all  [n] new  [x] clear  [tab] logs  [esc] menu"
	} else {
		shortcuts = "[s] stop  [S] all  [n] new  [x] clear  [tab] logs  [esc] menu"
	}
	b.WriteString(helpStyle.Render(shortcuts))

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
			// Show opencode web status
			if m.opencodeWebRunning {
				b.WriteString(successStyle.Render(fmt.Sprintf("opencode web: running :%d", m.opencodeWebPort)))
			} else {
				b.WriteString(statusMutedStyle.Render("opencode web: stopped"))
			}
		}
	} else {
		b.WriteString(statusMutedStyle.Render("No server selected"))
	}
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

// StopOpencodeWeb stops the opencode web server if running
func (m *serverManagerModel) StopOpencodeWeb() {
	if m.opencodeWebRunning && m.opencodeWebCmd != nil && m.opencodeWebCmd.Process != nil {
		// Kill process group (negative PID kills the group)
		killProcessGroup(m.opencodeWebCmd)
		m.opencodeWebRunning = false
		m.opencodeWebCmd = nil
		m.opencodeWebPort = 0
		os.Unsetenv("OPENCODE_CONFIG_CONTENT")
	}
}
