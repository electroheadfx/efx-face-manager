package server

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// UpdateType represents the type of server update
type UpdateType int

const (
	UpdateStarted UpdateType = iota
	UpdateStopped
	UpdateNewOutput
	UpdateError
)

// Update represents a server update message
type Update struct {
	Port int
	Type UpdateType
	Data string
}

// Instance represents a running server instance
type Instance struct {
	Model     string
	Type      string
	Port      int
	Host      string
	Args      []string
	Cmd       *exec.Cmd
	PTY       *os.File
	Output    *RingBuffer
	StartedAt time.Time
	Running   bool
	mu        sync.Mutex
}

// Manager handles multiple concurrent server instances
type Manager struct {
	instances map[int]*Instance
	mu        sync.RWMutex
	Updates   chan Update
}

// NewManager creates a new server manager
func NewManager() *Manager {
	return &Manager{
		instances: make(map[int]*Instance),
		Updates:   make(chan Update, 100),
	}
}

// Start starts a new server instance
func (m *Manager) Start(config Config) (*Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check port availability
	if _, exists := m.instances[config.Port]; exists {
		return nil, fmt.Errorf("port %d already in use", config.Port)
	}

	// Build command args
	args := config.BuildArgs()

	instance := &Instance{
		Model:     config.Model,
		Type:      string(config.Type),
		Port:      config.Port,
		Host:      config.Host,
		Args:      args,
		Output:    NewRingBuffer(1000),
		StartedAt: time.Now(),
	}

	// Start the command with PTY
	cmd := exec.Command("mlx-openai-server", args...)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	instance.Cmd = cmd
	instance.PTY = ptmx
	instance.Running = true

	m.instances[config.Port] = instance

	// Read output in goroutine
	go instance.readOutput(m.Updates)

	// Wait for process in goroutine
	go func() {
		cmd.Wait()
		m.mu.Lock()
		if inst, exists := m.instances[config.Port]; exists {
			inst.Running = false
		}
		m.mu.Unlock()
		m.Updates <- Update{Port: config.Port, Type: UpdateStopped}
	}()

	m.Updates <- Update{Port: config.Port, Type: UpdateStarted}
	return instance, nil
}

// Stop stops a server instance
func (m *Manager) Stop(port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.instances[port]
	if !exists {
		return fmt.Errorf("no server on port %d", port)
	}

	// Send SIGTERM for graceful shutdown
	if instance.Cmd.Process != nil {
		if err := instance.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// Force kill if SIGTERM fails
			instance.Cmd.Process.Kill()
		}
	}

	// Close PTY
	if instance.PTY != nil {
		instance.PTY.Close()
	}

	instance.Running = false
	delete(m.instances, port)

	return nil
}

// StopAll stops all server instances
func (m *Manager) StopAll() error {
	m.mu.Lock()
	ports := make([]int, 0, len(m.instances))
	for port := range m.instances {
		ports = append(ports, port)
	}
	m.mu.Unlock()

	for _, port := range ports {
		m.Stop(port)
	}
	return nil
}

// Get returns a server instance by port
func (m *Manager) Get(port int) *Instance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.instances[port]
}

// GetLogs returns the logs for a server
func (m *Manager) GetLogs(port int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if instance, exists := m.instances[port]; exists {
		return instance.Output.String()
	}
	return ""
}

// List returns all server instances sorted by port
func (m *Manager) List() []*Instance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*Instance, 0, len(m.instances))
	for _, instance := range m.instances {
		list = append(list, instance)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Port < list[j].Port
	})

	return list
}

// Count returns the number of running servers
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.instances)
}

// IsPortInUse checks if a port is in use
func (m *Manager) IsPortInUse(port int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.instances[port]
	return exists
}

// NextAvailablePort returns the next available port
func (m *Manager) NextAvailablePort(startPort int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	port := startPort
	for m.instances[port] != nil {
		port++
	}
	return port
}

// readOutput reads output from PTY and stores it
func (i *Instance) readOutput(updates chan Update) {
	buf := make([]byte, 1024)
	for {
		n, err := i.PTY.Read(buf)
		if err != nil {
			if err != io.EOF {
				i.Output.Write(fmt.Sprintf("Error reading output: %v", err))
			}
			return
		}
		if n > 0 {
			lines := strings.Split(string(buf[:n]), "\n")
			for _, line := range lines {
				if line != "" {
					i.Output.Write(line)
					updates <- Update{Port: i.Port, Type: UpdateNewOutput, Data: line}
				}
			}
		}
	}
}

// GetCommandString returns the full command as a string
func (i *Instance) GetCommandString() string {
	return "mlx-openai-server " + strings.Join(i.Args, " ")
}
