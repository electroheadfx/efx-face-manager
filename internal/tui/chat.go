package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarques/efx-face-manager/internal/chat"
)

// Message types for chat
type chatResponseMsg struct {
	content string
	err     error
}

// chatModel handles the chat view
type chatModel struct {
	// Server connection
	port      int
	host      string
	modelName string
	runner    *chat.Runner

	// Conversation
	conversation *chat.Conversation
	storage      *chat.Storage

	// UI state
	viewport        viewport.Model
	textInput       textinput.Model
	sending         bool
	err             error
	spinner         spinner.Model
	focusOnInput    bool // true = input, false = viewport
	currentExchange int  // -1 = show all, 0+ = specific exchange
	showAll         bool // toggle between paginated and scroll view

	// Markdown renderer
	mdRenderer *glamour.TermRenderer

	// Dimensions
	width  int
	height int
}

func newChatModel(host string, port int, modelName string, width, height int) chatModel {
	// Create runner and storage
	runner := chat.NewRunner(host, port, modelName)
	storage := chat.NewStorage()

	// Try to load existing conversation or create new one
	conv, _ := storage.FindByModelAndPort(modelName, port)
	if conv == nil {
		conv = chat.NewConversation(modelName, port)
	}

	// Create markdown renderer (use dracula style for better syntax highlighting)
	mdRenderer, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("dracula"),
		glamour.WithWordWrap(width-16),
	)

	// Create viewport for chat history
	vp := viewport.New(width-6, height-12)
	vp.SetContent(renderMessages(conv.Messages, mdRenderer))
	vp.GotoBottom()

	// Create spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(primary)

	// Create text input
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 2000
	ti.Width = width - 12
	ti.Prompt = "> "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(primary)

	return chatModel{
		port:            port,
		host:            host,
		modelName:       modelName,
		runner:          runner,
		conversation:    conv,
		storage:         storage,
		viewport:        vp,
		spinner:         sp,
		textInput:       ti,
		mdRenderer:      mdRenderer,
		focusOnInput:    true,
		showAll:         true,
		currentExchange: -1,
		width:           width,
		height:          height,
	}
}

func newChatModelWithConversation(host string, port int, modelName string, conv *chat.Conversation, width, height int) chatModel {
	// Create runner and storage
	runner := chat.NewRunner(host, port, modelName)
	storage := chat.NewStorage()

	// Create markdown renderer (use dracula style for better syntax highlighting)
	mdRenderer, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("dracula"),
		glamour.WithWordWrap(width-16),
	)

	// Create viewport for chat history
	vp := viewport.New(width-6, height-12)
	vp.SetContent(renderMessages(conv.Messages, mdRenderer))
	vp.GotoBottom()

	// Create spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(primary)

	// Create text input
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 2000
	ti.Width = width - 12
	ti.Prompt = "> "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(primary)

	return chatModel{
		port:            port,
		host:            host,
		modelName:       modelName,
		runner:          runner,
		conversation:    conv,
		storage:         storage,
		viewport:        vp,
		spinner:         sp,
		textInput:       ti,
		mdRenderer:      mdRenderer,
		focusOnInput:    true,
		showAll:         true,
		currentExchange: -1,
		width:           width,
		height:          height,
	}
}

