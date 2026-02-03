package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarques/efx-face-manager/internal/config"
	"github.com/lmarques/efx-face-manager/internal/hf"
	"github.com/lmarques/efx-face-manager/internal/model"
)

// detailsModel shows detailed model information
type detailsModel struct {
	cfg        *config.Config
	store      *model.Store
	hfClient   *hf.Client
	model      hf.Model
	width      int
	height     int
	selected   int // 0=Cancel, 1=Open Browser, 2=Install
	installing bool
	installed  bool
	err        error
	message    string
}

func newDetailsModel(cfg *config.Config, store *model.Store, hfModel hf.Model) detailsModel {
	// Check if already installed
	parts := strings.Split(hfModel.ID, "/")
	modelName := parts[len(parts)-1]
	installed := store.Exists(modelName)

	return detailsModel{
		cfg:       cfg,
		store:     store,
		hfClient:  hf.NewClient(),
		model:     hfModel,
		selected:  2, // Default to Install
		installed: installed,
	}
}

func (m detailsModel) Init() tea.Cmd {
	return nil
}

func (m detailsModel) Update(msg tea.Msg) (detailsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case installCompleteMsg:
		m.installing = false
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Install failed: %v", msg.err)
		} else {
			m.installed = true
			m.message = "Successfully installed!"
			// Refresh store
			m.store = model.NewStore(m.cfg.ModelDir)
		}

	case tea.KeyMsg:
		if m.installing {
			return m, nil
		}

		switch msg.String() {
		case "left", "h":
			if m.selected > 0 {
				m.selected--
			}
		case "right", "l":
			maxOpt := 2
			if m.installed {
				maxOpt = 1
			}
			if m.selected < maxOpt {
				m.selected++
			}
		case "enter":
			switch m.selected {
			case 0: // Cancel
				return m, func() tea.Msg { return goBackMsg{} }
			case 1: // Open Browser
				url := fmt.Sprintf("https://huggingface.co/%s", m.model.ID)
				exec.Command("open", url).Start()
			case 2: // Install
				if !m.installed {
					m.installing = true
					m.message = "Installing..."
					return m, m.performInstall()
				}
			}
		case "i":
			if !m.installed && !m.installing {
				m.installing = true
				m.message = "Installing..."
				return m, m.performInstall()
			}
		case "o":
			url := fmt.Sprintf("https://huggingface.co/%s", m.model.ID)
			exec.Command("open", url).Start()
		case "esc":
			return m, func() tea.Msg { return goBackMsg{} }
		}
	}
	return m, nil
}

func (m detailsModel) performInstall() tea.Cmd {
	return func() tea.Msg {
		modelID := m.model.ID
		parts := strings.Split(modelID, "/")
		modelName := parts[len(parts)-1]

		// Download using huggingface-cli
		cmd := exec.Command("huggingface-cli", "download", modelID)
		
		// Use cache directory
		cacheDir := m.cfg.ModelDir + "/cache"
		cmd.Env = append(cmd.Environ(), fmt.Sprintf("HF_HOME=%s", cacheDir))

		output, err := cmd.CombinedOutput()
		if err != nil {
			return installCompleteMsg{modelID: modelID, err: fmt.Errorf("%v: %s", err, string(output))}
		}

		// Find snapshot and create symlink
		orgName := strings.ReplaceAll(modelID, "/", "--")
		modelCacheDir := cacheDir + "/models--" + orgName + "/snapshots"

		entries, err := exec.Command("ls", modelCacheDir).Output()
		if err != nil {
			return installCompleteMsg{modelID: modelID, err: fmt.Errorf("could not find downloaded model")}
		}

		snapshots := strings.Fields(string(entries))
		if len(snapshots) == 0 {
			return installCompleteMsg{modelID: modelID, err: fmt.Errorf("no snapshots found")}
		}

		snapshotPath := modelCacheDir + "/" + snapshots[0]
		symlinkPath := m.cfg.ModelDir + "/" + modelName

		// Remove existing symlink
		exec.Command("rm", "-f", symlinkPath).Run()

		// Create symlink
		if err := exec.Command("ln", "-s", snapshotPath, symlinkPath).Run(); err != nil {
			return installCompleteMsg{modelID: modelID, err: fmt.Errorf("failed to create symlink: %v", err)}
		}

		return installCompleteMsg{modelID: modelID, err: nil}
	}
}

