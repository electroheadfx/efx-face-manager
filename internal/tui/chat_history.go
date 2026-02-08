package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarques/efx-face-manager/internal/chat"
)

// Message types for chat history
type loadChatMsg struct {
	conversation *chat.Conversation
}

type chatHistoryUpdatedMsg struct{}

// chatHistoryModel handles the chat history management view
type chatHistoryModel struct {
	storage       *chat.Storage
	conversations []*chat.Conversation
	selectedIdx   int
	currentPage   int
	itemsPerPage  int
	width         int
	height        int

	// For rename mode
	renaming   bool
	renameInput textinput.Model

	// For delete confirmation
	confirmDelete bool

	// Return context
	returnHost      string
	returnPort      int
	returnModelName string
}

func newChatHistoryModel(storage *chat.Storage, host string, port int, modelName string, width, height int) chatHistoryModel {
	convs, _ := storage.List()

	ti := textinput.New()
	ti.Placeholder = "New title..."
	ti.CharLimit = 100
	ti.Width = 40

	// Calculate items per page based on height
	itemsPerPage := height - 14
	if itemsPerPage < 5 {
		itemsPerPage = 5
	}

	return chatHistoryModel{
		storage:         storage,
		conversations:   convs,
		width:           width,
		height:          height,
		itemsPerPage:    itemsPerPage,
		renameInput:     ti,
		returnHost:      host,
		returnPort:      port,
		returnModelName: modelName,
	}
}

func (m chatHistoryModel) Init() tea.Cmd {
	return nil
}

func (m chatHistoryModel) Update(msg tea.Msg) (chatHistoryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case chatHistoryUpdatedMsg:
		// Refresh the list
		m.conversations, _ = m.storage.List()
		return m, nil

	case tea.KeyMsg:
		// Handle rename mode
		if m.renaming {
			switch msg.String() {
			case "enter":
				if len(m.conversations) > 0 && m.selectedIdx < len(m.conversations) {
					newTitle := strings.TrimSpace(m.renameInput.Value())
					if newTitle != "" {
						m.storage.Rename(m.conversations[m.selectedIdx].ID, newTitle)
						m.conversations, _ = m.storage.List()
					}
				}
				m.renaming = false
				m.renameInput.Blur()
				m.renameInput.Reset()
				return m, nil
			case "esc":
				m.renaming = false
				m.renameInput.Blur()
				m.renameInput.Reset()
				return m, nil
			default:
				var cmd tea.Cmd
				m.renameInput, cmd = m.renameInput.Update(msg)
				return m, cmd
			}
		}

		// Handle delete confirmation
		if m.confirmDelete {
			switch msg.String() {
			case "y", "Y":
				if len(m.conversations) > 0 && m.selectedIdx < len(m.conversations) {
					m.storage.Delete(m.conversations[m.selectedIdx].ID)
					m.conversations, _ = m.storage.List()
					if m.selectedIdx >= len(m.conversations) && m.selectedIdx > 0 {
						m.selectedIdx--
					}
				}
				m.confirmDelete = false
				return m, nil
			case "n", "N", "esc":
				m.confirmDelete = false
				return m, nil
			}
			return m, nil
		}

		// Normal mode
		switch msg.String() {
		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
				// Update page if needed
				m.currentPage = m.selectedIdx / m.itemsPerPage
			}
		case "down", "j":
			if m.selectedIdx < len(m.conversations)-1 {
				m.selectedIdx++
				// Update page if needed
				m.currentPage = m.selectedIdx / m.itemsPerPage
			}
		case "left", "h":
			// Previous page
			if m.currentPage > 0 {
				m.currentPage--
				m.selectedIdx = m.currentPage * m.itemsPerPage
			}
		case "right", "l":
			// Next page
			totalPages := (len(m.conversations) + m.itemsPerPage - 1) / m.itemsPerPage
			if m.currentPage < totalPages-1 {
				m.currentPage++
				m.selectedIdx = m.currentPage * m.itemsPerPage
			}
		case "enter":
			// Load selected conversation
			if len(m.conversations) > 0 && m.selectedIdx < len(m.conversations) {
				return m, func() tea.Msg {
					return loadChatMsg{conversation: m.conversations[m.selectedIdx]}
				}
			}
		case "r":
			// Rename
			if len(m.conversations) > 0 && m.selectedIdx < len(m.conversations) {
				m.renaming = true
				m.renameInput.SetValue(m.conversations[m.selectedIdx].Title)
				m.renameInput.Focus()
				return m, textinput.Blink
			}
		case "d":
			// Delete (ask confirmation)
			if len(m.conversations) > 0 {
				m.confirmDelete = true
			}
		case "n":
			// New chat - return to chat with new conversation
			return m, func() tea.Msg {
				return loadChatMsg{conversation: nil} // nil means create new
			}
		case "esc", "q":
			// Go back to chat
			return m, func() tea.Msg { return goBackMsg{} }
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.itemsPerPage = msg.Height - 14
		if m.itemsPerPage < 5 {
			m.itemsPerPage = 5
		}
		// Recalculate current page
		if len(m.conversations) > 0 {
			m.currentPage = m.selectedIdx / m.itemsPerPage
		}
	}

	return m, nil
}

