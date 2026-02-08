package opencode

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"github.com/lmarques/efx-face-manager/internal/config"
)

// DetectPlatform returns "unix" or "windows" based on runtime.GOOS
func DetectPlatform() string {
	if runtime.GOOS == "windows" {
		return "windows"
	}
	return "unix"
}

// ScriptExtension returns the appropriate script extension for the current platform
func ScriptExtension() string {
	if runtime.GOOS == "windows" {
		return ".bat"
	}
	return ".sh"
}

// SanitizeModelName cleans the model name for use in filenames
// Replaces special characters and converts to lowercase
func SanitizeModelName(modelName string) string {
	// Replace common separators
	name := strings.ReplaceAll(modelName, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")

	// Keep only safe characters
	var result strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '.' || r == '_' {
			result.WriteRune(unicode.ToLower(r))
		}
	}

	return result.String()
}

// RunnersDir returns the path to the runners directory
func RunnersDir() string {
	return filepath.Join(config.ConfigDir(), "runners")
}

// ScriptPath returns the full path for a runner script
func ScriptPath(modelName string, port int) string {
	sanitized := SanitizeModelName(modelName)
	filename := fmt.Sprintf("opencode-runner-%s-port%d%s", sanitized, port, ScriptExtension())
	return filepath.Join(RunnersDir(), filename)
}

// ScriptPathWeb returns the full path for a web runner script
func ScriptPathWeb(modelName string, port int, webPort int) string {
	sanitized := SanitizeModelName(modelName)
	filename := fmt.Sprintf("opencode-runner-%s-port%d-web%d%s", sanitized, port, webPort, ScriptExtension())
	return filepath.Join(RunnersDir(), filename)
}

// EnsureRunnersDir creates the runners directory if it doesn't exist
func EnsureRunnersDir() error {
	return os.MkdirAll(RunnersDir(), 0755)
}

// GenerateScriptContent creates the script content based on platform
func GenerateScriptContent(configJSON string, modelName string, platform string) string {
	if platform == "windows" {
		return generateWindowsScript(configJSON, modelName)
	}
	return generateUnixScript(configJSON, modelName)
}

// GenerateWebScriptContent creates the web script content based on platform
func GenerateWebScriptContent(configJSON string, modelName string, webPort int, platform string) string {
	if platform == "windows" {
		return generateWindowsWebScript(configJSON, modelName, webPort)
	}
	return generateUnixWebScript(configJSON, modelName, webPort)
}

// generateUnixScript creates a bash script for macOS/Linux
func generateUnixScript(configJSON string, modelName string) string {
	// Escape single quotes in JSON for bash
	escapedJSON := strings.ReplaceAll(configJSON, "'", "'\"'\"'")

	return fmt.Sprintf(`#!/bin/bash
export OPENCODE_CONFIG_CONTENT='%s'
exec opencode --model "mlx-community/%s"
`, escapedJSON, modelName)
}

// generateWindowsScript creates a batch script for Windows
func generateWindowsScript(configJSON string, modelName string) string {
	return fmt.Sprintf(`@echo off
set OPENCODE_CONFIG_CONTENT=%s
opencode --model "mlx-community/%s"
`, configJSON, modelName)
}

// generateUnixWebScript creates a bash web script for macOS/Linux
func generateUnixWebScript(configJSON string, modelName string, webPort int) string {
	// Escape single quotes in JSON for bash
	escapedJSON := strings.ReplaceAll(configJSON, "'", "'\"'\"'")

	return fmt.Sprintf(`#!/bin/bash
export OPENCODE_CONFIG_CONTENT='%s'
exec opencode web --port %d
`, escapedJSON, webPort)
}

// generateWindowsWebScript creates a batch web script for Windows
func generateWindowsWebScript(configJSON string, modelName string, webPort int) string {
	return fmt.Sprintf(`@echo off
set OPENCODE_CONFIG_CONTENT=%s
opencode web --port %d
`, configJSON, webPort)
}

// WriteScript writes the script to disk and sets execute permissions on Unix
func WriteScript(path string, content string) error {
	// Ensure directory exists
	if err := EnsureRunnersDir(); err != nil {
		return fmt.Errorf("failed to create runners directory: %w", err)
	}

	// Write the script file
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write script: %w", err)
	}

	return nil
}

// CreateRunnerScript is a convenience function that does the full flow:
// Load user config, merge with generated provider, create script, return path
func CreateRunnerScript(host string, port int, modelName string) (string, error) {
	// Load existing user config
	userConfig := LoadUserConfig()

	// Generate provider config
	providerConfig := GenerateProviderConfig(host, port, modelName)

	// Merge configs
	mergedConfig := MergeConfigs(userConfig, providerConfig)

	// Convert to JSON
	configJSON, err := ConfigToJSON(mergedConfig)
	if err != nil {
		return "", fmt.Errorf("failed to serialize config: %w", err)
	}

	// Generate script content
	platform := DetectPlatform()
	scriptContent := GenerateScriptContent(configJSON, modelName, platform)

	// Get script path
	scriptPath := ScriptPath(modelName, port)

	// Write script
	if err := WriteScript(scriptPath, scriptContent); err != nil {
		return "", err
	}

	return scriptPath, nil
}

// CreateWebRunnerScript creates a web runner script for opencode web
func CreateWebRunnerScript(host string, port int, modelName string, webPort int) (string, error) {
	// Load existing user config
	userConfig := LoadUserConfig()

	// Generate provider config
	providerConfig := GenerateProviderConfig(host, port, modelName)

	// Merge configs with model field set
	mergedConfig := MergeConfigsWithModel(userConfig, providerConfig, modelName)

	// Convert to JSON
	configJSON, err := ConfigToJSON(mergedConfig)
	if err != nil {
		return "", fmt.Errorf("failed to serialize config: %w", err)
	}

	// Generate web script content
	platform := DetectPlatform()
	scriptContent := GenerateWebScriptContent(configJSON, modelName, webPort, platform)

	// Get script path
	scriptPath := ScriptPathWeb(modelName, port, webPort)

	// Write script
	if err := WriteScript(scriptPath, scriptContent); err != nil {
		return "", err
	}

	return scriptPath, nil
}

// DisplayScriptPath returns a display-friendly path with ~ for home directory
func DisplayScriptPath(path string) string {
	return config.DisplayPath(path)
}
