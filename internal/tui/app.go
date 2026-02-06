package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarques/efx-face-manager/internal/config"
	"github.com/lmarques/efx-face-manager/internal/hf"
	"github.com/lmarques/efx-face-manager/internal/model"
	"github.com/lmarques/efx-face-manager/internal/server"
)

const version = "0.2.0"

// View states
type viewState int

const (
	viewMenu viewState = iota
	viewTemplates
	viewModels
	viewModelType
	viewConfig
	viewSearch
	viewDetails
	viewServerManager
	viewNewServer
	viewStorageConfig
	viewUninstall
	viewChat
)

// Main application model
type appModel struct {
	state          viewState
	history        []viewState // Navigation history stack
	lastQPressTime int64       // Track last 'q' press for double-q quit
	width          int
	height         int
	err            error

	// Core services
	cfg     *config.Config
	store   *model.Store
	servers *server.Manager

	// Sub-models
	menuModel          menuModel
	templatesModel     templatesModel
	modelsModel        modelsModel
	modelTypeModel     modelTypeModel
	configPanelModel   configPanelModel
	serverManagerModel serverManagerModel
	storageModel       storageModel
	searchModel        searchModel
	uninstallModel     uninstallModel
	detailsModel       detailsModel
	serverNewModel     serverNewModel
	chatModel          chatModel
}

// Initialize the main model
func initialModel() appModel {
	cfg, _ := config.Load()
	store := model.NewStore(cfg.ModelDir)
	servers := server.NewManager()

	return appModel{
		state:     viewMenu,
		history:   []viewState{},
		cfg:       cfg,
		store:     store,
		servers:   servers,
		menuModel: newMenuModel(cfg, store),
	}
}

// pushHistory adds current state to history before navigating (returns new history)
func pushHistory(history []viewState, state viewState) []viewState {
	return append(history, state)
}

// popHistory returns previous state and new history
func popHistory(history []viewState) (viewState, []viewState) {
	if len(history) == 0 {
		return viewMenu, history
	}
	prevState := history[len(history)-1]
	return prevState, history[:len(history)-1]
}

