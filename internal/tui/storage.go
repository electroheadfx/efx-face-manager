package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarques/efx-face-manager/internal/config"
)

// storageModel handles storage path configuration
type storageModel struct {
	cfg      *config.Config
	selected int
	width    int
	height   int
}

func newStorageModel(cfg *config.Config) storageModel {
	return storageModel{
		cfg:      cfg,
		selected: 0,
	}
}

func (m storageModel) Init() tea.Cmd {
	return nil
}

func (m storageModel) Update(msg tea.Msg) (storageModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < 3 {
				m.selected++
			}
		case "enter":
			switch m.selected {
			case 0: // External
				if config.IsExternalMounted() {
					m.cfg.ModelDir = config.ExternalModelPath
					m.cfg.AutoDetectPath = false
					m.cfg.Save()
					return m, func() tea.Msg {
						return configSavedMsg{config: m.cfg}
					}
				}
			case 1: // Local
				m.cfg.ModelDir = config.ExpandPath(config.LocalModelPath)
				m.cfg.AutoDetectPath = false
				m.cfg.Save()
				return m, func() tea.Msg {
					return configSavedMsg{config: m.cfg}
				}
			case 2: // Auto-detect
				m.cfg.ModelDir = config.DetectDefaultPath()
				m.cfg.AutoDetectPath = true
				m.cfg.Save()
				return m, func() tea.Msg {
					return configSavedMsg{config: m.cfg}
				}
			case 3: // Back
				return m, func() tea.Msg { return goBackMsg{} }
			}
		case "esc":
			return m, func() tea.Msg { return goBackMsg{} }
		}
	}
	return m, nil
}

func (m storageModel) View() string {
	contentWidth := getContentWidth(m.width)
	var b strings.Builder

	// Header (80% width)
	b.WriteString(renderHeader(version, m.width))
	b.WriteString("\n\n")

	// Section title
	b.WriteString(subtitleStyle.Render("Configure Model Storage Path"))
	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render(strings.Repeat("─", contentWidth-4)))
	b.WriteString("\n\n")

	// Get status for each path
	externalStatus := getPathStatus(config.ExternalModelPath)
	localStatus := getPathStatus(config.ExpandPath(config.LocalModelPath))
	
	// Current marker
	externalMarker := ""
	localMarker := ""
	if m.cfg.ModelDir == config.ExternalModelPath {
		externalMarker = " ← Current"
	} else if m.cfg.ModelDir == config.ExpandPath(config.LocalModelPath) {
		localMarker = " ← Current"
	}

	// Options
	options := []string{
		fmt.Sprintf("External: %s [%s]%s", config.ExternalModelPath, externalStatus, externalMarker),
		fmt.Sprintf("Local: %s [%s]%s", config.DisplayPath(config.ExpandPath(config.LocalModelPath)), localStatus, localMarker),
		"Auto-detect (External → Local fallback)",
		"✖ Back",
	}

	for i, opt := range options {
		if i == m.selected {
			b.WriteString(menuItemSelectedStyle.Width(contentWidth - 4).Render("> " + opt) + "\n")
		} else {
			b.WriteString(menuItemStyle.Render("  " + opt) + "\n")
		}
	}

	// Calculate padding to push footer to bottom
	content := b.String()
	contentLines := strings.Count(content, "\n") + 1
	padding := calculatePadding(contentLines, 1, m.height)
	b.WriteString(strings.Repeat("\n", padding))

	// Footer
	helpText := "[↵] select  [esc] back"
	b.WriteString("\n" + helpStyle.Render(helpText))

	return appStyle.Render(b.String())
}

func getPathStatus(path string) string {
	count := countModelsInPath(path)
	
	if count > 0 {
		return fmt.Sprintf("✓ Active (%d models)", count)
	}
	
	if path == config.ExternalModelPath && !config.IsExternalMounted() {
		return "✗ External drive not mounted"
	}
	
	// Check if directory exists
	if _, err := os.Stat(path); err == nil {
		return "✓ Available (no models)"
	}
	
	return "○ Not created"
}

func countModelsInPath(baseDir string) int {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.Name() != "cache" {
			fullPath := baseDir + "/" + entry.Name()
			info, err := os.Lstat(fullPath)
			if err == nil && info.Mode()&os.ModeSymlink != 0 {
				count++
			}
		}
	}
	return count
}
