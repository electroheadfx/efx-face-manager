package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarques/efx-face-manager/internal/config"
	"github.com/lmarques/efx-face-manager/internal/model"
	"github.com/lmarques/efx-face-manager/internal/server"
)

// Panel indices - 3 sections: action bar, options, setup
const (
	panelActionBar = iota
	panelOptions
	panelSetup
)

// Control actions (inline above panels)
const (
	actionRun = iota
	actionPort
	actionTrustToggle
	actionCancel
)

// configPanelModel handles the configuration view
type configPanelModel struct {
	config       server.Config
	cfg          *config.Config
	servers      *server.Manager
	width        int
	height       int
	focusedPanel int
	
	// Controls panel
	controlSelected int
	
	// Options panel
	options         []configOption
	optionSelected  int
	
	// Setup panel (for current option)
	setupChoices    []string
	setupSelected   int
	showSetup       bool
	editingValue    bool
	editBuffer      string
}

type configOption struct {
	key      string
	label    string
	value    string
	choices  []string // nil for text input
	isToggle bool
}

func newConfigPanelModel(cfg server.Config, appCfg *config.Config, servers *server.Manager) configPanelModel {
	m := configPanelModel{
		config:       cfg,
		cfg:          appCfg,
		servers:      servers,
		focusedPanel: panelActionBar,
	}
	m.buildOptions()
	return m
}

func (m *configPanelModel) buildOptions() {
	m.options = []configOption{}
	
	switch m.config.Type {
	case model.TypeLM, model.TypeMultimodal:
		m.options = append(m.options,
			configOption{key: "context_length", label: "Context length", value: formatInt(m.config.ContextLength)},
			configOption{key: "auto_tool_choice", label: "Auto tool choice", value: "enabled", isToggle: true},
			configOption{key: "tool_call_parser", label: "Tool call parser", value: formatStr(m.config.ToolCallParser), 
				choices: []string{"qwen3", "glm4_moe", "qwen3_coder", "qwen3_moe", "qwen3_next", "qwen3_vl", "harmony", "minimax_m2", "(clear)"}},
			configOption{key: "reasoning_parser", label: "Reasoning parser", value: formatStr(m.config.ReasoningParser),
				choices: []string{"qwen3", "glm4_moe", "qwen3_coder", "qwen3_moe", "qwen3_next", "qwen3_vl", "harmony", "minimax_m2", "glm47_flash", "(clear)"}},
			configOption{key: "message_converter", label: "Message converter", value: formatStr(m.config.MessageConverter),
				choices: []string{"glm4_moe", "minimax_m2", "nemotron3_nano", "qwen3_coder", "(clear)"}},
			configOption{key: "chat_template_file", label: "Chat template file", value: formatStr(m.config.ChatTemplateFile)},
			configOption{key: "debug", label: "Debug mode", value: formatBool(m.config.Debug), isToggle: true},
			configOption{key: "trust_remote_code", label: "Trust remote code", value: formatBool(m.config.TrustRemoteCode), isToggle: true},
		)
		if m.config.Type == model.TypeMultimodal {
			m.options = append(m.options,
				configOption{key: "disable_auto_resize", label: "Disable auto resize", value: formatBool(m.config.DisableAutoResize), isToggle: true},
			)
		}
		
	case model.TypeImageGeneration:
		m.options = append(m.options,
			configOption{key: "config_name", label: "Config name", value: m.config.ConfigName,
				choices: []string{"flux-schnell", "flux-dev", "flux-krea-dev", "qwen-image", "z-image-turbo", "fibo"}},
			configOption{key: "quantize", label: "Quantize level", value: formatInt(m.config.Quantize),
				choices: []string{"4", "8", "16"}},
			configOption{key: "lora_paths", label: "LoRA paths", value: formatStr(m.config.LoraPaths)},
			configOption{key: "lora_scales", label: "LoRA scales", value: formatStr(m.config.LoraScales)},
		)
		
	case model.TypeImageEdit:
		m.options = append(m.options,
			configOption{key: "config_name", label: "Config name", value: m.config.ConfigName,
				choices: []string{"flux-kontext-dev", "qwen-image-edit"}},
			configOption{key: "quantize", label: "Quantize level", value: formatInt(m.config.Quantize),
				choices: []string{"4", "8", "16"}},
			configOption{key: "lora_paths", label: "LoRA paths", value: formatStr(m.config.LoraPaths)},
			configOption{key: "lora_scales", label: "LoRA scales", value: formatStr(m.config.LoraScales)},
		)
		
	case model.TypeWhisper:
		m.options = append(m.options,
			configOption{key: "max_concurrency", label: "Max concurrency", value: formatInt(m.config.MaxConcurrency)},
			configOption{key: "queue_timeout", label: "Queue timeout", value: formatInt(m.config.QueueTimeout)},
			configOption{key: "queue_size", label: "Queue size", value: formatInt(m.config.QueueSize)},
		)
		
	case model.TypeEmbeddings:
		m.options = append(m.options,
			configOption{key: "max_concurrency", label: "Max concurrency", value: formatInt(m.config.MaxConcurrency)},
			configOption{key: "queue_timeout", label: "Queue timeout", value: formatInt(m.config.QueueTimeout)},
			configOption{key: "queue_size", label: "Queue size", value: formatInt(m.config.QueueSize)},
		)
	}
	
	// Common server options
	m.options = append(m.options,
		configOption{key: "port", label: "Port", value: fmt.Sprintf("%d", m.config.Port)},
		configOption{key: "host", label: "Host", value: m.config.Host},
		configOption{key: "log_level", label: "Log level", value: formatStr(m.config.LogLevel),
			choices: []string{"DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL", "(clear)"}},
	)
	
	// Done and Cancel removed - actions are in action bar only
}

