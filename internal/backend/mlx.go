package backend

import (
	"os/exec"
	"strings"

	"github.com/lmarques/efx-face-manager/internal/model"
)

// MLXBackend implements Backend for mlx-openai-server
type MLXBackend struct{}

func (b *MLXBackend) Name() string {
	return "MLX"
}

func (b *MLXBackend) Type() BackendType {
	return BackendMLX
}

func (b *MLXBackend) Detect() DetectionResult {
	result := DetectionResult{}

	path, err := exec.LookPath("mlx-openai-server")
	if err != nil {
		result.Error = err
		return result
	}

	result.Installed = true
	result.Path = path

	cmd := exec.Command("mlx-openai-server", "--version")
	output, err := cmd.Output()
	if err == nil {
		result.Version = strings.TrimSpace(string(output))
	}

	return result
}

func (b *MLXBackend) SupportedModelTypes() []model.ModelType {
	return []model.ModelType{
		model.TypeLM,
		model.TypeMultimodal,
		model.TypeImageGeneration,
		model.TypeImageEdit,
		model.TypeEmbeddings,
		model.TypeWhisper,
	}
}

func (b *MLXBackend) SupportsModelType(t model.ModelType) bool {
	for _, st := range b.SupportedModelTypes() {
		if st == t {
			return true
		}
	}
	return false
}

func (b *MLXBackend) DefaultPort() int {
	return 8000
}

func (b *MLXBackend) Executable() string {
	return "mlx-openai-server"
}

func (b *MLXBackend) IsPerModelServer() bool {
	return true
}

func (b *MLXBackend) BuildEnv(modelDir string, host string, port int) []string {
	// MLX inherits parent environment, no extra env vars needed
	return nil
}
