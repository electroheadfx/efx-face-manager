package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarques/efx-face-manager/internal/config"
	"github.com/lmarques/efx-face-manager/internal/model"
)

// Menu items
const (
	menuRunTemplate = iota
	menuInstall
	menuRunInstalled
	menuUninstall
	menuServerManager
	menuConfigStorage
	menuExit
)

// Box 1: 2x2 grid layout
// Row 0: [Run template, Install]
// Row 1: [Run model, Remove]
var box1Grid = [][]int{
	{menuRunTemplate, menuInstall},
	{menuRunInstalled, menuUninstall},
}

// Box 2: 2-col grid
// Row 0: [Server Manager, Setup]
// Row 1: [Exit]
var box2Grid = [][]int{
	{menuServerManager, menuConfigStorage},
	{menuExit},
}

var menuItems = []string{
	"⚡ Run a template",
	"↓  Install a model",
	"▶  Run a model",
	"✕  Remove a model",
	"◉  Server Manager",
	"⚙  Setup model path",
	"✖  Exit",
}

// menuModel handles the main menu
type menuModel struct {
	box         int // 0 = box1, 1 = box2
	row         int // row within current box
	col         int // col within current row
	width       int
	height      int
	cfg         *config.Config
	store       *model.Store
	modelCount  int
	serverCount int
}