func (m configPanelModel) Init() tea.Cmd {
	return nil
}

func (m configPanelModel) Update(msg tea.Msg) (configPanelModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if m.editingValue {
				return m, nil
			}
			switch m.focusedPanel {
			case panelActionBar:
				if m.controlSelected > 0 {
					m.controlSelected--
				}
			case panelSetup:
				m.focusedPanel = panelOptions
			}
		case "right", "l":
			if m.editingValue {
				return m, nil
			}
			switch m.focusedPanel {
			case panelActionBar:
				if m.controlSelected < actionCancel {
					m.controlSelected++
				}
			case panelOptions:
				m.focusedPanel = panelSetup
				m.updateSetupPanel()
			}
		case "up", "k":
			if m.editingValue {
				return m, nil
			}
			switch m.focusedPanel {
			case panelActionBar:
				// Stay in action bar
			case panelOptions:
				if m.optionSelected > 0 {
					m.optionSelected--
					m.updateSetupPanel()
				} else {
					m.focusedPanel = panelActionBar
				}
			case panelSetup:
				if m.setupSelected > 0 {
					m.setupSelected--
				}
			}
		case "down", "j":
			if m.editingValue {
				return m, nil
			}
			switch m.focusedPanel {
			case panelActionBar:
				m.focusedPanel = panelOptions
			case panelOptions:
				if m.optionSelected < len(m.options)-1 {
					m.optionSelected++
					m.updateSetupPanel()
				}
			case panelSetup:
				if m.setupSelected < len(m.setupChoices)-1 {
					m.setupSelected++
				}
			}
		case "enter":
			return m.handleEnter()
		case "tab":
			// Tab cycles: Run -> Port -> Options -> Run
			if m.focusedPanel == panelActionBar {
				// Move within action bar or to options
				if m.controlSelected < actionCancel {
					m.controlSelected++
				} else {
					m.focusedPanel = panelOptions
				}
			} else {
				// From options back to Run button
				m.focusedPanel = panelActionBar
				m.controlSelected = actionRun
			}
		case "shift+tab":
			// Shift+Tab reverse: Options -> Port -> Run -> Options
			if m.focusedPanel == panelOptions || m.focusedPanel == panelSetup {
				m.focusedPanel = panelActionBar
				m.controlSelected = actionCancel
			} else if m.controlSelected > actionRun {
				m.controlSelected--
			} else {
				m.focusedPanel = panelOptions
			}
		case "esc":
			if m.editingValue {
				m.editingValue = false
				m.editBuffer = ""
			}
			// Don't handle ESC when not editing - let app.go global handler use history
		case "q":
			// Navigate to homepage when not editing
			if !m.editingValue {
				return m, func() tea.Msg { return goBackMsg{} }
			}
		case "backspace":
			if m.editingValue && len(m.editBuffer) > 0 {
				m.editBuffer = m.editBuffer[:len(m.editBuffer)-1]
			}
		default:
			if m.editingValue && len(msg.String()) == 1 {
				m.editBuffer += msg.String()
			}
		}
	}
	return m, nil
}

