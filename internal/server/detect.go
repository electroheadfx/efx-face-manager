package server

import (
	"os/exec"
	"strings"
)

// DetectionResult holds the mlx-openai-server detection status
type DetectionResult struct {
	Installed bool
	Path      string
	Version   string
	Error     error
}

// Detect checks if mlx-openai-server is installed and available
func Detect() DetectionResult {
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

// RequiredTools returns a summary of all required tools and their status
func RequiredTools() map[string]DetectionResult {
	return map[string]DetectionResult{
		"mlx-openai-server": Detect(),
		"huggingface-cli":   DetectHFCLI(),
	}
}

// CheckRequirements returns any missing required tools
func CheckRequirements() []string {
	missing := []string{}
	
	tools := RequiredTools()
	if !tools["mlx-openai-server"].Installed {
		missing = append(missing, "mlx-openai-server (pip install mlx-openai-server)")
	}
	if !tools["huggingface-cli"].Installed {
		missing = append(missing, "huggingface-cli (pip install huggingface_hub)")
	}
	
	return missing
}