func (m chatHistoryModel) View() string {
	var b strings.Builder

	b.WriteString("\n\n")
	b.WriteString(subtitleStyle.Render("Chat History"))
	b.WriteString("\n\n")

	if len(m.conversations) == 0 {
		b.WriteString(statusMutedStyle.Render("No saved conversations"))
		b.WriteString("\n\n")
	} else {
		// Calculate pagination
		totalPages := (len(m.conversations) + m.itemsPerPage - 1) / m.itemsPerPage
		startIdx := m.currentPage * m.itemsPerPage
		endIdx := startIdx + m.itemsPerPage
		if endIdx > len(m.conversations) {
			endIdx = len(m.conversations)
		}

		// List conversations for current page
		for i := startIdx; i < endIdx; i++ {
			conv := m.conversations[i]
			
			// Format: Title | Model | Messages | Date
			msgCount := len(conv.Messages)
			dateStr := conv.UpdatedAt.Format("2006-01-02 15:04")
			
			title := conv.Title
			if len(title) > 35 {
				title = title[:32] + "..."
			}
			
			model := conv.Model
			if len(model) > 20 {
				model = model[:17] + "..."
			}

			line := fmt.Sprintf("%-35s  %-20s  %3d msgs  %s", title, model, msgCount, dateStr)

			if i == m.selectedIdx {
				if m.renaming {
					// Show rename input
					b.WriteString(optionSelectedStyle.Render("> "))
					b.WriteString(m.renameInput.View())
				} else if m.confirmDelete {
					b.WriteString(errorStyle.Render(fmt.Sprintf("> %s  [Delete? y/n]", title)))
				} else {
					b.WriteString(optionSelectedStyle.Render(fmt.Sprintf("> %s", line)))
				}
			} else {
				b.WriteString(optionNormalStyle.Render(fmt.Sprintf("  %s", line)))
			}
			b.WriteString("\n")
		}

		// Pagination bullets
		if totalPages > 1 {
			b.WriteString("\n")
			var bullets strings.Builder
			for i := 0; i < totalPages; i++ {
				if i == m.currentPage {
					bullets.WriteString("●")
				} else {
					bullets.WriteString("○")
				}
				if i < totalPages-1 {
					bullets.WriteString(" ")
				}
			}
			b.WriteString(helpStyle.Render(fmt.Sprintf("  Page %d/%d  %s", m.currentPage+1, totalPages, bullets.String())))
		}
	}

	// Help
	b.WriteString("\n")
	if m.renaming {
		b.WriteString(helpStyle.Render("[enter] save  [esc] cancel"))
	} else if m.confirmDelete {
		b.WriteString(helpStyle.Render("[y] yes  [n] no"))
	} else {
		b.WriteString(helpStyle.Render("[enter] load  [r] rename  [d] delete  [n] new  [←/→] page  [esc] back"))
	}

	return appStyle.Render(b.String())
}