func (m *configPanelModel) handleEnter() (configPanelModel, tea.Cmd) {
	switch m.focusedPanel {
	case panelActionBar:
		switch m.controlSelected {
		case actionRun:
			// Run button just runs the server with current port
			// Auto-increment port if in use
			if m.servers.IsPortInUse(m.config.Port) {
				m.config.Port = m.servers.NextAvailablePort(m.config.Port)
			}
			_, err := m.servers.Start(m.config)
			if err != nil {
				return *m, nil
			}
			return *m, func() tea.Msg {
				return serverStartedMsg{port: m.config.Port}
			}
		case actionPort:
			// Port field - toggle editing mode
			if !m.editingValue {
				// Enter edit mode
				m.editingValue = true
				m.editBuffer = fmt.Sprintf("%d", m.config.Port)
			} else {
				// Save edited port
				if port, err := strconv.Atoi(m.editBuffer); err == nil && port > 0 && port <= 65535 {
					m.config.Port = port
					m.editingValue = false
					m.editBuffer = ""
				} else {
					// Invalid port, keep editing
					return *m, nil
				}
			}
		case actionTrustToggle:
			m.config.TrustRemoteCode = !m.config.TrustRemoteCode
			m.buildOptions()
		case actionCancel:
			return *m, func() tea.Msg { return goBackMsg{} }
		}
		return *m, nil

	case panelOptions:
		opt := m.options[m.optionSelected]
		if opt.key == "done" {
			_, err := m.servers.Start(m.config)
			if err != nil {
				return *m, nil
			}
			return *m, func() tea.Msg {
				return serverStartedMsg{port: m.config.Port}
			}
		} else if opt.key == "cancel" {
			return *m, func() tea.Msg { return goBackMsg{} }
		} else if opt.isToggle {
			m.toggleOption(opt.key)
			m.buildOptions()
		} else if opt.choices != nil {
			m.focusedPanel = panelSetup
			m.updateSetupPanel()
		} else {
			// Text input - move to setup panel for editing
			m.focusedPanel = panelSetup
			m.editingValue = true
			m.editBuffer = opt.value
			if m.editBuffer == "(not set)" {
				m.editBuffer = ""
			}
		}
		
	case panelSetup:
		if m.editingValue {
			// Save the edited value
			m.applyTextValue(m.options[m.optionSelected].key, m.editBuffer)
			m.editingValue = false
			m.editBuffer = ""
			m.buildOptions()
			m.focusedPanel = panelOptions
		} else if len(m.setupChoices) > 0 {
			choice := m.setupChoices[m.setupSelected]
			m.applyChoice(m.options[m.optionSelected].key, choice)
			m.buildOptions()
			m.focusedPanel = panelOptions
		}
	}
	
	return *m, nil
}

func (m *configPanelModel) updateSetupPanel() {
	if m.optionSelected < len(m.options) {
		opt := m.options[m.optionSelected]
		m.setupChoices = opt.choices
		m.setupSelected = 0
		m.showSetup = opt.choices != nil
	}
}

func (m *configPanelModel) toggleOption(key string) {
	switch key {
	case "trust_remote_code":
		m.config.TrustRemoteCode = !m.config.TrustRemoteCode
	case "debug":
		m.config.Debug = !m.config.Debug
	case "disable_auto_resize":
		m.config.DisableAutoResize = !m.config.DisableAutoResize
	}
}

