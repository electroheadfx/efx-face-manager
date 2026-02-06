package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarques/efx-face-manager/internal/chat"
	"github.com/lmarques/efx-face-manager/internal/server"
)

// chatModel handles the chat interface
type chatModel struct {
	// Connection
	port       int
	serverName string
	servers    *server.Manager
	client     *chat.Client

	// Conversation
	conversation *chat.Conversation
	messages     []chat.Message

	// Input
	inputBuffer string
	inputMode   bool

	// Streaming
	streaming    bool
	streamBuffer string
	contentCh    <-chan string
	errCh        <-chan error
	cancelStream func()

	// Spinner for loading
	spinner spinner.Model

	// View toggle (chat vs logs)
	showLogs     bool
	logsViewport viewport.Model

	// Markdown renderer
	glamourRenderer *glamour.TermRenderer

	// UI
	chatViewport viewport.Model
	width        int
	height       int
	err          error

	// Title editing
	editingTitle bool
	titleBuffer  string
}

// Message types for chat
type openChatMsg struct {
	port           int
	conversationID string // Optional: load specific conversation
}
type chatStreamChunkMsg struct{ content string }
type chatStreamCompleteMsg struct{}
type chatStreamErrorMsg struct{ err error }
type chatTitleGeneratedMsg struct{ title string }

func newChatModel(port int, serverName string, servers *server.Manager, conversationID string, width, height int) chatModel {
	// Create glamour renderer
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-10),
	)

	// Calculate viewport dimensions
	chatViewportHeight := height - 15
	if chatViewportHeight < 5 {
		chatViewportHeight = 5
	}
	chatVP := viewport.New(width-8, chatViewportHeight)

	logsViewportHeight := height - 10
	if logsViewportHeight < 5 {
		logsViewportHeight = 5
	}
	logsVP := viewport.New(width-8, logsViewportHeight)

	// Create spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(primary)

	m := chatModel{
		port:            port,
		serverName:      serverName,
		servers:         servers,
		client:          chat.NewClient(port),
		glamourRenderer: renderer,
		chatViewport:    chatVP,
		logsViewport:    logsVP,
		spinner:         sp,
		width:           width,
		height:          height,
		inputMode:       true, // Start in input mode
	}

	// Load or create conversation
	if conversationID != "" {
		// Load specific conversation
		if conv, err := chat.LoadConversation(conversationID); err == nil {
			m.conversation = conv
			m.messages = conv.Messages
		}
	}

	if m.conversation == nil {
		// Try to load latest conversation for this port
		if conv, err := chat.GetLatestConversation(port); err == nil && conv != nil {
			m.conversation = conv
			m.messages = conv.Messages
		} else {
			// Create new conversation
			if conv, err := chat.NewConversation(port, serverName); err == nil {
				m.conversation = conv
				m.messages = []chat.Message{}
			}
		}
	}

	// Update chat viewport content
	m.updateChatViewport()

	// Update logs viewport
	if inst := servers.Get(port); inst != nil {
		m.logsViewport.SetContent(servers.GetLogs(port))
		m.logsViewport.GotoBottom()
	}

	return m
}