func (m chatModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m chatModel) Update(msg tea.Msg) (chatModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case chatResponseMsg:
		m.sending = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Add assistant response to conversation
			m.conversation.AddMessage("assistant", msg.content)
			// Save conversation
			m.storage.Save(m.conversation)
			// Update viewport
			m.viewport.SetContent(renderMessages(m.conversation.Messages, m.mdRenderer))
			m.viewport.GotoBottom()
		}
		m.focusOnInput = true
		m.textInput.Focus()
		return m, textinput.Blink

	case spinner.TickMsg:
		if m.sending {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.MouseMsg:
		// Handle mouse wheel scrolling on viewport
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.viewport.LineUp(3)
			case tea.MouseButtonWheelDown:
				m.viewport.LineDown(3)
			}
		}
		return m, nil

	case tea.KeyMsg:
		// Don't process keys while sending
		if m.sending {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, func() tea.Msg { return goBackMsg{} }
		case "tab":
			// Toggle focus between viewport and input
			m.focusOnInput = !m.focusOnInput
			if m.focusOnInput {
				m.textInput.Focus()
				return m, textinput.Blink
			}
			m.textInput.Blur()
			return m, nil
		case "ctrl+n":
			// New conversation (keep file)
			m.conversation = chat.NewConversation(m.modelName, m.port)
			m.viewport.SetContent(renderMessages(m.conversation.Messages, m.mdRenderer))
			m.err = nil
			m.focusOnInput = true
			m.textInput.Focus()
			m.textInput.Reset()
			return m, textinput.Blink
		case "ctrl+d":
			// Delete conversation file and start fresh
			if m.conversation != nil && m.conversation.ID != "" {
				m.storage.Delete(m.conversation.ID)
			}
			m.conversation = chat.NewConversation(m.modelName, m.port)
			m.viewport.SetContent(renderMessages(m.conversation.Messages, m.mdRenderer))
			m.err = nil
			m.focusOnInput = true
			m.textInput.Focus()
			m.textInput.Reset()
			return m, textinput.Blink
		case "ctrl+l":
			// Open chat history
			return m, func() tea.Msg {
				return openChatHistoryMsg{host: m.host, port: m.port, modelName: m.modelName}
			}
		case "enter":
			if m.focusOnInput && len(strings.TrimSpace(m.textInput.Value())) > 0 {
				// Send message
				userMsg := strings.TrimSpace(m.textInput.Value())
				m.textInput.Reset()
				m.err = nil

				// Add user message to conversation
				m.conversation.AddMessage("user", userMsg)
				m.storage.Save(m.conversation)

				// Update viewport immediately with user message
				m.viewport.SetContent(renderMessages(m.conversation.Messages, m.mdRenderer))
				m.viewport.GotoBottom()

				// Start sending
				m.sending = true

				// Build prompt with history
				prompt := chat.BuildPromptWithHistory(m.conversation.Messages[:len(m.conversation.Messages)-1], userMsg)

				// Run async
				return m, tea.Batch(
					m.spinner.Tick,
					m.runOpencode(prompt),
				)
			}
		case "up", "down", "pgup", "pgdown":
			// Pass to viewport for scrolling (works in both focus modes)
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		case "left", "h":
			// Previous exchange (paginated view)
			if !m.showAll {
				if m.currentExchange > 0 {
					m.currentExchange--
					m.updateViewportForExchange()
				}
			}
		case "right", "l":
			// Next exchange (paginated view)
			if !m.showAll {
				totalExchanges := m.countExchanges()
				if m.currentExchange < totalExchanges-1 {
					m.currentExchange++
					m.updateViewportForExchange()
				}
			}
		case "ctrl+p":
			// Toggle between paginated and scroll view
			m.showAll = !m.showAll
			if m.showAll {
				// Returning to scroll mode - scroll to the exchange we were viewing
				previousExchange := m.currentExchange
				m.currentExchange = -1
				m.viewport.SetContent(renderMessages(m.conversation.Messages, m.mdRenderer))
				// Scroll to exact position of the exchange we were viewing
				if previousExchange >= 0 {
					offsets := m.getExchangeLineOffsets()
					if previousExchange < len(offsets) {
						m.viewport.SetYOffset(offsets[previousExchange])
					}
				}
			} else {
				// Entering paginated mode - estimate which exchange is currently visible
				currentOffset := m.viewport.YOffset
				m.currentExchange = m.getExchangeAtOffset(currentOffset)
				m.updateViewportForExchange()
			}
		}

		// Pass other keys to textinput when focused
		if m.focusOnInput {
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 6
		m.viewport.Height = msg.Height - 12
		m.textInput.Width = msg.Width - 12
		// Recreate markdown renderer with new width
		m.mdRenderer, _ = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(msg.Width-16),
		)
		m.viewport.SetContent(renderMessages(m.conversation.Messages, m.mdRenderer))
	}

	return m, tea.Batch(cmds...)
}

