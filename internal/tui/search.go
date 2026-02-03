package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarques/efx-face-manager/internal/config"
	"github.com/lmarques/efx-face-manager/internal/hf"
	"github.com/lmarques/efx-face-manager/internal/model"
)

// HuggingFace sources
var hfSources = []string{
	"mlx-community",
	"lmstudio-community",
	"All Models",
}

// searchModel handles HuggingFace model search/install with pre-load + filter UX
type searchModel struct {
	cfg          *config.Config
	store        *model.Store
	hfClient     *hf.Client
	width        int
	height       int
	sourceIdx    int
	allModels    []hf.Model // All loaded models
	filtered     []hf.Model // Filtered models
	cursor       int
	currentPage  int
	filter       string
	filtering    bool // In filter input mode
	loading      bool
	loadedSource int  // Track which source is loaded
	err          error
	installing   bool
	installMsg   string
	spinner      spinner.Model
}

func newSearchModel(cfg *config.Config, store *model.Store) searchModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return searchModel{
		cfg:          cfg,
		store:        store,
		hfClient:     hf.NewClient(),
		sourceIdx:    0,
		loadedSource: -1, // Not loaded yet
		spinner:      s,
	}
}

func (m searchModel) Init() tea.Cmd {
	// Auto-load models from first source on init
	return tea.Batch(m.spinner.Tick, m.loadModels(0))
}

// Message types for search
type modelsLoadedMsg struct {
	models    []hf.Model
	sourceIdx int
}

type loadErrorMsg struct {
	err error
}

type installCompleteMsg struct {
	modelID string
	err     error
}

// openDetailsMsg is sent to open model details view
type openDetailsMsg struct {
	model hf.Model
}

func (m searchModel) Update(msg tea.Msg) (searchModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.loading || m.installing {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case modelsLoadedMsg:
		m.loading = false
		m.loadedSource = msg.sourceIdx
		m.allModels = msg.models
		m.applyFilter()

	case loadErrorMsg:
		m.loading = false
		m.err = msg.err

	case installCompleteMsg:
		m.installing = false
		if msg.err != nil {
			m.err = msg.err
			m.installMsg = fmt.Sprintf("Install failed: %v", msg.err)
		} else {
			m.installMsg = fmt.Sprintf("Successfully installed %s", msg.modelID)
			// Refresh store
			m.store = model.NewStore(m.cfg.ModelDir)
		}

	case tea.KeyMsg:
		if m.installing {
			return m, nil
		}

		// Filter input mode
		if m.filtering {
			switch msg.String() {
			case "enter":
				m.filtering = false
				m.cursor = 0
				return m, nil
			case "esc":
				m.filtering = false
				m.filter = ""
				m.applyFilter()
				return m, nil
			case "backspace":
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.applyFilter()
				}
				return m, nil
			default:
				if len(msg.String()) == 1 {
					m.filter += msg.String()
					m.applyFilter()
				}
				return m, nil
			}
		}

		// Normal mode
		switch msg.String() {
		case "?", "/":
			// Enter filter mode
			m.filtering = true
			return m, nil

		case "esc":
			if m.filter != "" {
				m.filter = ""
				m.applyFilter()
				return m, nil
			}
			return m, func() tea.Msg { return goBackMsg{} }

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.currentPage = m.cursor / m.getItemsPerPage()
			}

		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				m.currentPage = m.cursor / m.getItemsPerPage()
			}

		case "left", "h":
			// Previous page
			if m.currentPage > 0 {
				m.currentPage--
				m.cursor = m.currentPage * m.getItemsPerPage()
			}

		case "right", "l":
			// Next page
			totalPages := m.getTotalPages()
			if m.currentPage < totalPages-1 {
				m.currentPage++
				m.cursor = m.currentPage * m.getItemsPerPage()
			}

		case "tab":
			// Switch source
			m.sourceIdx = (m.sourceIdx + 1) % len(hfSources)
			if m.sourceIdx != m.loadedSource {
				m.loading = true
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, m.loadModels(m.sourceIdx))
			}

		case "enter", "i":
			// Install selected model
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) && !m.installing {
				selectedModel := m.filtered[m.cursor]
				m.installing = true
				m.installMsg = fmt.Sprintf("Installing %s...", selectedModel.ID)
				return m, tea.Batch(m.spinner.Tick, m.performInstall(selectedModel.ID))
			}

		case "o":
			// Open in browser
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				selectedModel := m.filtered[m.cursor]
				url := fmt.Sprintf("https://huggingface.co/%s", selectedModel.ID)
				exec.Command("open", url).Start()
			}

		case "d":
			// Open details view
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				selectedModel := m.filtered[m.cursor]
				return m, func() tea.Msg { return openDetailsMsg{model: selectedModel} }
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m searchModel) loadModels(sourceIdx int) tea.Cmd {
	return func() tea.Msg {
		author := ""
		limit := 100
		if sourceIdx < len(hfSources)-1 {
			// Specific community source
			author = hfSources[sourceIdx]
		} else {
			// "All Models" - get ALL MLX models (no author filter, high limit)
			limit = 500
		}

		models, err := m.hfClient.Search("", author, limit)
		if err != nil {
			return loadErrorMsg{err: err}
		}
		return modelsLoadedMsg{models: models, sourceIdx: sourceIdx}
	}
}

