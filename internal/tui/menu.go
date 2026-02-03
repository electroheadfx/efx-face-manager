package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarques/efx-face-manager/internal/config"
	"github.com/lmarques/efx-face-manager/internal/model"
)

// Menu items - 2x2 grid + bottom row
const (
	menuRunTemplate = iota
	menuRunInstalled
	menuInstall
	menuUninstall
	menuConfigStorage
	menuExit
)

// Grid layout: row 0 = [template, model], row 1 = [install, remove], row 2 = [setup, exit]
var menuGrid = [][]int{
	{menuRunTemplate, menuRunInstalled},
	{menuInstall, menuUninstall},
	{menuConfigStorage, menuExit},
}

var menuItems = []string{
	"⚡ Run a template",
	"▶  Run a model",
	"📦 Install a model",
	"🗑  Remove a model",
	"⚙  Setup model path",
	"✖  Exit",
}

// menuModel handles the main menu
type menuModel struct {
	row         int // current row in grid
	col         int // current column in grid
	width       int
	height      int
	cfg         *config.Config
	store       *model.Store
	modelCount  int
	serverCount int
}

func newMenuModel(cfg *config.Config, store *model.Store) menuModel {
	return menuModel{
		row:        0,
		col:        0,
		cfg:        cfg,
		store:      store,
		modelCount: store.Count(),
	}
}

func (m menuModel) Init() tea.Cmd {
	return nil
}

func (m menuModel) getSelectedIndex() int {
	if m.row < len(menuGrid) && m.col < len(menuGrid[m.row]) {
		return menuGrid[m.row][m.col]
	}
	return 0
}

func (m menuModel) Update(msg tea.Msg) (menuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.row > 0 {
				m.row--
				// Clamp column to row length
				if m.col >= len(menuGrid[m.row]) {
					m.col = len(menuGrid[m.row]) - 1
				}
			}
		case "down", "j":
			if m.row < len(menuGrid)-1 {
				m.row++
				// Clamp column to row length
				if m.col >= len(menuGrid[m.row]) {
					m.col = len(menuGrid[m.row]) - 1
				}
			}
		case "left", "h":
			if m.col > 0 {
				m.col--
			}
		case "right", "l":
			if m.col < len(menuGrid[m.row])-1 {
				m.col++
			}
		case "enter":
			selected := m.getSelectedIndex()
			switch selected {
			case menuRunTemplate:
				return m, func() tea.Msg { return openTemplatesMsg{} }
			case menuRunInstalled:
				return m, func() tea.Msg { return openModelsMsg{} }
			case menuInstall:
				return m, func() tea.Msg { return openInstallMsg{} }
			case menuUninstall:
				return m, func() tea.Msg { return openUninstallMsg{} }
			case menuConfigStorage:
				return m, func() tea.Msg { return openStorageConfigMsg{} }
			case menuExit:
				return m, tea.Quit
			}
		case "s":
			// Quick access to server manager
			return m, func() tea.Msg { return openServerManagerMsg{} }
		}
	}
	return m, nil
}

func (m menuModel) View() string {
	contentWidth := getContentWidth(m.width)
	colWidth := (contentWidth - 4) / 2
	var b strings.Builder

	// Header (80% width)
	b.WriteString(renderHeader(version, m.width))
	b.WriteString("\n\n")

	// Render menu as 2-column grid with spacing
	for rowIdx, rowItems := range menuGrid {
		var rowStr strings.Builder
		for colIdx, itemIdx := range rowItems {
			item := menuItems[itemIdx]
			isSelected := rowIdx == m.row && colIdx == m.col

			style := menuItemStyle.Width(colWidth)
			prefix := "  "
			if isSelected {
				style = menuItemSelectedStyle.Width(colWidth)
				prefix = "> "
			}
			rowStr.WriteString(style.Render(prefix + item))
		}
		b.WriteString(rowStr.String() + "\n")
		// Add spacing after each row
		if rowIdx < 2 {
			b.WriteString("\n")
		}
		// Extra spacing before setup/exit row
		if rowIdx == 1 {
			b.WriteString("\n")
		}
	}

	// Status info
	b.WriteString("\n")
	modelDir := config.DisplayPath(m.cfg.ModelDir)
	storageStatus := "Local"
	if config.IsExternalMounted() && m.cfg.ModelDir == config.ExternalModelPath {
		storageStatus = "External"
	}

	b.WriteString(infoLineStyle.Render(fmt.Sprintf("Models: %s (%d installed)", modelDir, m.modelCount)))
	b.WriteString("\n")
	if m.serverCount > 0 {
		b.WriteString(infoLineStyle.Render(fmt.Sprintf("Servers: %d running", m.serverCount)))
		b.WriteString("\n")
	}
	b.WriteString(infoLineStyle.Render(fmt.Sprintf("Storage: %s", storageStatus)))

	// Calculate padding to push footer to bottom
	content := b.String()
	contentLines := strings.Count(content, "\n") + 1
	padding := calculatePadding(contentLines, 1, m.height)
	b.WriteString(strings.Repeat("\n", padding))

	// Footer
	helpText := "[↵] select  [←→↑↓] navigate  [s] servers  [q] quit"
	b.WriteString("\n" + helpStyle.Render(helpText))

	return appStyle.Render(b.String())
}