func (m chatModel) runOpencode(prompt string) tea.Cmd {
	runner := m.runner
	return func() tea.Msg {
		response, err := runner.Run(prompt)
		return chatResponseMsg{content: response, err: err}
	}
}

// countExchanges returns the number of message exchanges (user+assistant pairs)
func (m chatModel) countExchanges() int {
	count := 0
	for _, msg := range m.conversation.Messages {
		if msg.Role == "user" {
			count++
		}
	}
	return count
}

// getExchangeMessages returns messages for a specific exchange index
func (m chatModel) getExchangeMessages(exchangeIdx int) []chat.Message {
	var result []chat.Message
	currentExchange := -1
	
	for _, msg := range m.conversation.Messages {
		if msg.Role == "user" {
			currentExchange++
		}
		if currentExchange == exchangeIdx {
			result = append(result, msg)
		}
		if currentExchange > exchangeIdx {
			break
		}
	}
	return result
}

// updateViewportForExchange updates viewport content for current exchange
func (m *chatModel) updateViewportForExchange() {
	if m.currentExchange >= 0 {
		msgs := m.getExchangeMessages(m.currentExchange)
		m.viewport.SetContent(renderMessages(msgs, m.mdRenderer))
		m.viewport.GotoTop()
	}
}

// getExchangeLineOffsets calculates the starting line offset for each exchange
// Returns a slice where index i contains the line number where exchange i starts
func (m chatModel) getExchangeLineOffsets() []int {
	offsets := []int{}
	
	// Render messages cumulatively to get accurate line positions
	for exchIdx := 0; exchIdx < m.countExchanges(); exchIdx++ {
		if exchIdx == 0 {
			offsets = append(offsets, 0)
		} else {
			// Get all messages up to (but not including) this exchange
			var msgsBeforeExchange []chat.Message
			currentExch := -1
			for _, msg := range m.conversation.Messages {
				if msg.Role == "user" {
					currentExch++
					if currentExch >= exchIdx {
						break
					}
				}
				msgsBeforeExchange = append(msgsBeforeExchange, msg)
			}
			// Render and count lines
			rendered := renderMessages(msgsBeforeExchange, m.mdRenderer)
			// Count lines including the trailing separator
			lineCount := strings.Count(rendered, "\n")
			// Add 2 for the separator between exchanges
			offsets = append(offsets, lineCount+2)
		}
	}
	
	return offsets
}

// getExchangeAtOffset returns which exchange is visible at a given line offset
func (m chatModel) getExchangeAtOffset(offset int) int {
	offsets := m.getExchangeLineOffsets()
	if len(offsets) == 0 {
		return 0
	}
	
	// Find the exchange that contains this offset
	// Use the next exchange's start to determine boundaries
	for i := len(offsets) - 1; i >= 0; i-- {
		if offset >= offsets[i] {
			return i
		}
	}
	return 0
}

