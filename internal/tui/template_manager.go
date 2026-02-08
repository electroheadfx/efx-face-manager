package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarques/efx-face-manager/internal/config"
	"github.com/lmarques/efx-face-manager/internal/model"
)

// templateManagerModel handles template management view
type templateManagerModel struct {
	templates    []model.Template
	selected     int
	width        int
	height       int
	cfg          *config.Config
	store        *model.Store
	confirmDelete bool
	err          error
}

func newTemplateManagerModel(cfg *config.Config, store *model.Store) templateManagerModel {
	templates, _ := model.LoadTemplates()
	return templateManagerModel{
		templates: templates,
		selected:  0,
		cfg:       cfg,
		store:     store,
	}
}

func (m templateManagerModel) Init() tea.Cmd {
	return nil
}

func (m templateManagerModel) Update(msg tea.Msg) (templateManagerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle confirmation dialog
		if m.confirmDelete {
			switch msg.String() {
			case "y", "Y":
				// Delete the template
				if m.selected < len(m.templates) {
					templateName := m.templates[m.selected].Name
					if err := model.DeleteTemplate(templateName); err != nil {
						m.err = err
					}
					// Reload templates
					m.templates, _ = model.LoadTemplates()
					// Adjust selection if needed
					if m.selected >= len(m.templates) && m.selected > 0 {
						m.selected--
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

		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.templates)-1 {
				m.selected++
			}
		case "d":
			// Delete selected template (show confirmation)
			if len(m.templates) > 0 && m.selected < len(m.templates) {
				m.confirmDelete = true
			}
		case "n":
			// TODO: Create new template (could navigate to a form)
			// For now, this is a placeholder
			return m, nil
		case "esc", "m":
			return m, func() tea.Msg { return goBackMsg{} }
		}
	}
	return m, nil
}

func (m templateManagerModel) View() string {
	contentWidth := getContentWidth(m.width)
	var b strings.Builder

	// Header
	b.WriteString(renderHeader(version, m.width))
	b.WriteString("\n\n")

	// Section title
	b.WriteString(subtitleStyle.Render("Manage Templates"))
	b.WriteString("\n")
	b.WriteString(infoLineStyle.Render("~/.config/efx-face-manager/templates.yaml"))
	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render(strings.Repeat("-", contentWidth-4)))
	b.WriteString("\n")

	// Confirmation dialog
	if m.confirmDelete && m.selected < len(m.templates) {
		templateName := m.templates[m.selected].Name
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Delete template '%s'? [y/n]", templateName)))
		b.WriteString("\n\n")
	}

	// Error display
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	// Calculate column widths
	totalWidth := contentWidth - 8
	col1Width := totalWidth * 60 / 100 // 60% for name

	// Template list
	if len(m.templates) == 0 {
		b.WriteString(statusMutedStyle.Render("  No templates configured"))
		b.WriteString("\n")
	} else {
		for i, t := range m.templates {
			installed := " "
			if m.store.Exists(t.ModelName) {
				installed = "+"
			}

			line := fmt.Sprintf("%s %-*s [%s]",
				installed,
				col1Width, truncateStr(t.Name, col1Width),
				t.ModelType)

			if i == m.selected {
				b.WriteString(menuItemSelectedStyle.Width(contentWidth - 4).Render("> " + line) + "\n")
			} else {
				b.WriteString(menuItemStyle.Render("  " + line) + "\n")
			}
		}
	}

	// Calculate padding to push footer to bottom
	content := b.String()
	contentLines := strings.Count(content, "\n") + 1
	padding := calculatePadding(contentLines, 1, m.height)
	b.WriteString(strings.Repeat("\n", padding))

	// Footer
	var helpText string
	if m.confirmDelete {
		helpText = "[y] confirm delete  [n] cancel"
	} else {
		helpText = "[d] delete  [esc] back  [q] home"
	}
	b.WriteString("\n" + helpStyle.Render(helpText))

	return appStyle.Render(b.String())
}
