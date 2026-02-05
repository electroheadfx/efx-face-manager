package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarques/efx-face-manager/internal/config"
	"github.com/lmarques/efx-face-manager/internal/model"
	"github.com/lmarques/efx-face-manager/internal/server"
)

// templatesModel handles template selection
type templatesModel struct {
	templates []model.Template
	selected  int
	width     int
	height    int
	cfg       *config.Config
	store     *model.Store
}

func newTemplatesModel(cfg *config.Config, store *model.Store) templatesModel {
	return templatesModel{
		templates: model.DefaultTemplates(),
		selected:  0,
		cfg:       cfg,
		store:     store,
	}
}

func (m templatesModel) Init() tea.Cmd {
	return nil
}

func (m templatesModel) Update(msg tea.Msg) (templatesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.templates) {
				m.selected++
			}
		case "tab":
			// Switch to models view
			return m, func() tea.Msg { return openModelsMsg{} }
		case "enter":
			// Back option selected
			if m.selected == len(m.templates) {
				return m, func() tea.Msg { return goBackMsg{} }
			}
			if m.selected < len(m.templates) {
				template := m.templates[m.selected]
				// Check if model is installed
				if !m.store.Exists(template.ModelName) {
					// TODO: Show error or offer to install
					return m, nil
				}
				// Create config from template
				cfg := server.FromTemplate(&template, m.cfg.ModelDir)
				return m, func() tea.Msg {
					return openConfigPanelMsg{config: cfg}
				}
			}
		case "esc":
			return m, func() tea.Msg { return goBackMsg{} }
		}
	}
	return m, nil
}

func (m templatesModel) View() string {
	contentWidth := getContentWidth(m.width)
	var b strings.Builder

	// Header (80% width)
	b.WriteString(renderHeader(version, m.width))
	b.WriteString("\n\n")

	// Section title
	b.WriteString(subtitleStyle.Render("Select Template Model"))
	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render(strings.Repeat("─", contentWidth-4)))
	b.WriteString("\n\n")

	// Calculate column widths dynamically (total = contentWidth - 8 for margins/prefix)
	totalWidth := contentWidth - 8
	col1Width := totalWidth * 50 / 100  // 50% for name
	col2Width := totalWidth * 15 / 100  // 15% for type (fits "multimodal")
	col3Width := totalWidth - col1Width - col2Width  // rest for description

	// Template list
	for i, t := range m.templates {
		installed := "✗"
		if m.store.Exists(t.ModelName) {
			installed = "✓"
		}
		
		line := fmt.Sprintf("%s %-*s %-*s %s", 
			installed, 
			col1Width, truncateStr(t.Name, col1Width), 
			col2Width, t.ModelType, 
			truncateStr(t.Description, col3Width))

		if i == m.selected {
			b.WriteString(menuItemSelectedStyle.Width(contentWidth - 4).Render("> " + line) + "\n")
		} else {
			b.WriteString(menuItemStyle.Width(contentWidth - 4).Render("  " + line) + "\n")
		}
	}

	// Back option
	b.WriteString("\n")
	if m.selected == len(m.templates) {
		b.WriteString(menuItemSelectedStyle.Width(contentWidth - 4).Render("> [Back]"))
	} else {
		b.WriteString(menuItemStyle.Render("  [Back]"))
	}

	// Calculate padding to push footer to bottom
	content := b.String()
	contentLines := strings.Count(content, "\n") + 1
	padding := calculatePadding(contentLines, 1, m.height)
	b.WriteString(strings.Repeat("\n", padding))

	// Footer
	helpText := "[↵] run  [tab] models  [esc] back"
	b.WriteString("\n" + helpStyle.Render(helpText))

	return appStyle.Render(b.String())
}

// modelsModel handles installed model selection
type modelsModel struct {
	models   []model.Model
	selected int
	width    int
	height   int
	cfg      *config.Config
	store    *model.Store
}

func newModelsModel(cfg *config.Config, store *model.Store) modelsModel {
	models, _ := store.List()
	return modelsModel{
		models:   models,
		selected: 0,
		cfg:      cfg,
		store:    store,
	}
}

func (m modelsModel) Init() tea.Cmd {
	return nil
}

func (m modelsModel) Update(msg tea.Msg) (modelsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			maxIdx := len(m.models) // includes Back option
			if m.selected < maxIdx {
				m.selected++
			}
		case "tab":
			// Switch to templates view
			return m, func() tea.Msg { return openTemplatesMsg{} }
		case "enter":
			// Back option selected
			if m.selected == len(m.models) {
				return m, func() tea.Msg { return goBackMsg{} }
			}
			if m.selected < len(m.models) {
				modelName := m.models[m.selected].Name
				return m, func() tea.Msg {
					return openModelTypeMsg{model: modelName}
				}
			}
		case "esc":
			return m, func() tea.Msg { return goBackMsg{} }
		}
	}
	return m, nil
}

