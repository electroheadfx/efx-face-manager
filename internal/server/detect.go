package server

import (
	"os/exec"
	"strings"

	"github.com/lmarques/efx-face-manager/internal/backend"
)

// DetectionResult holds the detection status for a tool
type DetectionResult struct {
	Installed bool
	Path      string
	Version   string
	Error     error
}

// Detect checks if mlx-openai-server is installed and available
func Detect() DetectionResult {
	return DetectMLX()
}

// DetectMLX checks if mlx-openai-server is installed and available
func DetectMLX() DetectionResult {
	result := DetectionResult{}

	// Try to find mlx-openai-server in PATH
	path, err := exec.LookPath("mlx-openai-server")
	if err != nil {
		result.Error = err
		return result
	}

	result.Installed = true
	result.Path = path

	// Try to get version
	cmd := exec.Command("mlx-openai-server", "--version")
	output, err := cmd.Output()
	if err == nil {
		result.Version = strings.TrimSpace(string(output))
	}

	return result
}

// DetectOllama checks if Ollama is installed and available
func DetectOllama() DetectionResult {
	result := DetectionResult{}

	path, err := exec.LookPath("ollama")
	if err != nil {
		result.Error = err
		return result
	}

	result.Installed = true
	result.Path = path

	cmd := exec.Command("ollama", "--version")
	output, err := cmd.Output()
	if err == nil {
		result.Version = strings.TrimSpace(string(output))
	}

	return result
}

// DetectHFCLI checks if huggingface-cli is installed
func DetectHFCLI() DetectionResult {
	result := DetectionResult{}

	// Try huggingface-cli first
	path, err := exec.LookPath("huggingface-cli")
	if err == nil {
		result.Installed = true
		result.Path = path
		
		// Try to get version
		cmd := exec.Command("huggingface-cli", "version")
		output, err := cmd.Output()
		if err == nil {
			result.Version = strings.TrimSpace(string(output))
		}
		return result
	}

	// Fallback to hf command
	path, err = exec.LookPath("hf")
	if err == nil {
		result.Installed = true
		result.Path = path
		return result
	}

	result.Error = err
	return result
}

// RequiredTools returns a summary of all tools and their status
func RequiredTools() map[string]DetectionResult {
	return map[string]DetectionResult{
		"mlx-openai-server": DetectMLX(),
		"ollama":            DetectOllama(),
		"huggingface-cli":   DetectHFCLI(),
	}
}

// CheckRequirements returns any missing required tools based on the active backend.
// If backendType is empty or "auto", it checks for at least one backend.
func CheckRequirements(backendType string) []string {
	missing := []string{}
	
	bt := backend.BackendType(backendType)
	if bt == "" {
		bt = backend.BackendAuto
	}

	switch bt {
	case backend.BackendMLX:
		if !DetectMLX().Installed {
			missing = append(missing, "mlx-openai-server (pip install mlx-openai-server)")
		}
		if !DetectHFCLI().Installed {
			missing = append(missing, "huggingface-cli (pip install huggingface_hub)")
		}
	case backend.BackendOllama:
		if !DetectOllama().Installed {
			missing = append(missing, "ollama (https://ollama.com)")
		}
	default: // auto
		mlx := DetectMLX()
		ollama := DetectOllama()
		if !mlx.Installed && !ollama.Installed {
			missing = append(missing, "mlx-openai-server (pip install mlx-openai-server) or ollama (https://ollama.com)")
		}
	}
	
	return missing
}