func (m *searchModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.allModels
	} else {
		filter := strings.ToLower(m.filter)
		m.filtered = []hf.Model{}
		for _, mdl := range m.allModels {
			if strings.Contains(strings.ToLower(mdl.ID), filter) {
				m.filtered = append(m.filtered, mdl)
			}
		}
	}
	m.cursor = 0
	m.currentPage = 0
}

func (m searchModel) getItemsPerPage() int {
	perPage := m.height - 14
	if perPage < 5 {
		perPage = 10
	}
	return perPage
}

func (m searchModel) getTotalPages() int {
	perPage := m.getItemsPerPage()
	if len(m.filtered) == 0 {
		return 1
	}
	return (len(m.filtered) + perPage - 1) / perPage
}

func (m searchModel) performInstall(modelID string) tea.Cmd {
	return func() tea.Msg {
		// Create cache directory if needed
		cacheDir := m.cfg.ModelDir
		os.MkdirAll(filepath.Join(cacheDir, "cache"), 0755)

		// Parse model name from ID
		parts := strings.Split(modelID, "/")
		modelName := parts[len(parts)-1]

		// Check which hf command is available
		hfCmd := "hf"
		if _, err := exec.LookPath("hf"); err != nil {
			if _, err := exec.LookPath("huggingface-cli"); err != nil {
				return installCompleteMsg{modelID: modelID, err: fmt.Errorf("hf CLI not found. Install with: pip install huggingface_hub")}
			}
			hfCmd = "huggingface-cli"
		}

		// Download using hf
		var cmd *exec.Cmd
		if hfCmd == "hf" {
			cmd = exec.Command(hfCmd, "download", modelID, "--cache-dir", filepath.Join(cacheDir, "cache"), "--no-quiet")
		} else {
			cmd = exec.Command(hfCmd, "download", modelID, "--cache-dir", filepath.Join(cacheDir, "cache"))
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			return installCompleteMsg{modelID: modelID, err: fmt.Errorf("%v: %s", err, string(output))}
		}

		// Find the downloaded snapshot path
		orgName := strings.ReplaceAll(modelID, "/", "--")
		modelCacheDir := filepath.Join(cacheDir, "cache", "models--"+orgName, "snapshots")

		entries, err := os.ReadDir(modelCacheDir)
		if err != nil || len(entries) == 0 {
			return installCompleteMsg{modelID: modelID, err: fmt.Errorf("could not find downloaded model in cache")}
		}

		snapshotPath := filepath.Join(modelCacheDir, entries[0].Name())

		// Create symlink
		symlinkPath := filepath.Join(cacheDir, modelName)
		os.Remove(symlinkPath)

		if err := os.Symlink(snapshotPath, symlinkPath); err != nil {
			return installCompleteMsg{modelID: modelID, err: fmt.Errorf("failed to create symlink: %v", err)}
		}

		return installCompleteMsg{modelID: modelID, err: nil}
	}
}