func newMenuModel(cfg *config.Config, store *model.Store) menuModel {
	return menuModel{
		box:        0,
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

func (m menuModel) getCurrentGrid() [][]int {
	if m.box == 0 {
		return box1Grid
	}
	return box2Grid
}

func (m menuModel) getSelectedIndex() int {
	grid := m.getCurrentGrid()
	if m.row < len(grid) && m.col < len(grid[m.row]) {
		return grid[m.row][m.col]
	}
	return 0
}

func (m menuModel) Update(msg tea.Msg) (menuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		grid := m.getCurrentGrid()
		switch msg.String() {
		case "up", "k":
			if m.row > 0 {
				m.row--
				// Clamp col to row length
				if m.col >= len(grid[m.row]) {
					m.col = len(grid[m.row]) - 1
				}
			}
		case "down", "j":
			if m.row < len(grid)-1 {
				m.row++
				// Clamp col to row length
				if m.col >= len(grid[m.row]) {
					m.col = len(grid[m.row]) - 1
				}
			}
		case "left", "h":
			if m.col > 0 {
				m.col--
			}
		case "right", "l":
			if m.col < len(grid[m.row])-1 {
				m.col++
			}
		case "tab":
			// TAB cycles within current column only
			if m.col == 0 {
				// Col 0 cycle: Run template -> Run model -> Server Manager -> Exit -> back
				currentIdx := m.getSelectedIndex()
				switch currentIdx {
				case menuRunTemplate:
					m.box = 0
					m.row = 1
					m.col = 0
				case menuRunInstalled:
					m.box = 1
					m.row = 0
					m.col = 0
				case menuServerManager:
					m.box = 1
					m.row = 1
					m.col = 0
				case menuExit:
					m.box = 0
					m.row = 0
					m.col = 0
				default:
					m.box = 0
					m.row = 0
					m.col = 0
				}
			} else {
				// Col 1 cycle: Install -> Remove -> Setup -> back
				currentIdx := m.getSelectedIndex()
				switch currentIdx {
				case menuInstall:
					m.box = 0
					m.row = 1
					m.col = 1
				case menuUninstall:
					m.box = 1
					m.row = 0
					m.col = 1
				case menuConfigStorage:
					m.box = 0
					m.row = 0
					m.col = 1
				default:
					m.box = 0
					m.row = 0
					m.col = 1
				}
			}
		case "shift+tab":
			// Reverse TAB cycles within current column only
			if m.col == 0 {
				// Col 0 reverse: Exit -> Server Manager -> Run model -> Run template
				currentIdx := m.getSelectedIndex()
				switch currentIdx {
				case menuExit:
					m.box = 1
					m.row = 0
					m.col = 0
				case menuServerManager:
					m.box = 0
					m.row = 1
					m.col = 0
				case menuRunInstalled:
					m.box = 0
					m.row = 0
					m.col = 0
				case menuRunTemplate:
					m.box = 1
					m.row = 1
					m.col = 0
				default:
					m.box = 0
					m.row = 0
					m.col = 0
				}
			} else {
				// Col 1 reverse: Setup -> Remove -> Install
				currentIdx := m.getSelectedIndex()
				switch currentIdx {
				case menuConfigStorage:
					m.box = 0
					m.row = 1
					m.col = 1
				case menuUninstall:
					m.box = 0
					m.row = 0
					m.col = 1
				case menuInstall:
					m.box = 1
					m.row = 0
					m.col = 1
				default:
					m.box = 0
					m.row = 0
					m.col = 1
				}
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
			case menuServerManager:
				return m, func() tea.Msg { return openServerManagerMsg{} }
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
	var b strings.Builder

	// Big header for main menu only
	b.WriteString(titleStyle.Render(asciiHeaderBig))
	b.WriteString("\n")
	
	// Spaced subtitle first (on top)
	spacedSubtitle := "M L X   H u g g i n g   F a c e   M a n a g e r"
	b.WriteString(infoLineStyle.Render(spacedSubtitle))
	b.WriteString("\n")
	
	// Version line below with leading spaces
	versionLine := "                                                    v" + version + " - Efx"
	b.WriteString(infoLineStyle.Render(versionLine))
	b.WriteString("\n\n")

	// Box styles
	boxWidth := contentWidth - 8
	colWidth := (boxWidth - 6) / 2
	
	activeBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Padding(0, 2).
		Width(boxWidth)
	
	inactiveBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(muted).
		Padding(0, 2).
		Width(boxWidth)

	// Render Box 1 as 2-col grid
	var box1Content strings.Builder
	for rowIdx, rowItems := range box1Grid {
		var rowStr strings.Builder
		for colIdx, itemIdx := range rowItems {
			item := menuItems[itemIdx]
			isSelected := m.box == 0 && rowIdx == m.row && colIdx == m.col
			
			style := optionNormalStyle.Width(colWidth)
			prefix := "  "
			if isSelected {
				style = optionSelectedStyle.Width(colWidth)
				prefix = "> "
			}
			rowStr.WriteString(style.Render(prefix + item))
		}
		box1Content.WriteString(rowStr.String())
		if rowIdx < len(box1Grid)-1 {
			box1Content.WriteString("\n")
		}
	}
	
	box1Style := inactiveBoxStyle
	if m.box == 0 {
		box1Style = activeBoxStyle
	}
	b.WriteString(box1Style.Render(box1Content.String()))
	b.WriteString("\n")

	// Render Box 2 as 2-col grid
	var box2Content strings.Builder
	for rowIdx, rowItems := range box2Grid {
		var rowStr strings.Builder
		for colIdx, itemIdx := range rowItems {
			item := menuItems[itemIdx]
			isSelected := m.box == 1 && rowIdx == m.row && colIdx == m.col
			
			style := optionNormalStyle.Width(colWidth)
			prefix := "  "
			if isSelected {
				style = optionSelectedStyle.Width(colWidth)
				prefix = "> "
			}
			rowStr.WriteString(style.Render(prefix + item))
		}
		box2Content.WriteString(rowStr.String())
		if rowIdx < len(box2Grid)-1 {
			box2Content.WriteString("\n")
		}
	}
	
	box2Style := inactiveBoxStyle
	if m.box == 1 {
		box2Style = activeBoxStyle
	}
	b.WriteString(box2Style.Render(box2Content.String()))

	// Status info
	b.WriteString("\n\n")
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
	helpText := "[↵] select  [←→↑↓] navigate  [tab] cycle column  [s] servers  [q] quit"
	b.WriteString("\n" + helpStyle.Render(helpText))

	return appStyle.Render(b.String())
}
