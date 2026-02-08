package backend

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lmarques/efx-face-manager/internal/model"
)

// OllamaBackend implements Backend for Ollama
type OllamaBackend struct{}

func (b *OllamaBackend) Name() string {
	return "Ollama"
}

func (b *OllamaBackend) Type() BackendType {
	return BackendOllama
}

func (b *OllamaBackend) Detect() DetectionResult {
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

func (b *OllamaBackend) SupportedModelTypes() []model.ModelType {
	return []model.ModelType{
		model.TypeLM,
		model.TypeMultimodal,
		model.TypeEmbeddings,
	}
}

func (b *OllamaBackend) SupportsModelType(t model.ModelType) bool {
	for _, st := range b.SupportedModelTypes() {
		if st == t {
			return true
		}
	}
	return false
}

func (b *OllamaBackend) DefaultPort() int {
	return 11434
}

func (b *OllamaBackend) Executable() string {
	return "ollama"
}

func (b *OllamaBackend) IsPerModelServer() bool {
	return false
}

func (b *OllamaBackend) BuildEnv(modelDir string, host string, port int) []string {
	return []string{
		fmt.Sprintf("OLLAMA_MODELS=%s", filepath.Join(modelDir, "cache")),
		fmt.Sprintf("OLLAMA_HOST=%s:%d", host, port),
	}
}