func (m searchModel) View() string {
	contentWidth := getContentWidth(m.width)
	var b strings.Builder

	// Header (80% width)
	b.WriteString(renderHeader(version, m.width))
	b.WriteString("\n\n")

	// Source tabs
	var tabsBuilder strings.Builder
	for i, src := range hfSources {
		if i == m.sourceIdx {
			tabsBuilder.WriteString(activeTabStyle.Render(src))
		} else {
			tabsBuilder.WriteString(inactiveTabStyle.Render(src))
		}
	}
	b.WriteString(tabBarStyle.Render(tabsBuilder.String()))
	b.WriteString("\n\n")

	// Filter input
	if m.filtering {
		searchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
		b.WriteString(searchStyle.Render("🔍 Filter: ") + m.filter + "█")
		b.WriteString("\n")
	} else if m.filter != "" {
		searchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
		b.WriteString(searchStyle.Render("🔍 \""+m.filter+"\"") + statusMutedStyle.Render(fmt.Sprintf(" (%d results)", len(m.filtered))) + "\n")
	}

	// Column headers - adjusted for 80% width
	nameWidth := contentWidth - 20
	if nameWidth > 50 {
		nameWidth = 50
	}
	dlWidth := 10
	headerStyle := statusMutedStyle.Copy().Bold(true)
	header := fmt.Sprintf("  %-*s  %*s", nameWidth, "Model", dlWidth, "Downloads")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(statusMutedStyle.Render("  " + strings.Repeat("─", nameWidth+dlWidth+4)))
	b.WriteString("\n")

	// Items
	if m.loading {
		b.WriteString(m.spinner.View() + " Loading models from " + hfSources[m.sourceIdx] + "...")
		b.WriteString("\n")
	} else if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString("\n")
	} else if len(m.filtered) == 0 {
		b.WriteString(statusMutedStyle.Render("  No models found"))
		b.WriteString("\n")
	} else {
		perPage := m.getItemsPerPage()
		startIdx := m.currentPage * perPage
		endIdx := startIdx + perPage
		if endIdx > len(m.filtered) {
			endIdx = len(m.filtered)
		}

		renderedLines := 0
		for idx := startIdx; idx < endIdx; idx++ {
			mdl := m.filtered[idx]
			isSelected := idx == m.cursor

			// Check if installed
			installed := ""
			modelName := strings.Split(mdl.ID, "/")
			if len(modelName) > 1 && m.store.Exists(modelName[1]) {
				installed = " ✓"
			}

			prefix := "  "
			nameStyle := menuItemStyle
			if isSelected {
				prefix = "▸ "
				nameStyle = menuItemSelectedStyle
			}

			downloads := hf.FormatDownloads(mdl.Downloads)
			line := fmt.Sprintf("%s%-*s  %*s%s", prefix, nameWidth, truncateStr(mdl.ID, nameWidth), dlWidth, downloads, installed)
			b.WriteString(nameStyle.Render(line))
			b.WriteString("\n")
			renderedLines++
		}

		// Pad for consistent height
		for i := renderedLines; i < perPage; i++ {
			b.WriteString("\n")
		}
	}

	// Install message
	if m.installing {
		b.WriteString("\n" + m.spinner.View() + " " + m.installMsg)
	} else if m.installMsg != "" {
		if strings.Contains(m.installMsg, "Successfully") {
			b.WriteString("\n" + successStyle.Render(m.installMsg))
		} else if strings.Contains(m.installMsg, "failed") {
			b.WriteString("\n" + errorStyle.Render(m.installMsg))
		}
	}

	// Calculate padding to push footer to bottom
	content := b.String()
	contentLines := strings.Count(content, "\n") + 1
	footerLines := 2
	availableHeight := m.height - 4
	if availableHeight > contentLines+footerLines {
		padding := availableHeight - contentLines - footerLines
		b.WriteString(strings.Repeat("\n", padding))
	}

	// Pagination dots
	totalPages := m.getTotalPages()
	var paginationStr string
	if totalPages > 1 && !m.loading {
		var pagination strings.Builder
		pagination.WriteString("  ")

		activeBullet := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
		inactiveBullet := statusMutedStyle

		maxBullets := 9
		if totalPages <= maxBullets {
			for p := 0; p < totalPages; p++ {
				if p == m.currentPage {
					pagination.WriteString(activeBullet.Render("●"))
				} else {
					pagination.WriteString(inactiveBullet.Render("○"))
				}
				if p < totalPages-1 {
					pagination.WriteString(" ")
				}
			}
		} else {
			// Abbreviated pagination for many pages
			if m.currentPage > 0 {
				pagination.WriteString(inactiveBullet.Render("○ "))
			}
			pagination.WriteString(activeBullet.Render("●"))
			if m.currentPage < totalPages-1 {
				pagination.WriteString(inactiveBullet.Render(" ○"))
			}
		}

		pagination.WriteString(fmt.Sprintf("  %d/%d", m.currentPage+1, totalPages))
		paginationStr = statusMutedStyle.Render(pagination.String())
	}

	b.WriteString("\n" + paginationStr)

	// Help bar
	var helpText string
	if m.installing {
		helpText = "Installing... please wait"
	} else if m.filtering {
		helpText = "Type to filter • Enter confirm • Esc clear"
	} else {
		helpText = "Tab source • ←/→ page • ↑/↓ select • ?/filter • i install • o browser • Esc back"
	}
	b.WriteString("\n" + helpStyle.Render(helpText))

	return appStyle.Render(b.String())
}