func (m chatModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString("\n\n")
	b.WriteString(subtitleStyle.Render(fmt.Sprintf("Chat - %s :%d", m.modelName, m.port)))
	b.WriteString("\n")

	// Error display
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	// Chat viewport - highlight border based on focus
	viewportBorderColor := muted
	if !m.focusOnInput {
		viewportBorderColor = primary
	}
	chatBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(viewportBorderColor).
		Width(m.width - 6).
		Height(m.height - 12).
		Padding(0, 1)

	b.WriteString(chatBorder.Render(m.viewport.View()))

	// Pagination bullets (if in paginated mode) - show directly under viewport
	totalExchanges := m.countExchanges()
	if !m.showAll && totalExchanges > 0 {
		var bullets strings.Builder
		maxBullets := 25
		if totalExchanges <= maxBullets {
			for i := 0; i < totalExchanges; i++ {
				if i == m.currentExchange {
					bullets.WriteString("●")
				} else {
					bullets.WriteString("○")
				}
				if i < totalExchanges-1 {
					bullets.WriteString(" ")
				}
			}
		} else {
			// Abbreviated view for many exchanges
			bullets.WriteString(fmt.Sprintf("%d/%d", m.currentExchange+1, totalExchanges))
		}
		b.WriteString(helpStyle.Render(fmt.Sprintf("  Exchange %d/%d  %s", m.currentExchange+1, totalExchanges, bullets.String())))
	}
	b.WriteString("\n")

	// Input area - highlight border based on focus
	inputBorderColor := muted
	if m.focusOnInput {
		inputBorderColor = primary
	}
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(inputBorderColor).
		Width(m.width - 6).
		Padding(0, 1)

	var inputContent string
	if m.sending {
		inputContent = m.spinner.View() + " Thinking..."
	} else {
		inputContent = m.textInput.View()
	}
	b.WriteString(inputStyle.Render(inputContent))
	b.WriteString("\n")

	// Help
	var helpText string
	if m.showAll {
		helpText = "[enter] send  [tab] focus  [ctrl+p] pages  [ctrl+l] history  [ctrl+n] new  [esc] back"
	} else {
		helpText = "[←/→] page  [ctrl+p] scroll  [enter] send  [ctrl+l] history  [esc] back"
	}
	b.WriteString(helpStyle.Render(helpText))

	return appStyle.Render(b.String())
}

// stripModelArtifacts removes various model output artifacts
func stripModelArtifacts(content string) string {
	// Remove <think>...</think> blocks (including multiline)
	content = regexp.MustCompile(`(?s)<think>.*?</think>\s*`).ReplaceAllString(content, "")
	
	// Remove <tool_call>...</tool_call> blocks
	content = regexp.MustCompile(`(?s)<tool_call>.*?</tool_call>\s*`).ReplaceAllString(content, "")
	
	// Remove standalone tags
	content = regexp.MustCompile(`</?tool_call>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`<endoftext>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`<\|.*?\|>`).ReplaceAllString(content, "") // <|endoftext|> style tokens
	
	// Remove ANSI escape sequences and control characters
	content = regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`\[ðm`).ReplaceAllString(content, "")
	
	// Remove Human:/Assistant: artifacts from context bleeding
	content = regexp.MustCompile(`Human:.*?Assistant:`).ReplaceAllString(content, "")
	
	return strings.TrimSpace(content)
}

// addCodeBlockLanguage adds language hints to code blocks that don't have them
func addCodeBlockLanguage(content string) string {
	// Find all code blocks and try to detect language
	lines := strings.Split(content, "\n")
	var result []string
	inCodeBlock := false
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				// Opening code block
				inCodeBlock = true
				if trimmed == "```" {
					// No language specified, try to detect from next lines
					lang := detectLanguage(lines, i+1)
					result = append(result, "```"+lang)
					continue
				}
			} else {
				// Closing code block
				inCodeBlock = false
			}
		}
		result = append(result, line)
	}
	
	return strings.Join(result, "\n")
}