func (m *configPanelModel) applyChoice(key, choice string) {
	if choice == "(clear)" {
		choice = ""
	}
	switch key {
	case "tool_call_parser":
		m.config.ToolCallParser = choice
	case "reasoning_parser":
		m.config.ReasoningParser = choice
	case "message_converter":
		m.config.MessageConverter = choice
	case "config_name":
		m.config.ConfigName = choice
	case "quantize":
		fmt.Sscanf(choice, "%d", &m.config.Quantize)
	case "log_level":
		m.config.LogLevel = choice
	}
}

func (m *configPanelModel) applyTextValue(key, value string) {
	switch key {
	case "context_length":
		fmt.Sscanf(value, "%d", &m.config.ContextLength)
	case "port":
		fmt.Sscanf(value, "%d", &m.config.Port)
	case "host":
		m.config.Host = value
	case "chat_template_file":
		m.config.ChatTemplateFile = value
	case "lora_paths":
		m.config.LoraPaths = value
	case "lora_scales":
		m.config.LoraScales = value
	case "max_concurrency":
		fmt.Sscanf(value, "%d", &m.config.MaxConcurrency)
	case "queue_timeout":
		fmt.Sscanf(value, "%d", &m.config.QueueTimeout)
	case "queue_size":
		fmt.Sscanf(value, "%d", &m.config.QueueSize)
	}
}

func (m configPanelModel) View() string {
	contentWidth := m.width - 4
	var b strings.Builder

	// Model path and configuration on one line, gray text
	modelPath := m.cfg.ModelDir + "/" + m.config.Model
	headerLine := fmt.Sprintf("Model: %s    -    Configuration: %s", modelPath, m.config.Type)
	b.WriteString(infoLineStyle.Render(headerLine))
	b.WriteString("\n\n")

	// Command preview - wrap to multiple lines, white text
	args := m.config.BuildArgs()
	cmdStr := "mlx-openai-server " + strings.Join(args, " ")
	cmdLines := wrapText(cmdStr, contentWidth-4)
	for _, line := range cmdLines {
		b.WriteString(optionNormalStyle.Render(line) + "\n")
	}
	b.WriteString("\n")

	// Two columns: Options (50%) | Setup (50%)
	optionsWidth := contentWidth*50/100 - 2
	setupWidth := contentWidth*50/100 - 2
	panelHeight := m.height - 20

	// Action bar in a box (horizontal) - Width must match the combined panels below
	// Each panel rendered width = panelWidth + 2 (border), so total = optionsWidth + setupWidth + 4
	// Action bar rendered width = actionBarWidth + 2 (border), so we need actionBarWidth + 2 = optionsWidth + setupWidth + 4
	actionBarWidth := optionsWidth + setupWidth + 2
	actionBar := m.renderActionBar()
	actionBoxStyle := panelStyle.Width(actionBarWidth)
	if m.focusedPanel == panelActionBar {
		actionBoxStyle = panelFocusedStyle.Width(actionBarWidth)
	}
	b.WriteString(actionBoxStyle.Render(actionBar))
	b.WriteString("\n\n")

	optionsContent := m.renderOptionsPanel(optionsWidth)
	setupContent := m.renderSetupPanel()

	// Count lines in each panel content to find the maximum
	optionsLines := strings.Count(optionsContent, "\n") + 1
	setupLines := strings.Count(setupContent, "\n") + 1
	maxLines := optionsLines
	if setupLines > maxLines {
		maxLines = setupLines
	}
	
	// Pad both panels to have exactly maxLines
	if optionsLines < maxLines {
		optionsContent += strings.Repeat("\n", maxLines-optionsLines)
	}
	if setupLines < maxLines {
		setupContent += strings.Repeat("\n", maxLines-setupLines)
	}

	optionsBox := getPanelStyle(optionsWidth, panelHeight, m.focusedPanel == panelOptions).Render(optionsContent)
	setupBox := getPanelStyle(setupWidth, panelHeight, m.focusedPanel == panelSetup).Render(setupContent)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, optionsBox, setupBox)
	b.WriteString(panels)

	// Calculate padding to push footer to bottom
	content := b.String()
	contentLines := strings.Count(content, "\n") + 1
	padding := calculatePadding(contentLines, 1, m.height)
	b.WriteString(strings.Repeat("\n", padding))

	// Footer
	b.WriteString("\n" + helpStyle.Render("[↑/↓] navigate  [←/→] select  [↵] confirm  [tab] switch  [esc] back"))

	return appStyle.Render(b.String())
}