func (m chatModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m chatModel) Update(msg tea.Msg) (chatModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.streaming {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case serverUpdateMsg:
		// Update logs viewport if showing logs
		if msg.Port == m.port {
			m.logsViewport.SetContent(m.servers.GetLogs(m.port))
			if m.showLogs {
				m.logsViewport.GotoBottom()
			}
		}
		return m, nil

	case chatStreamChunkMsg:
		m.streamBuffer += msg.content
		m.updateChatViewport()
		m.chatViewport.GotoBottom()
		// Continue listening for more chunks
		return m, m.listenForChunks()

	case chatStreamCompleteMsg:
		// Save the complete assistant message
		if m.streamBuffer != "" {
			m.messages = append(m.messages, chat.Message{
				Role:    "assistant",
				Content: m.streamBuffer,
			})
			m.conversation.Messages = m.messages
			chat.SaveConversation(m.conversation)

			// Generate title if this is the first exchange
			if len(m.messages) == 2 && m.conversation.Title == "New Conversation" {
				cmds = append(cmds, m.generateTitle())
			}
		}
		m.streamBuffer = ""
		m.streaming = false
		m.contentCh = nil
		m.errCh = nil
		m.cancelStream = nil
		m.updateChatViewport()
		return m, tea.Batch(cmds...)

	case chatStreamErrorMsg:
		m.err = msg.err
		m.streaming = false
		if m.streamBuffer != "" {
			// Save partial response
			m.messages = append(m.messages, chat.Message{
				Role:    "assistant",
				Content: m.streamBuffer + "\n\n[Stream interrupted]",
			})
			m.conversation.Messages = m.messages
			chat.SaveConversation(m.conversation)
		}
		m.streamBuffer = ""
		m.contentCh = nil
		m.errCh = nil
		m.cancelStream = nil
		m.updateChatViewport()
		return m, nil

	case chatTitleGeneratedMsg:
		m.conversation.Title = msg.title
		chat.SaveConversation(m.conversation)
		return m, nil

	case tea.KeyMsg:
		// Handle title editing mode first
		if m.editingTitle {
			switch msg.String() {
			case "enter":
				if m.titleBuffer != "" {
					m.conversation.Title = m.titleBuffer
					chat.SaveConversation(m.conversation)
				}
				m.editingTitle = false
				m.titleBuffer = ""
			case "esc":
				m.editingTitle = false
				m.titleBuffer = ""
			case "backspace":
				if len(m.titleBuffer) > 0 {
					m.titleBuffer = m.titleBuffer[:len(m.titleBuffer)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.titleBuffer += msg.String()
				}
			}
			return m, nil
		}

		// Handle input mode - capture ALL characters first
		if m.inputMode {
			switch msg.String() {
			case "enter":
				if !m.streaming && m.inputBuffer != "" {
					return m, m.sendMessage()
				}
			case "esc":
				m.inputMode = false
			case "backspace":
				if len(m.inputBuffer) > 0 {
					m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
				}
			case "tab":
				// Allow tab to switch views even in input mode
				m.showLogs = !m.showLogs
				if m.showLogs {
					m.logsViewport.SetContent(m.servers.GetLogs(m.port))
					m.logsViewport.GotoBottom()
				}
			default:
				// Capture ALL single characters including 'i', 'n', 't', etc.
				if len(msg.String()) == 1 {
					m.inputBuffer += msg.String()
				}
			}
			return m, nil
		}

		// Not in input mode - handle shortcuts
		switch msg.String() {
		case "tab":
			m.showLogs = !m.showLogs
			if m.showLogs {
				m.logsViewport.SetContent(m.servers.GetLogs(m.port))
				m.logsViewport.GotoBottom()
			}
			return m, nil

		case "n":
			// New conversation (only if not streaming)
			if !m.streaming {
				if conv, err := chat.NewConversation(m.port, m.serverName); err == nil {
					m.conversation = conv
					m.messages = []chat.Message{}
					m.updateChatViewport()
				}
			}
			return m, nil

		case "t":
			// Edit title (only if not streaming)
			if !m.streaming {
				m.editingTitle = true
				m.titleBuffer = m.conversation.Title
			}
			return m, nil

		case "i":
			// Enter input mode
			if !m.showLogs {
				m.inputMode = true
			}
			return m, nil

		case "esc":
			if m.showLogs {
				m.showLogs = false
			} else {
				return m, func() tea.Msg { return goBackMsg{} }
			}
			return m, nil

		case "up", "k":
			if m.showLogs {
				m.logsViewport, cmd = m.logsViewport.Update(msg)
			} else {
				m.chatViewport, cmd = m.chatViewport.Update(msg)
			}
			return m, cmd

		case "down", "j":
			if m.showLogs {
				m.logsViewport, cmd = m.logsViewport.Update(msg)
			} else {
				m.chatViewport, cmd = m.chatViewport.Update(msg)
			}
			return m, cmd

		case "g":
			if m.showLogs {
				m.logsViewport.GotoTop()
			} else {
				m.chatViewport.GotoTop()
			}
			return m, nil

		case "G":
			if m.showLogs {
				m.logsViewport.GotoBottom()
			} else {
				m.chatViewport.GotoBottom()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Recreate glamour renderer with new width
		renderer, _ := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(msg.Width-10),
		)
		m.glamourRenderer = renderer

		// Update viewports
		chatViewportHeight := msg.Height - 15
		if chatViewportHeight < 5 {
			chatViewportHeight = 5
		}
		m.chatViewport.Width = msg.Width - 8
		m.chatViewport.Height = chatViewportHeight

		logsViewportHeight := msg.Height - 10
		if logsViewportHeight < 5 {
			logsViewportHeight = 5
		}
		m.logsViewport.Width = msg.Width - 8
		m.logsViewport.Height = logsViewportHeight

		m.updateChatViewport()
	}

	return m, tea.Batch(cmds...)
}

func (m chatModel) View() string {
	contentWidth := m.width - 4
	var b strings.Builder

	b.WriteString("\n\n\n")

	// Header with tab indicators
	chatTab := "Chat"
	logsTab := "Logs"
	if m.showLogs {
		chatTab = " Chat "
		logsTab = "[Logs]"
	} else {
		chatTab = "[Chat]"
		logsTab = " Logs "
	}

	title := m.conversation.Title
	if m.editingTitle {
		title = m.titleBuffer + "_"
	}

	header := fmt.Sprintf("%s %s  |  %s  :%d", chatTab, logsTab, title, m.port)
	b.WriteString(subtitleStyle.Render(header))
	b.WriteString("\n")

	if m.showLogs {
		b.WriteString(m.renderLogsView(contentWidth))
	} else {
		b.WriteString(m.renderChatView(contentWidth))
	}

	return appStyle.Render(b.String())
}

func (m *chatModel) renderChatView(contentWidth int) string {
	var b strings.Builder

	// Chat viewport
	chatPanelHeight := m.height - 15
	if chatPanelHeight < 5 {
		chatPanelHeight = 5
	}

	chatBorder := primary
	chatPanel := lipgloss.NewStyle().
		Width(contentWidth-4).
		Height(chatPanelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(chatBorder).
		Padding(0, 1).
		Render(m.chatViewport.View())

	b.WriteString(chatPanel)
	b.WriteString("\n")

	// Input area
	inputBorder := muted
	if m.inputMode {
		inputBorder = primary
	}

	inputContent := m.inputBuffer
	if m.inputMode {
		inputContent += "_"
	}
	if inputContent == "" && !m.inputMode {
		inputContent = statusMutedStyle.Render("Press 'i' to type a message...")
	}

	inputPanel := lipgloss.NewStyle().
		Width(contentWidth-4).
		Height(3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(inputBorder).
		Padding(0, 1).
		Render("> " + inputContent)

	b.WriteString(inputPanel)
	b.WriteString("\n")

	// Error display
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	// Help bar with spinner when streaming
	var helpText string
	if m.editingTitle {
		helpText = "[enter] save title  [esc] cancel"
	} else if m.streaming {
		helpText = m.spinner.View() + " Generating response..."
	} else if m.inputMode {
		helpText = "[enter] send  [esc] exit input  [tab] logs"
	} else {
		helpText = "[tab] logs  [i] input  [n] new  [t] title  [esc] back"
	}
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

func (m *chatModel) renderLogsView(contentWidth int) string {
	var b strings.Builder

	// Logs viewport
	logsPanelHeight := m.height - 10
	if logsPanelHeight < 5 {
		logsPanelHeight = 5
	}

	logsPanel := lipgloss.NewStyle().
		Width(contentWidth-4).
		Height(logsPanelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Padding(0, 1).
		Render(m.logsViewport.View())

	b.WriteString(logsPanel)
	b.WriteString("\n")

	// Scroll indicator
	b.WriteString(infoLineStyle.Render(fmt.Sprintf("%.0f%% scroll  [g/G] top/bottom", m.logsViewport.ScrollPercent()*100)))
	b.WriteString("\n")

	// Help bar
	b.WriteString(helpStyle.Render("[tab] chat  [esc] back"))

	return b.String()
}

func (m *chatModel) updateChatViewport() {
	var content strings.Builder

	if len(m.messages) == 0 && m.streamBuffer == "" && !m.streaming {
		content.WriteString(statusMutedStyle.Render("No messages yet. Type a message to start chatting."))
	} else {
		for _, msg := range m.messages {
			content.WriteString(m.formatMessage(msg))
			content.WriteString("\n")
		}

		// Show streaming content or loading indicator
		if m.streaming {
			if m.streamBuffer != "" {
				content.WriteString(m.formatStreamingMessage())
			} else {
				// Show loading indicator while waiting for first chunk
				roleStyle := lipgloss.NewStyle().Bold(true).Foreground(primary)
				content.WriteString(roleStyle.Render("Assistant:"))
				content.WriteString("\n")
				content.WriteString(m.spinner.View() + " Thinking...")
				content.WriteString("\n")
			}
		}
	}

	m.chatViewport.SetContent(content.String())
}

func (m *chatModel) formatMessage(msg chat.Message) string {
	var b strings.Builder

	roleStyle := lipgloss.NewStyle().Bold(true)
	switch msg.Role {
	case "user":
		roleStyle = roleStyle.Foreground(secondary)
		b.WriteString(roleStyle.Render("You:"))
		b.WriteString("\n")
		b.WriteString(msg.Content)
	case "assistant":
		roleStyle = roleStyle.Foreground(primary)
		b.WriteString(roleStyle.Render("Assistant:"))
		b.WriteString("\n")
		// Render markdown for assistant responses
		if m.glamourRenderer != nil {
			rendered, err := m.glamourRenderer.Render(msg.Content)
			if err == nil {
				b.WriteString(strings.TrimSpace(rendered))
			} else {
				b.WriteString(msg.Content)
			}
		} else {
			b.WriteString(msg.Content)
		}
	case "system":
		roleStyle = roleStyle.Foreground(muted)
		b.WriteString(roleStyle.Render("System:"))
		b.WriteString("\n")
		b.WriteString(msg.Content)
	}

	b.WriteString("\n")
	return b.String()
}

func (m *chatModel) formatStreamingMessage() string {
	var b strings.Builder

	roleStyle := lipgloss.NewStyle().Bold(true).Foreground(primary)
	b.WriteString(roleStyle.Render("Assistant:"))
	b.WriteString("\n")

	// Render markdown for streaming content
	if m.glamourRenderer != nil {
		rendered, err := m.glamourRenderer.Render(m.streamBuffer)
		if err == nil {
			b.WriteString(strings.TrimSpace(rendered))
		} else {
			b.WriteString(m.streamBuffer)
		}
	} else {
		b.WriteString(m.streamBuffer)
	}

	b.WriteString(" " + m.spinner.View()) // Spinner at end of streaming text
	b.WriteString("\n")
	return b.String()
}

func (m *chatModel) sendMessage() tea.Cmd {
	userMessage := strings.TrimSpace(m.inputBuffer)
	if userMessage == "" {
		return nil
	}

	// Add user message
	m.messages = append(m.messages, chat.Message{
		Role:    "user",
		Content: userMessage,
	})

	// Save conversation
	m.conversation.Messages = m.messages
	chat.SaveConversation(m.conversation)

	// Clear input and start streaming
	m.inputBuffer = ""
	m.streaming = true
	m.streamBuffer = ""
	m.err = nil
	m.updateChatViewport()
	m.chatViewport.GotoBottom()

	// Start streaming request - store channels for reuse
	contentCh, errCh, cancel := m.client.StreamChatCompletion(m.conversation.ToAPIMessages())
	m.contentCh = contentCh
	m.errCh = errCh
	m.cancelStream = cancel

	// Return commands: spinner tick + listen for first chunk
	return tea.Batch(m.spinner.Tick, m.listenForChunks())
}

func (m *chatModel) listenForChunks() tea.Cmd {
	// Use the stored channels instead of creating new request
	contentCh := m.contentCh
	errCh := m.errCh

	if contentCh == nil {
		return nil
	}

	return func() tea.Msg {
		select {
		case content, ok := <-contentCh:
			if !ok {
				return chatStreamCompleteMsg{}
			}
			return chatStreamChunkMsg{content: content}
		case err := <-errCh:
			if err != nil {
				return chatStreamErrorMsg{err: err}
			}
			return chatStreamCompleteMsg{}
		}
	}
}

func (m *chatModel) generateTitle() tea.Cmd {
	return func() tea.Msg {
		title, _ := m.client.GenerateTitle(m.conversation.ToAPIMessages())
		return chatTitleGeneratedMsg{title: title}
	}
}
