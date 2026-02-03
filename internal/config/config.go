package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	ExternalModelPath = "/Volumes/T7/mlx-server"
	LocalModelPath    = "~/mlx-server"
)

// Config holds the application configuration
type Config struct {
	Version        string            `json:"version"`
	ModelDir       string            `json:"modelDir"`
	AutoDetectPath bool              `json:"autoDetectPath"`
	DefaultPort    int               `json:"defaultPort"`
	DefaultHost    string            `json:"defaultHost"`
	LastUsed       LastUsedConfig    `json:"lastUsed"`
}

type LastUsedConfig struct {
	Model string `json:"model"`
	Type  string `json:"type"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Version:        "1.0.0",
		ModelDir:       DetectDefaultPath(),
		AutoDetectPath: true,
		DefaultPort:    8000,
		DefaultHost:    "0.0.0.0",
	}
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "efx-face-manager", "config.json")
}

// LegacyConfigPath returns the path to the legacy config file
func LegacyConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".efx-face-manager.conf")
}

// Load loads the configuration from disk
func Load() (*Config, error) {
	configPath := ConfigPath()
	
	// Try new config location first
	if data, err := os.ReadFile(configPath); err == nil {
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err == nil {
			return &cfg, nil
		}
	}
	
	// Try legacy config (just a path string)
	legacyPath := LegacyConfigPath()
	if data, err := os.ReadFile(legacyPath); err == nil {
		cfg := DefaultConfig()
		// Trim whitespace/newlines from legacy config
		cfg.ModelDir = strings.TrimSpace(string(data))
		cfg.AutoDetectPath = false
		return cfg, nil
	}
	
	// Return default config
	return DefaultConfig(), nil
}

// Save saves the configuration to disk
func (c *Config) Save() error {
	configPath := ConfigPath()
	
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0644)
}

// DetectDefaultPath detects the best model storage path
func DetectDefaultPath() string {
	// Check if external drive is mounted
	if _, err := os.Stat("/Volumes/T7"); err == nil {
		return ExternalModelPath
	}
	
	// Fall back to local path
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "mlx-server")
}

// ExpandPath expands ~ in paths
func ExpandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}

// DisplayPath returns a display-friendly path (with ~ for home)
func DisplayPath(path string) string {
	home, _ := os.UserHomeDir()
	if len(path) >= len(home) && path[:len(home)] == home {
		return "~" + path[len(home):]
	}
	return path
}

// IsExternalMounted checks if the external drive is mounted
func IsExternalMounted() bool {
	_, err := os.Stat("/Volumes/T7")
	return err == nil
}
