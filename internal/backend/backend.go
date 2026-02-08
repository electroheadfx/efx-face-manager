package backend

import (
	"fmt"
	"runtime"

	"github.com/lmarques/efx-face-manager/internal/model"
)

// BackendType represents the type of server backend
type BackendType string

const (
	BackendAuto   BackendType = "auto"
	BackendMLX    BackendType = "mlx"
	BackendOllama BackendType = "ollama"
)

// DetectionResult holds the detection status for a backend
type DetectionResult struct {
	Installed bool
	Path      string
	Version   string
	Error     error
}

// Backend defines the interface for a server backend
type Backend interface {
	// Name returns the display name of the backend
	Name() string
	// Type returns the backend type identifier
	Type() BackendType
	// Detect checks if the backend is installed and available
	Detect() DetectionResult
	// SupportedModelTypes returns the model types this backend supports
	SupportedModelTypes() []model.ModelType
	// SupportsModelType checks if a specific model type is supported
	SupportsModelType(t model.ModelType) bool
	// DefaultPort returns the default server port for this backend
	DefaultPort() int
	// Executable returns the CLI command name
	Executable() string
	// IsPerModelServer returns true if the backend runs one server per model (MLX)
	// vs a single shared server (Ollama)
	IsPerModelServer() bool
	// BuildEnv returns environment variables needed to start the server
	BuildEnv(modelDir string, host string, port int) []string
}

// Resolve selects the best backend based on config preference and platform.
// Returns the backend and nil error on success, or nil and an error if no
// suitable backend is found.
func Resolve(preference BackendType) (Backend, error) {
	switch preference {
	case BackendMLX:
		b := &MLXBackend{}
		if d := b.Detect(); !d.Installed {
			return nil, fmt.Errorf("mlx-openai-server not installed. Install with: pip install mlx-openai-server")
		}
		return b, nil

	case BackendOllama:
		b := &OllamaBackend{}
		if d := b.Detect(); !d.Installed {
			return nil, fmt.Errorf("Ollama not installed. Install from: https://ollama.com")
		}
		return b, nil

	default: // BackendAuto
		// On macOS, prefer MLX (better performance on Apple Silicon)
		if runtime.GOOS == "darwin" {
			mlx := &MLXBackend{}
			if d := mlx.Detect(); d.Installed {
				return mlx, nil
			}
		}
		// Fallback to Ollama (cross-platform)
		ollama := &OllamaBackend{}
		if d := ollama.Detect(); d.Installed {
			return ollama, nil
		}
		return nil, fmt.Errorf("no backend found. Install mlx-openai-server (macOS) or Ollama (https://ollama.com)")
	}
}

// DetectAll returns detection results for all known backends
func DetectAll() map[BackendType]DetectionResult {
	return map[BackendType]DetectionResult{
		BackendMLX:    (&MLXBackend{}).Detect(),
		BackendOllama: (&OllamaBackend{}).Detect(),
	}
}