// detectLanguage tries to detect programming language from code content
func detectLanguage(lines []string, startIdx int) string {
	if startIdx >= len(lines) {
		return "text"
	}
	
	// Look at first few lines of code
	sample := ""
	for i := startIdx; i < len(lines) && i < startIdx+15; i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			break
		}
		sample += lines[i] + "\n"
	}
	sampleLower := strings.ToLower(sample)
	
	// Simple heuristics for common languages
	switch {
	// HTML detection (check first, as it can contain JS-like syntax)
	case strings.Contains(sampleLower, "<!doctype") || strings.Contains(sampleLower, "<html") ||
		strings.Contains(sampleLower, "</html>") || strings.Contains(sampleLower, "<head>") ||
		strings.Contains(sampleLower, "</head>") || strings.Contains(sampleLower, "<body>") ||
		strings.Contains(sampleLower, "</body>") || strings.Contains(sampleLower, "<title>"):
		return "html"
	// CSS detection
	case strings.Contains(sample, "{") && (strings.Contains(sample, "color:") ||
		strings.Contains(sample, "background:") || strings.Contains(sample, "margin:") ||
		strings.Contains(sample, "padding:") || strings.Contains(sample, "font-")):
		return "css"
	// React/JSX detection
	case strings.Contains(sample, "import React") || strings.Contains(sample, "from 'react'") ||
		strings.Contains(sample, "useState") || strings.Contains(sample, "useEffect") ||
		strings.Contains(sample, "className={") || strings.Contains(sample, "export default"):
		return "jsx"
	// TypeScript detection
	case strings.Contains(sample, "interface ") && strings.Contains(sample, ": ") ||
		strings.Contains(sample, ": string") || strings.Contains(sample, ": number") ||
		strings.Contains(sample, ": boolean"):
		return "typescript"
	// JavaScript detection
	case strings.Contains(sample, "function ") || strings.Contains(sample, "const ") ||
		strings.Contains(sample, "let ") || strings.Contains(sample, "=> {") ||
		strings.Contains(sample, "document.") || strings.Contains(sample, "console.log"):
		return "javascript"
	// Python detection
	case strings.Contains(sample, "def ") || strings.Contains(sample, "import ") ||
		(strings.Contains(sample, "class ") && strings.Contains(sample, ":")):
		return "python"
	// Go detection
	case strings.Contains(sample, "func ") || strings.Contains(sample, "package "):
		return "go"
	// SQL detection
	case strings.Contains(sampleLower, "select ") && strings.Contains(sampleLower, "from ") ||
		strings.Contains(sampleLower, "insert into") || strings.Contains(sampleLower, "create table"):
		return "sql"
	// JSON detection
	case (strings.HasPrefix(strings.TrimSpace(sample), "{") || strings.HasPrefix(strings.TrimSpace(sample), "[")) &&
		strings.Contains(sample, "\""):
		return "json"
	// YAML detection
	case strings.Contains(sample, ": ") && !strings.Contains(sample, "{") &&
		(strings.Contains(sample, "- ") || strings.HasPrefix(strings.TrimSpace(sample), "name:")):
		return "yaml"
	// Bash/Shell detection
	case strings.HasPrefix(strings.TrimSpace(sample), "#!/") ||
		strings.HasPrefix(strings.TrimSpace(sample), "$ ") ||
		strings.Contains(sample, "echo ") || strings.Contains(sample, "export "):
		return "bash"
	default:
		return "text"
	}
}

// renderMessages formats messages for display
func renderMessages(messages []chat.Message, mdRenderer *glamour.TermRenderer) string {
	if len(messages) == 0 {
		return statusMutedStyle.Render("No messages yet. Type something to start chatting.")
	}

	var b strings.Builder
	for i, msg := range messages {
		if msg.Role == "user" {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Bold(true)
			b.WriteString(style.Render("You: "))
			b.WriteString(msg.Content)
		} else {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
			b.WriteString(style.Render("AI: "))
			// Strip model artifacts, add language hints, and render markdown
			cleanContent := stripModelArtifacts(msg.Content)
			cleanContent = addCodeBlockLanguage(cleanContent)
			if mdRenderer != nil {
				rendered, err := mdRenderer.Render(cleanContent)
				if err == nil {
					b.WriteString(strings.TrimSpace(rendered))
				} else {
					b.WriteString(cleanContent)
				}
			} else {
				b.WriteString(cleanContent)
			}
		}

		if i < len(messages)-1 {
			b.WriteString("\n\n")
		}
	}

	return b.String()
}