func (m detailsModel) View() string {
	contentWidth := getContentWidth(m.width)
	var b strings.Builder

	// Header (80% width)
	b.WriteString(renderHeader(version, m.width))
	b.WriteString("\n\n")

	// Model ID
	b.WriteString(subtitleStyle.Render(m.model.ID))
	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render(strings.Repeat("─", contentWidth-4)))
	b.WriteString("\n\n")

	// Model details
	b.WriteString(fmt.Sprintf("  %-15s %s\n", "Models:", config.DisplayPath(m.cfg.ModelDir)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %-15s %s\n", "Downloads:", formatNumber(m.model.Downloads)))
	b.WriteString(fmt.Sprintf("  %-15s %s\n", "Likes:", formatNumber(m.model.Likes)))
	if m.model.PipelineTag != "" {
		b.WriteString(fmt.Sprintf("  %-15s %s\n", "Pipeline:", m.model.PipelineTag))
	}
	if m.model.LibraryName != "" {
		b.WriteString(fmt.Sprintf("  %-15s %s\n", "Library:", m.model.LibraryName))
	}
	if m.model.LastModified != "" {
		date := m.model.LastModified
		if len(date) > 10 {
			date = date[:10]
		}
		b.WriteString(fmt.Sprintf("  %-15s %s\n", "Updated:", date))
	}

	// Status indicator
	if m.installed {
		b.WriteString("\n")
		b.WriteString(successStyle.Render("  [INSTALLED]"))
		b.WriteString("\n")
	}

	// Action buttons
	b.WriteString("\n")
	b.WriteString(m.renderButtons())
	b.WriteString("\n")

	// Message area
	if m.message != "" {
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(errorStyle.Render(m.message))
		} else if m.installing {
			b.WriteString(infoLineStyle.Render(m.message))
		} else {
			b.WriteString(successStyle.Render(m.message))
		}
	}

	// Calculate padding to push footer to bottom
	content := b.String()
	contentLines := strings.Count(content, "\n") + 1
	padding := calculatePadding(contentLines, 1, m.height)
	b.WriteString(strings.Repeat("\n", padding))

	// Footer
	helpText := "[i] install  [o] open browser  [←/→] navigate  [↵] select  [esc] back"
	b.WriteString("\n" + helpStyle.Render(helpText))

	return appStyle.Render(b.String())
}

func (m detailsModel) renderButtons() string {
	cancelStyle := buttonStyle
	browserStyle := buttonStyle
	installStyle := buttonStyle

	if m.selected == 0 {
		cancelStyle = buttonSelectedStyle
	}
	if m.selected == 1 {
		browserStyle = buttonSelectedStyle
	}
	if m.selected == 2 {
		installStyle = buttonSelectedStyle
	}

	cancel := cancelStyle.Render("[ Cancel ]")
	browser := browserStyle.Render("[ Open in Browser ]")

	var install string
	if m.installed {
		install = buttonDisabledStyle.Render("[ Installed ]")
	} else if m.installing {
		install = buttonDisabledStyle.Render("[ Installing... ]")
	} else {
		install = installStyle.Render("[ Install ]")
	}

	return lipgloss.JoinHorizontal(lipgloss.Center,
		"           ",
		cancel,
		"  ",
		browser,
		"  ",
		install,
	)
}

// Button styles
var (
	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A1A1AA")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#52525B")).
			Padding(0, 1)

	buttonSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7C3AED")).
				Bold(true).
				Padding(0, 1)

	buttonDisabledStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#71717A")).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#3F3F46")).
				Padding(0, 1)
)

// formatNumber formats large numbers nicely
func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