func (m appModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		listenForServerUpdates(m.servers),
	)
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle ESC globally for ALL views EXCEPT viewSearch and viewChat (which handle their own ESC)
		if msg.String() == "esc" && m.state != viewMenu && m.state != viewSearch && m.state != viewChat {
			prevState, newHistory := popHistory(m.history)
			m.history = newHistory
			m.state = prevState

			// Reinitialize the model for the previous state
			switch prevState {
			case viewMenu:
				m.menuModel = newMenuModel(m.cfg, m.store)
				m.menuModel.width = m.width
				m.menuModel.height = m.height
				m.menuModel.serverCount = m.servers.Count()
			case viewTemplates:
				m.templatesModel = newTemplatesModel(m.cfg, m.store)
				m.templatesModel.width = m.width
				m.templatesModel.height = m.height
			case viewModels:
				m.modelsModel = newModelsModel(m.cfg, m.store)
				m.modelsModel.width = m.width
				m.modelsModel.height = m.height
			case viewModelType:
				modelName := ""
				if m.configPanelModel.config.Model != "" {
					modelName = m.configPanelModel.config.Model
				}
				m.modelTypeModel = newModelTypeModel(modelName, m.cfg)
				m.modelTypeModel.width = m.width
				m.modelTypeModel.height = m.height
			case viewSearch:
				m.searchModel = newSearchModel(m.cfg, m.store)
				m.searchModel.width = m.width
				m.searchModel.height = m.height
			case viewUninstall:
				m.uninstallModel = newUninstallModel(m.cfg, m.store)
				m.uninstallModel.width = m.width
				m.uninstallModel.height = m.height
			case viewServerManager:
				m.serverManagerModel = newServerManagerModel(m.servers, m.width, m.height)
			case viewNewServer:
				m.serverNewModel = newServerNewModel(m.cfg, m.store, m.servers)
				m.serverNewModel.width = m.width
				m.serverNewModel.height = m.height
			}
			return m, nil
		}

		// Handle ctrl+c globally
		if msg.String() == "ctrl+c" {
			m.servers.StopAll()
			return m, tea.Quit
		}

		// Handle 'q' key - skip for views with text input
		if m.state != viewSearch && m.state != viewConfig && m.state != viewStorageConfig && m.state != viewChat {
			if msg.String() == "q" {
				if m.state == viewMenu {
					// On home page - quit application
					m.servers.StopAll()
					return m, tea.Quit
				}
				// On any other page - return to home page
				m.history = []viewState{} // Clear history
				m.state = viewMenu
				m.menuModel = newMenuModel(m.cfg, m.store)
				m.menuModel.width = m.width
				m.menuModel.height = m.height
				m.menuModel.serverCount = m.servers.Count()
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update sub-models with new size
		m.menuModel.width = msg.Width
		m.menuModel.height = msg.Height

	case serverUpdateMsg:
		// Handle server updates
		cmds = append(cmds, listenForServerUpdates(m.servers))
		if m.state == viewServerManager {
			m.serverManagerModel, _ = m.serverManagerModel.Update(msg)
		}
		if m.state == viewChat {
			m.chatModel, _ = m.chatModel.Update(msg)
		}
		// Update menu to show server count
		m.menuModel.serverCount = m.servers.Count()
		return m, tea.Batch(cmds...)

	case openTemplatesMsg:
		m.history = pushHistory(m.history, m.state)
		m.state = viewTemplates
		m.templatesModel = newTemplatesModel(m.cfg, m.store)
		m.templatesModel.width = m.width
		m.templatesModel.height = m.height
		return m, nil

	case openModelsMsg:
		m.history = pushHistory(m.history, m.state)
		m.state = viewModels
		m.modelsModel = newModelsModel(m.cfg, m.store)
		m.modelsModel.width = m.width
		m.modelsModel.height = m.height
		return m, nil

	case openModelTypeMsg:
		m.history = pushHistory(m.history, m.state)
		m.state = viewModelType
		m.modelTypeModel = newModelTypeModel(msg.model, m.cfg)
		m.modelTypeModel.width = m.width
		m.modelTypeModel.height = m.height
		return m, nil

	case openConfigPanelMsg:
		m.history = pushHistory(m.history, m.state)
		m.state = viewConfig
		m.configPanelModel = newConfigPanelModel(msg.config, m.cfg, m.servers)
		m.configPanelModel.width = m.width
		m.configPanelModel.height = m.height
		return m, nil

	case openServerManagerMsg:
		m.history = pushHistory(m.history, m.state)
		m.state = viewServerManager
		m.serverManagerModel = newServerManagerModel(m.servers, m.width, m.height)
		return m, nil

	case openStorageConfigMsg:
		m.history = pushHistory(m.history, m.state)
		m.state = viewStorageConfig
		m.storageModel = newStorageModel(m.cfg)
		m.storageModel.width = m.width
		m.storageModel.height = m.height
		return m, nil

	case serverStartedMsg:
		// Server started, go to server manager
		m.history = pushHistory(m.history, m.state)
		m.state = viewServerManager
		m.serverManagerModel = newServerManagerModel(m.servers, m.width, m.height)
		m.serverManagerModel.selectedPort = msg.port
		return m, listenForServerUpdates(m.servers)

	case configSavedMsg:
		m.cfg = msg.config
		m.store = model.NewStore(m.cfg.ModelDir)
		m.history = []viewState{} // Clear history when returning to menu
		m.state = viewMenu
		m.menuModel = newMenuModel(m.cfg, m.store)
		m.menuModel.width = m.width
		m.menuModel.height = m.height
		m.menuModel.serverCount = m.servers.Count()
		return m, nil

	case goBackMsg:
		// Go back to menu (clear history)
		m.history = []viewState{}
		m.state = viewMenu
		m.menuModel = newMenuModel(m.cfg, m.store)
		m.menuModel.width = m.width
		m.menuModel.height = m.height
		m.menuModel.serverCount = m.servers.Count()
		return m, nil

	case openInstallMsg:
		m.history = pushHistory(m.history, m.state)
		m.state = viewSearch
		m.searchModel = newSearchModel(m.cfg, m.store)
		m.searchModel.width = m.width
		m.searchModel.height = m.height
		return m, m.searchModel.Init()

	case openUninstallMsg:
		m.history = pushHistory(m.history, m.state)
		m.state = viewUninstall
		m.uninstallModel = newUninstallModel(m.cfg, m.store)
		m.uninstallModel.width = m.width
		m.uninstallModel.height = m.height
		return m, m.uninstallModel.Init()

	case openDetailsMsg:
		m.history = pushHistory(m.history, m.state)
		m.state = viewDetails
		m.detailsModel = newDetailsModel(m.cfg, m.store, msg.model)
		m.detailsModel.width = m.width
		m.detailsModel.height = m.height
		return m, nil

	case openNewServerMsg:
		m.history = pushHistory(m.history, m.state)
		m.state = viewNewServer
		m.serverNewModel = newServerNewModel(m.cfg, m.store, m.servers)
		m.serverNewModel.width = m.width
		m.serverNewModel.height = m.height
		return m, nil

	case openConfigMsg:
		// From server_new, open config with pre-filled values
		cfg := server.NewConfig()
		cfg.Model = msg.model
		cfg.ModelPath = m.cfg.ModelDir + "/" + msg.model
		cfg.Type = msg.modelType
		cfg.Port = msg.port
		m.history = pushHistory(m.history, m.state)
		m.state = viewConfig
		m.configPanelModel = newConfigPanelModel(cfg, m.cfg, m.servers)
		m.configPanelModel.width = m.width
		m.configPanelModel.height = m.height
		return m, nil

	case openChatMsg:
		// Open chat window for a server
		m.history = pushHistory(m.history, m.state)
		m.state = viewChat
		serverName := ""
		if inst := m.servers.Get(msg.port); inst != nil {
			serverName = inst.Model
		}
		m.chatModel = newChatModel(msg.port, serverName, m.servers, msg.conversationID, m.width, m.height)
		return m, nil
	}

	// Delegate to sub-models
	var cmd tea.Cmd
	switch m.state {
	case viewMenu:
		m.menuModel, cmd = m.menuModel.Update(msg)
	case viewTemplates:
		m.templatesModel, cmd = m.templatesModel.Update(msg)
	case viewModels:
		m.modelsModel, cmd = m.modelsModel.Update(msg)
	case viewModelType:
		m.modelTypeModel, cmd = m.modelTypeModel.Update(msg)
	case viewConfig:
		m.configPanelModel, cmd = m.configPanelModel.Update(msg)
	case viewServerManager:
		m.serverManagerModel, cmd = m.serverManagerModel.Update(msg)
	case viewStorageConfig:
		m.storageModel, cmd = m.storageModel.Update(msg)
	case viewSearch:
		m.searchModel, cmd = m.searchModel.Update(msg)
	case viewUninstall:
		m.uninstallModel, cmd = m.uninstallModel.Update(msg)
	case viewDetails:
		m.detailsModel, cmd = m.detailsModel.Update(msg)
	case viewNewServer:
		m.serverNewModel, cmd = m.serverNewModel.Update(msg)
	case viewChat:
		m.chatModel, cmd = m.chatModel.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m appModel) View() string {
	switch m.state {
	case viewMenu:
		return m.menuModel.View()
	case viewTemplates:
		return m.templatesModel.View()
	case viewModels:
		return m.modelsModel.View()
	case viewModelType:
		return m.modelTypeModel.View()
	case viewConfig:
		return m.configPanelModel.View()
	case viewServerManager:
		return m.serverManagerModel.View()
	case viewStorageConfig:
		return m.storageModel.View()
	case viewSearch:
		return m.searchModel.View()
	case viewUninstall:
		return m.uninstallModel.View()
	case viewDetails:
		return m.detailsModel.View()
	case viewNewServer:
		return m.serverNewModel.View()
	case viewChat:
		return m.chatModel.View()
	default:
		return m.menuModel.View()
	}
}

// Message types
type openTemplatesMsg struct{}
type openModelsMsg struct{}
type openModelTypeMsg struct{ model string }
type openConfigPanelMsg struct{ config server.Config }
type openServerManagerMsg struct{}
type openStorageConfigMsg struct{}
type openInstallMsg struct{}
type openUninstallMsg struct{}
type openNewServerMsg struct{}
type serverStartedMsg struct{ port int }
type configSavedMsg struct{ config *config.Config }
type serverUpdateMsg server.Update
type goBackMsg struct{} // Navigation back to menu

// Command to listen for server updates
func listenForServerUpdates(mgr *server.Manager) tea.Cmd {
	return func() tea.Msg {
		update := <-mgr.Updates
		return serverUpdateMsg(update)
	}
}

// Viewport helper for scrollable content

// Run starts the main TUI
func Run() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// RunModel runs a specific model
func RunModel(modelName string) error {
	// TODO: Implement direct model run
	return Run()
}

// RunList lists installed models
func RunList() error {
	cfg, _ := config.Load()
	store := model.NewStore(cfg.ModelDir)
	models, _ := store.List()

	println("Installed Models")
	println("================")
	println()
	println("Storage:", config.DisplayPath(cfg.ModelDir))
	println("Total:", len(models))
	println()

	for _, m := range models {
		println("  •", m.Name)
	}

	return nil
}

// RunSearch searches HuggingFace models (CLI mode)
func RunSearch(query string) error {
	if query == "" {
		// No query = open TUI search
		m := initialModel()
		m.state = viewSearch
		m.searchModel = newSearchModel(m.cfg, m.store)
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		_, err := p.Run()
		return err
	}

	// CLI mode: search and print results
	client := hf.NewClient()
	results, err := client.Search(query, "", 20)
	if err != nil {
		return err
	}

	println()
	println("HuggingFace MLX Models")
	println("======================")
	println()
	println("Query:", query)
	println("Found:", len(results))
	println()

	for _, m := range results {
		downloads := hf.FormatDownloads(m.Downloads)
		fmt.Printf("  • %-45s %8s ↓\n", m.ID, downloads)
	}

	return nil
}

// RunServerManager opens server manager
func RunServerManager() error {
	m := initialModel()
	m.state = viewServerManager
	m.serverManagerModel = newServerManagerModel(m.servers, 80, 24)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// RunConfig opens config view
func RunConfig() error {
	m := initialModel()
	m.state = viewStorageConfig
	m.storageModel = newStorageModel(m.cfg)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// RunInstall installs a model from HuggingFace (CLI mode)
func RunInstall(repoID string) error {
	cfg, _ := config.Load()

	fmt.Println()
	fmt.Println("Installing model:", repoID)
	fmt.Println("Target:", config.DisplayPath(cfg.ModelDir))
	fmt.Println()

	// Use huggingface-cli to download
	client := hf.NewClient()

	fmt.Println("Downloading from HuggingFace...")
	err := client.Download(repoID, cfg.ModelDir)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Println("Successfully installed:", repoID)
	return nil
}

// RunUninstall removes a model (CLI mode)
func RunUninstall(modelName string) error {
	cfg, _ := config.Load()
	store := model.NewStore(cfg.ModelDir)

	// Check if model exists
	if !store.Exists(modelName) {
		return fmt.Errorf("model not found: %s", modelName)
	}

	fmt.Println()
	fmt.Println("Uninstalling model:", modelName)
	fmt.Println("Storage:", config.DisplayPath(cfg.ModelDir))
	fmt.Println()

	// Remove with cache
	err := store.RemoveWithCache(modelName)
	if err != nil {
		return fmt.Errorf("uninstall failed: %w", err)
	}

	fmt.Println("Successfully uninstalled:", modelName)
	return nil
}