// wrapText wraps text to fit within maxWidth
func wrapText(text string, maxWidth int) []string {
	if len(text) <= maxWidth {
		return []string{text}
	}
	var lines []string
	for len(text) > maxWidth {
		// Find last space before maxWidth
		breakPoint := maxWidth
		for i := maxWidth; i > 0; i-- {
			if text[i] == ' ' {
				breakPoint = i
				break
			}
		}
		lines = append(lines, text[:breakPoint])
		text = text[breakPoint:]
		if len(text) > 0 && text[0] == ' ' {
			text = text[1:]
		}
	}
	if len(text) > 0 {
		lines = append(lines, text)
	}
	return lines
}

func (m configPanelModel) renderActionBar() string {
	// Separate Run button and Port field
	runLabel := "▶ Run"
	
	var portLabel string
	if m.editingValue && m.focusedPanel == panelActionBar && m.controlSelected == actionPort {
		// Show editable port when in edit mode
		portLabel = fmt.Sprintf("Port: %s_", m.editBuffer)
	} else {
		portLabel = fmt.Sprintf("Port: %d", m.config.Port)
	}
	
	trustLabel := "🔐 Trust: "
	if m.config.TrustRemoteCode {
		trustLabel += "on"
	} else {
		trustLabel += "off"
	}
	
	controls := []string{runLabel, portLabel, trustLabel, "✖ Cancel"}
	
	var parts []string
	for i, ctrl := range controls {
		style := optionNormalStyle
		if i == m.controlSelected && m.focusedPanel == panelActionBar {
			style = optionSelectedStyle
		}
		parts = append(parts, style.Render(ctrl))
	}
	
	return strings.Join(parts, "     ")
}

func (m configPanelModel) renderOptionsPanel(width int) string {
	title := panelTitleStyle.Render(fmt.Sprintf("Configure %s options", m.config.Type))
	
	// Calculate sub-column widths: 50% key, 50% value
	keyWidth := (width - 6) / 2
	
	var content string
	for i, opt := range m.options {
		label := opt.label
		value := opt.value
		
		// Format with fixed columns
		var line string
		if opt.value == "" {
			line = label
		} else {
			line = fmt.Sprintf("%-*s %s", keyWidth, label+":", value)
		}
		
		if i == m.optionSelected && m.focusedPanel == panelOptions {
			content += optionSelectedStyle.Render("> " + line) + "\n"
		} else {
			content += optionNormalStyle.Render("  " + line) + "\n"
		}
	}
	
	return lipgloss.JoinVertical(lipgloss.Left, title, "", content)
}

func (m configPanelModel) renderSetupPanel() string {
	if m.optionSelected >= len(m.options) {
		return ""
	}
	
	opt := m.options[m.optionSelected]
	title := panelTitleStyle.Render(opt.label)
	
	// Text input mode - show in this panel
	if m.editingValue {
		inputLine := m.editBuffer + "█"
		return lipgloss.JoinVertical(lipgloss.Left, 
			title, 
			"", 
			optionSelectedStyle.Render(inputLine),
			"",
			infoLineStyle.Render("Enter to save, Esc to cancel"),
		)
	}
	
	if opt.choices == nil {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", infoLineStyle.Render("Press Enter to edit"))
	}
	
	var content string
	for i, choice := range m.setupChoices {
		if i == m.setupSelected && m.focusedPanel == panelSetup {
			content += optionSelectedStyle.Render("> " + choice) + "\n"
		} else {
			content += optionNormalStyle.Render("  " + choice) + "\n"
		}
	}
	
	return lipgloss.JoinVertical(lipgloss.Left, title, "", content)
}

// Helper functions
func formatInt(v int) string {
	if v == 0 {
		return "(not set)"
	}
	return fmt.Sprintf("%d", v)
}

func formatStr(v string) string {
	if v == "" {
		return "(not set)"
	}
	return v
}

func formatBool(v bool) string {
	if v {
		return "enabled"
	}
	return "disabled"
}