// Tab styles for source selector
var (
	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 2).
			Bold(true)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Padding(0, 2)

	tabBarStyle = lipgloss.NewStyle().
			MarginBottom(0)
)

// uninstallModel handles model uninstallation
type uninstallModel struct {
	cfg       *config.Config
	store     *model.Store
	models    []model.Model
	selected  int
	width     int
	height    int
	confirm   bool
	confirmed bool
	err       error
}

func newUninstallModel(cfg *config.Config, store *model.Store) uninstallModel {
	models, _ := store.List()
	return uninstallModel{
		cfg:      cfg,
		store:    store,
		models:   models,
		selected: 0,
	}
}

func (m uninstallModel) Init() tea.Cmd {
	return nil
}

func (m uninstallModel) Update(msg tea.Msg) (uninstallModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirm {
			switch msg.String() {
			case "y", "Y":
				if m.selected < len(m.models) {
					modelName := m.models[m.selected].Name
					err := m.store.RemoveWithCache(modelName)
					if err != nil {
						m.err = err
					} else {
						m.models, _ = m.store.List()
						if m.selected >= len(m.models) {
							m.selected = len(m.models) - 1
						}
						if m.selected < 0 {
							m.selected = 0
						}
					}
				}
				m.confirm = false
				m.confirmed = true
				return m, nil
			case "n", "N", "esc":
				m.confirm = false
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
			maxIdx := len(m.models)
			if m.selected < maxIdx {
				m.selected++
			}
		case "enter":
			if m.selected == len(m.models) {
				return m, func() tea.Msg { return goBackMsg{} }
			}
			if m.selected < len(m.models) {
				m.confirm = true
				m.confirmed = false
			}
		case "d":
			if m.selected < len(m.models) {
				m.confirm = true
				m.confirmed = false
			}
		case "esc":
			return m, func() tea.Msg { return goBackMsg{} }
		}
	}
	return m, nil
}

func (m uninstallModel) View() string {
	contentWidth := getContentWidth(m.width)
	var b strings.Builder

	// Header (80% width)
	b.WriteString(renderHeader(version, m.width))
	b.WriteString("\n\n")

	// Section title
	b.WriteString(subtitleStyle.Render("Uninstall a Model"))
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
			if mdl.IsSymlink {
				line += " (symlink)"
			}
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

	// Error message
	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	// Confirmation message
	if m.confirm && m.selected < len(m.models) {
		b.WriteString("\n" + warningStyle.Render(fmt.Sprintf("Delete %s and its cache? [y/N]", m.models[m.selected].Name)))
	} else if m.confirmed {
		b.WriteString("\n" + successStyle.Render("Model deleted successfully"))
	}

	// Status
	b.WriteString("\n\n")
	b.WriteString(infoLineStyle.Render(fmt.Sprintf("Models: %s (%d installed)", config.DisplayPath(m.cfg.ModelDir), len(m.models))))

	// Calculate padding to push footer to bottom
	content := b.String()
	contentLines := strings.Count(content, "\n") + 1
	padding := calculatePadding(contentLines, 1, m.height)
	b.WriteString(strings.Repeat("\n", padding))

	// Footer
	helpText := "[↵/d] delete  [↑/↓] navigate  [esc] back"
	b.WriteString("\n" + helpStyle.Render(helpText))

	return appStyle.Render(b.String())
}

var warningStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#F59E0B")).
	Bold(true)
