package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarques/efx-face-manager/internal/config"
	"github.com/lmarques/efx-face-manager/internal/model"
	"github.com/lmarques/efx-face-manager/internal/server"
)

// serverNewModel handles the new server dialog from server manager
type serverNewModel struct {
	cfg         *config.Config
	store       *model.Store
	servers     *server.Manager
	models      []model.Model
	width       int
	height      int
	selectedIdx int
	port        int
	modelType   model.ModelType
	focusField  int // 0=model list, 1=port, 2=type
	editingPort bool
	portBuffer  string
}

func newServerNewModel(cfg *config.Config, store *model.Store, servers *server.Manager) serverNewModel {
	models, _ := store.List()
	
	return serverNewModel{
		cfg:        cfg,
		store:      store,
		servers:    servers,
		models:     models,
		port:       servers.NextAvailablePort(8000),
		modelType:  model.TypeLM,
		focusField: 0,
	}
}

func (m serverNewModel) Init() tea.Cmd {
	return nil
}

func (m serverNewModel) Update(msg tea.Msg) (serverNewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editingPort {
			switch msg.String() {
			case "enter":
				// Parse port
				fmt.Sscanf(m.portBuffer, "%d", &m.port)
				m.editingPort = false
			case "esc":
				m.editingPort = false
				m.portBuffer = ""
			case "backspace":
				if len(m.portBuffer) > 0 {
					m.portBuffer = m.portBuffer[:len(m.portBuffer)-1]
				}
			default:
				if len(msg.String()) == 1 && msg.String()[0] >= '0' && msg.String()[0] <= '9' {
					m.portBuffer += msg.String()
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			switch m.focusField {
			case 0:
				if m.selectedIdx > 0 {
					m.selectedIdx--
				}
			}
		case "down", "j":
			switch m.focusField {
			case 0:
				if m.selectedIdx < len(m.models)-1 {
					m.selectedIdx++
				}
			}
		case "tab":
			m.focusField = (m.focusField + 1) % 3
		case "left", "h":
			if m.focusField == 2 {
				// Cycle model type backwards
				types := []model.ModelType{model.TypeLM, model.TypeMultimodal, model.TypeImageGeneration, model.TypeImageEdit, model.TypeEmbeddings, model.TypeWhisper}
				for i, t := range types {
					if t == m.modelType && i > 0 {
						m.modelType = types[i-1]
						break
					}
				}
			}
		case "right", "l":
			if m.focusField == 2 {
				// Cycle model type forward
				types := []model.ModelType{model.TypeLM, model.TypeMultimodal, model.TypeImageGeneration, model.TypeImageEdit, model.TypeEmbeddings, model.TypeWhisper}
				for i, t := range types {
					if t == m.modelType && i < len(types)-1 {
						m.modelType = types[i+1]
						break
					}
				}
			}
		case "enter":
			switch m.focusField {
			case 0:
				// Select model and proceed to config
				if m.selectedIdx < len(m.models) {
					selectedModel := m.models[m.selectedIdx]
					return m, func() tea.Msg {
						return openConfigMsg{
							model:     selectedModel.Name,
							modelType: m.modelType,
							port:      m.port,
						}
					}
				}
			case 1:
				// Edit port
				m.editingPort = true
				m.portBuffer = fmt.Sprintf("%d", m.port)
			case 2:
				// Cycle type
				types := []model.ModelType{model.TypeLM, model.TypeMultimodal, model.TypeImageGeneration, model.TypeImageEdit, model.TypeEmbeddings, model.TypeWhisper}
				for i, t := range types {
					if t == m.modelType {
						m.modelType = types[(i+1)%len(types)]
						break
					}
				}
			}
		case "esc":
			return m, func() tea.Msg { return goBackMsg{} }
		}
	}
	return m, nil
}

func (m serverNewModel) View() string {
	// Use full width for this complex page
	contentWidth := m.width - 4
	var b strings.Builder

	// Compact header
	b.WriteString(subtitleStyle.Render("Start New Server"))
	b.WriteString("\n\n")

	// Calculate panel widths - full width
	leftWidth := contentWidth*50/100 - 2
	rightWidth := contentWidth*50/100 - 2
	panelHeight := m.height - 8

	// Left panel: Model selection and config
	leftContent := m.renderConfigPanel(leftWidth)

	// Right panel: Currently running servers
	rightContent := m.renderRunningPanel(rightWidth)

	// Apply styles
	leftBorder := primary
	rightBorder := muted

	leftPanel := lipgloss.NewStyle().
		Width(leftWidth).
		Height(panelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(leftBorder).
		Padding(1, 1).
		Render(leftContent)

	rightPanel := lipgloss.NewStyle().
		Width(rightWidth).
		Height(panelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(rightBorder).
		Padding(1, 1).
		Render(rightContent)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	b.WriteString(panels)

	// Footer (no padding)
	b.WriteString("\n" + helpStyle.Render("[↵] select model  [tab] next field  [esc] cancel"))

	return appStyle.Render(b.String())
}

func (m serverNewModel) renderConfigPanel(width int) string {
	var b strings.Builder

	b.WriteString(panelTitleStyle.Render("Configure New Server"))
	b.WriteString("\n\n")

	// Model selection
	b.WriteString(sectionTitleStyle.Render("Select Model:"))
	b.WriteString("\n")

	maxVisible := 8
	startIdx := 0
	if m.selectedIdx >= maxVisible {
		startIdx = m.selectedIdx - maxVisible + 1
	}

	endIdx := startIdx + maxVisible
	if endIdx > len(m.models) {
		endIdx = len(m.models)
	}

	for i := startIdx; i < endIdx; i++ {
		mdl := m.models[i]
		line := truncateStr(mdl.Name, width-6)
		if i == m.selectedIdx && m.focusField == 0 {
			b.WriteString(optionSelectedStyle.Render("> " + line))
		} else {
			b.WriteString(optionNormalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	if len(m.models) > maxVisible {
		b.WriteString(statusMutedStyle.Render(fmt.Sprintf("  ... (%d more)", len(m.models)-maxVisible)))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Port field
	portLabel := "Port: "
	portStyle := optionNormalStyle
	if m.focusField == 1 {
		portStyle = optionSelectedStyle
	}
	
	portValue := fmt.Sprintf("[%d]", m.port)
	if m.editingPort {
		portValue = fmt.Sprintf("[%s█]", m.portBuffer)
	}
	
	portLine := portLabel + portValue
	if m.servers.IsPortInUse(m.port) {
		portLine += warningStyle.Render(" (in use!)")
	}
	b.WriteString(portStyle.Render(portLine))
	b.WriteString("\n")

	// Type field
	typeLabel := "Type: "
	typeStyle := optionNormalStyle
	if m.focusField == 2 {
		typeStyle = optionSelectedStyle
	}
	b.WriteString(typeStyle.Render(typeLabel + "[" + string(m.modelType) + " ▾]"))
	b.WriteString("\n\n")

	// Action buttons
	b.WriteString(sectionTitleStyle.Render("Actions"))
	b.WriteString("\n")
	b.WriteString("  [↵] Configure & Start\n")
	b.WriteString("  [esc] Cancel\n")

	return b.String()
}

func (m serverNewModel) renderRunningPanel(width int) string {
	var b strings.Builder

	b.WriteString(panelTitleStyle.Render("Currently Running"))
	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render(strings.Repeat("─", width-4)))
	b.WriteString("\n\n")

	list := m.servers.List()
	if len(list) == 0 {
		b.WriteString(statusMutedStyle.Render("No servers running"))
	} else {
		for _, inst := range list {
			line := fmt.Sprintf("● %-20s :%d  %s",
				truncateStr(inst.Model, 20),
				inst.Port,
				inst.Type)
			b.WriteString(infoLineStyle.Render(line))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render("Available Ports:"))
	b.WriteString("\n")
	
	// Show next few available ports
	availablePorts := []int{}
	for port := 8000; port < 8010 && len(availablePorts) < 5; port++ {
		if !m.servers.IsPortInUse(port) {
			availablePorts = append(availablePorts, port)
		}
	}
	
	if len(availablePorts) > 0 {
		portStrs := make([]string, len(availablePorts))
		for i, p := range availablePorts {
			portStrs[i] = fmt.Sprintf("%d", p)
		}
		b.WriteString(infoLineStyle.Render(strings.Join(portStrs, ", ") + ", ..."))
	}

	// Show warnings for ports in use
	b.WriteString("\n\n")
	for _, inst := range list {
		b.WriteString(warningStyle.Render(fmt.Sprintf("Port %d in use", inst.Port)))
		b.WriteString("\n")
	}

	return b.String()
}

// openConfigMsg is sent to open config panel with pre-filled values
type openConfigMsg struct {
	model     string
	modelType model.ModelType
	port      int
}
