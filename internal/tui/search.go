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

// Loading modes
const (
	loadModeAll    = "all"    // Load all models from source
	loadModeSearch = "search" // Live search from API
)

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
	filtering    bool   // In filter input mode
	loading      bool
	loadedSource int    // Track which source is loaded
	loadingMode  string // "all" or "search"
	searchQuery  string // Search query for live search mode
	searching    bool   // In search input mode
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
		loadedSource: -1,            // Not loaded yet
		loadingMode:  loadModeAll,   // Default to All mode (load models on startup)
		loading:      true,          // Start in loading state
		spinner:      s,
	}
}

func (m searchModel) Init() tea.Cmd {
	// In search mode, don't auto-load - wait for user search query
	if m.loadingMode == loadModeSearch {
		return m.spinner.Tick
	}
	// In all mode (default), auto-load models from first source
	return tea.Batch(m.spinner.Tick, m.loadModels(0, ""))
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

		// Search input mode (for live search)
		if m.searching {
			switch msg.String() {
			case "enter":
				m.searching = false
				if m.searchQuery != "" {
					m.loading = true
					m.err = nil
					return m, tea.Batch(m.spinner.Tick, m.loadModels(m.sourceIdx, m.searchQuery))
				}
				return m, nil
			case "esc":
				// ESC clears search and switches to All mode
				m.searching = false
				m.searchQuery = ""
				if m.loadingMode == loadModeSearch {
					m.loadingMode = loadModeAll
					m.loading = true
					m.err = nil
					return m, tea.Batch(m.spinner.Tick, m.loadModels(m.sourceIdx, ""))
				}
				return m, nil
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				}
				return m, nil
			default:
				if len(msg.String()) == 1 {
					m.searchQuery += msg.String()
				}
				return m, nil
			}
		}

		// Filter input mode
		if m.filtering {
			switch msg.String() {
			case "enter":
				m.filtering = false
				m.cursor = 0
				return m, nil
			case "esc":
				// ESC clears filter and shows all models
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
		case "q":
			// Navigate to homepage
			return m, func() tea.Msg { return goBackMsg{} }

		case "a":
			// Switch to "All models" mode
			if m.loadingMode != loadModeAll {
				m.loadingMode = loadModeAll
				m.searchQuery = ""
				m.allModels = nil
				m.filtered = nil
				m.cursor = 0
				m.currentPage = 0
				m.loading = true
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, m.loadModels(m.sourceIdx, ""))
			}

		case "s":
			// Switch to "Search" mode
			if m.loadingMode != loadModeSearch {
				m.loadingMode = loadModeSearch
				m.allModels = nil
				m.filtered = nil
				m.cursor = 0
				m.currentPage = 0
				m.loadedSource = -1
				m.searching = true // Open search input
			} else {
				// Already in search mode, open search input
				m.searching = true
			}
			return m, nil

		case "?", "/", "f":
			// Enter filter mode (only if we have loaded models)
			if len(m.allModels) > 0 {
				m.filtering = true
			}
			return m, nil

		case "esc":
			// In search mode with results: ESC clears search and switches to All mode
			if m.loadingMode == loadModeSearch {
				m.loadingMode = loadModeAll
				m.searchQuery = ""
				m.loading = true
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, m.loadModels(m.sourceIdx, ""))
			}
			// In filter mode with active filter: clear filter
			if m.filter != "" {
				m.filter = ""
				m.applyFilter()
				return m, nil
			}
			return m, nil

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
			// In all mode, reload models from new source
			if m.loadingMode == loadModeAll && m.sourceIdx != m.loadedSource {
				m.loading = true
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, m.loadModels(m.sourceIdx, ""))
			}
			// In search mode, clear results and let user search again
			if m.loadingMode == loadModeSearch {
				m.allModels = nil
				m.filtered = nil
				m.cursor = 0
				m.currentPage = 0
				m.loadedSource = -1
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

func (m searchModel) loadModels(sourceIdx int, searchQuery string) tea.Cmd {
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

		models, err := m.hfClient.Search(searchQuery, author, limit)
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
	perPage := m.height - 18 // Increased from 14 to leave room for logo header
	if perPage < 5 {
		perPage = 5
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

	// Header (80% width) - same spacing as menu.go
	b.WriteString(renderHeader(version, m.width))
	b.WriteString("\n\n\n")

	// Source tabs + mode indicators
	var tabsBuilder strings.Builder
	for i, src := range hfSources {
		if i == m.sourceIdx {
			tabsBuilder.WriteString(activeTabStyle.Render(src))
		} else {
			tabsBuilder.WriteString(inactiveTabStyle.Render(src))
		}
	}
	// Add mode indicators: 📁 (All) and 🔍 (Search)
	tabsBuilder.WriteString("  ")
	if m.loadingMode == loadModeAll {
		tabsBuilder.WriteString(activeTabStyle.Render("📁"))
		tabsBuilder.WriteString(inactiveTabStyle.Render("🔍"))
	} else {
		tabsBuilder.WriteString(inactiveTabStyle.Render("📁"))
		tabsBuilder.WriteString(activeTabStyle.Render("🔍"))
	}
	b.WriteString(tabBarStyle.Render(tabsBuilder.String()))
	b.WriteString("\n")

	// Search/filter input area - ALWAYS reserve one line for layout stability
	var inputLine strings.Builder
	
	// Search part
	if m.searching {
		searchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
		inputLine.WriteString(searchStyle.Render("🔎 Search: ") + m.searchQuery + "█")
	} else if m.loadingMode == loadModeSearch && m.searchQuery != "" {
		searchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
		inputLine.WriteString(searchStyle.Render("🔎 \"" + m.searchQuery + "\""))
	}
	
	// Add separator if both search and filter are visible
	if inputLine.Len() > 0 && (m.filtering || m.filter != "") {
		inputLine.WriteString("  ")
	}
	
	// Filter part
	if m.filtering {
		filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
		inputLine.WriteString(filterStyle.Render("🔍 Filter: ") + m.filter + "█")
	} else if m.filter != "" {
		filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
		inputLine.WriteString(filterStyle.Render("🔍 \"" + m.filter + "\""))
		inputLine.WriteString(statusMutedStyle.Render(fmt.Sprintf(" (%d)", len(m.filtered))))
	}
	
	// Always output a fixed line (empty or with content) for layout stability
	b.WriteString("\n") // Newline before search/filter line
	b.WriteString(inputLine.String())
	b.WriteString("\n\n")

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
	perPage := m.getItemsPerPage()
	
	if m.loading {
		b.WriteString(m.spinner.View() + " Loading models from " + hfSources[m.sourceIdx] + "...")
		b.WriteString("\n")
		// Pad to maintain consistent height during loading
		for i := 1; i < perPage; i++ {
			b.WriteString("\n")
		}
	} else if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString("\n")
		// Pad to maintain consistent height
		for i := 1; i < perPage; i++ {
			b.WriteString("\n")
		}
	} else if len(m.filtered) == 0 {
		// Different message based on mode
		var emptyMsg string
		if m.loadingMode == loadModeSearch && m.searchQuery == "" {
			emptyMsg = "  Press 's' to enter search query"
		} else if m.loadingMode == loadModeSearch {
			emptyMsg = "  No models found for \"" + m.searchQuery + "\""
		} else {
			emptyMsg = "  No models found"
		}
		b.WriteString(statusMutedStyle.Render(emptyMsg))
		b.WriteString("\n")
		// Pad to maintain consistent height
		for i := 1; i < perPage; i++ {
			b.WriteString("\n")
		}
	} else {
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

		maxBullets := 30
		if totalPages <= maxBullets {
			// Show all dots if we have 25 or fewer pages
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
			// Abbreviated pagination for many pages (>25)
			// Show: ○ ... ○ ● ○ ... ○
			if m.currentPage > 1 {
				pagination.WriteString(inactiveBullet.Render("○ ... "))
			} else if m.currentPage == 1 {
				pagination.WriteString(inactiveBullet.Render("○ "))
			}
			
			// Show current and adjacent pages
			if m.currentPage > 0 {
				pagination.WriteString(inactiveBullet.Render("○ "))
			}
			pagination.WriteString(activeBullet.Render("●"))
			if m.currentPage < totalPages-1 {
				pagination.WriteString(inactiveBullet.Render(" ○"))
			}
			
			if m.currentPage < totalPages-2 {
				pagination.WriteString(inactiveBullet.Render(" ... ○"))
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
	} else if m.searching {
		helpText = "Type to search • Enter fetch • Esc clear"
	} else if m.filtering {
		helpText = "Type to filter • Enter confirm • Esc clear"
	} else {
		helpText = "Tab source • a All • s Search • f filter • ←/→ page • ↑/↓ select • i install • q back"
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