func (m modelsModel) View() string {
	contentWidth := getContentWidth(m.width)
	var b strings.Builder

	// Header (80% width)
	b.WriteString(renderHeader(version, m.width))
	b.WriteString("\n\n")

	// Section title
	b.WriteString(subtitleStyle.Render("Select Installed Model"))
	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render(strings.Repeat("─", contentWidth-4)))
	b.WriteString("\n\n")

	// Model list
	if len(m.models) == 0 {
		b.WriteString(statusMutedStyle.Render("  No models installed"))
		b.WriteString("\n")
	} else {
		for i, mdl := range m.models {
			line := mdl.Name
			if i == m.selected {
				b.WriteString(menuItemSelectedStyle.Width(contentWidth - 4).Render("> " + line) + "\n")
			} else {
				b.WriteString(menuItemStyle.Render("  " + line) + "\n")
			}
		}
	}

	// Back option
	b.WriteString("\n")
	if m.selected == len(m.models) {
		b.WriteString(menuItemSelectedStyle.Width(contentWidth - 4).Render("> [Back]"))
	} else {
		b.WriteString(menuItemStyle.Render("  [Back]"))
	}

	// Status
	b.WriteString("\n\n")
	b.WriteString(infoLineStyle.Render(fmt.Sprintf("Models: %s", config.DisplayPath(m.cfg.ModelDir))))

	// Calculate padding to push footer to bottom
	content := b.String()
	contentLines := strings.Count(content, "\n") + 1
	padding := calculatePadding(contentLines, 1, m.height)
	b.WriteString(strings.Repeat("\n", padding))

	// Footer
	helpText := "[↵] select type  [tab] templates  [esc] back"
	b.WriteString("\n" + helpStyle.Render(helpText))

	return appStyle.Render(b.String())
}

// modelTypeModel handles model type selection
type modelTypeModel struct {
	modelName string
	types     []model.ModelType
	labels    []string
	selected  int
	width     int
	height    int
	cfg       *config.Config
}

var modelTypes = []model.ModelType{
	model.TypeLM,
	model.TypeMultimodal,
	model.TypeImageGeneration,
	model.TypeImageEdit,
	model.TypeEmbeddings,
	model.TypeWhisper,
}

var modelTypeLabels = []string{
	"lm (text-only)",
	"multimodal (vision, audio)",
	"image-generation (qwen-image, q16)",
	"image-edit (qwen-image-edit, q16)",
	"embeddings",
	"whisper (audio transcription)",
}

func newModelTypeModel(modelName string, cfg *config.Config) modelTypeModel {
	return modelTypeModel{
		modelName: modelName,
		types:     modelTypes,
		labels:    modelTypeLabels,
		selected:  0,
		cfg:       cfg,
	}
}

func (m modelTypeModel) Init() tea.Cmd {
	return nil
}

func (m modelTypeModel) Update(msg tea.Msg) (modelTypeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.types) {
				m.selected++
			}
		case "enter":
			// Back option selected
			if m.selected == len(m.types) {
				return m, func() tea.Msg { return openModelsMsg{} }
			}
			if m.selected < len(m.types) {
				// Create config for selected type
				cfg := server.NewConfig()
				cfg.Model = m.modelName
				cfg.ModelPath = m.cfg.ModelDir + "/" + m.modelName
				cfg.Type = m.types[m.selected]
				
				// Apply defaults for image types
				switch cfg.Type {
				case model.TypeImageGeneration:
					cfg.ConfigName = "qwen-image"
					cfg.Quantize = 16
				case model.TypeImageEdit:
					cfg.ConfigName = "qwen-image-edit"
					cfg.Quantize = 16
				}
				
				return m, func() tea.Msg {
					return openConfigPanelMsg{config: cfg}
				}
			}
		case "esc":
			return m, func() tea.Msg { return openModelsMsg{} }
		}
	}
	return m, nil
}

func (m modelTypeModel) View() string {
	contentWidth := getContentWidth(m.width)
	var b strings.Builder

	// Header (80% width)
	b.WriteString(renderHeader(version, m.width))
	b.WriteString("\n\n")

	// Model info
	b.WriteString(subtitleStyle.Render("Model: " + m.modelName))
	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render("Select Configuration"))
	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render(strings.Repeat("─", contentWidth-4)))
	b.WriteString("\n\n")

	// Type list
	for i, label := range m.labels {
		if i == m.selected {
			b.WriteString(menuItemSelectedStyle.Width(contentWidth - 4).Render("> " + label) + "\n")
		} else {
			b.WriteString(menuItemStyle.Render("  " + label) + "\n")
		}
	}

	// Back option
	b.WriteString("\n")
	if m.selected == len(m.types) {
		b.WriteString(menuItemSelectedStyle.Width(contentWidth - 4).Render("> [Back]"))
	} else {
		b.WriteString(menuItemStyle.Render("  [Back]"))
	}

	// Calculate padding to push footer to bottom
	content := b.String()
	contentLines := strings.Count(content, "\n") + 1
	padding := calculatePadding(contentLines, 1, m.height)
	b.WriteString(strings.Repeat("\n", padding))

	// Footer
	helpText := "[↵] configure  [esc] back"
	b.WriteString("\n" + helpStyle.Render(helpText))

	return appStyle.Render(b.String())
}

// Helper to truncate strings
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
