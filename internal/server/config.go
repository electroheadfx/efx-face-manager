package server

import (
	"fmt"
	"path/filepath"

	"github.com/lmarques/efx-face-manager/internal/backend"
	"github.com/lmarques/efx-face-manager/internal/model"
)

// Config holds server configuration
type Config struct {
	Backend          backend.BackendType
	Model            string
	ModelPath        string
	Type             model.ModelType
	Port             int
	Host             string
	ContextLength    int
	ToolCallParser   string
	ReasoningParser  string
	MessageConverter string
	TrustRemoteCode  bool
	Debug            bool
	DisableAutoResize bool
	ChatTemplateFile string
	LogLevel         string
	
	// Image generation/edit specific (MLX only)
	ConfigName  string
	Quantize    int
	LoraPaths   string
	LoraScales  string
	
	// Whisper/embeddings specific (MLX only)
	MaxConcurrency int
	QueueTimeout   int
	QueueSize      int
}

// NewConfig creates a new server config with defaults
func NewConfig() Config {
	return Config{
		Backend:        backend.BackendMLX,
		Port:           8000,
		Host:           "0.0.0.0",
		Type:           model.TypeLM,
		MaxConcurrency: 1,
		QueueTimeout:   300,
		QueueSize:      100,
	}
}

// NewConfigForBackend creates a new server config with backend-appropriate defaults
func NewConfigForBackend(bt backend.BackendType) Config {
	cfg := NewConfig()
	cfg.Backend = bt
	if bt == backend.BackendOllama {
		cfg.Port = 11434
	}
	return cfg
}

// BuildArgs builds the command line arguments for the configured backend
func (c *Config) BuildArgs() []string {
	if c.Backend == backend.BackendOllama {
		return c.buildOllamaArgs()
	}
	return c.buildMLXArgs()
}

// buildMLXArgs builds mlx-openai-server arguments (existing logic)
func (c *Config) buildMLXArgs() []string {
	args := []string{"launch"}
	
	args = append(args, "--model-path", c.ModelPath)
	args = append(args, "--model-type", string(c.Type))
	args = append(args, "--port", fmt.Sprintf("%d", c.Port))
	args = append(args, "--host", c.Host)
	
	switch c.Type {
	case model.TypeLM, model.TypeMultimodal:
		if c.ContextLength > 0 {
			args = append(args, "--context-length", fmt.Sprintf("%d", c.ContextLength))
		}
		if c.ToolCallParser != "" {
			args = append(args, "--tool-call-parser", c.ToolCallParser)
		}
		if c.ReasoningParser != "" {
			args = append(args, "--reasoning-parser", c.ReasoningParser)
		}
		if c.MessageConverter != "" {
			args = append(args, "--message-converter", c.MessageConverter)
		}
		if c.TrustRemoteCode {
			args = append(args, "--trust-remote-code")
		}
		if c.Debug {
			args = append(args, "--debug")
		}
		if c.ChatTemplateFile != "" {
			args = append(args, "--chat-template-file", c.ChatTemplateFile)
		}
		if c.Type == model.TypeMultimodal && c.DisableAutoResize {
			args = append(args, "--disable-auto-resize")
		}
		
	case model.TypeImageGeneration, model.TypeImageEdit:
		if c.ConfigName != "" {
			args = append(args, "--config-name", c.ConfigName)
		}
		if c.Quantize > 0 {
			args = append(args, "--quantize", fmt.Sprintf("%d", c.Quantize))
		}
		if c.LoraPaths != "" {
			args = append(args, "--lora-paths", c.LoraPaths)
		}
		if c.LoraScales != "" {
			args = append(args, "--lora-scales", c.LoraScales)
		}
		
	case model.TypeWhisper, model.TypeEmbeddings:
		args = append(args, "--max-concurrency", fmt.Sprintf("%d", c.MaxConcurrency))
		args = append(args, "--queue-timeout", fmt.Sprintf("%d", c.QueueTimeout))
		args = append(args, "--queue-size", fmt.Sprintf("%d", c.QueueSize))
	}
	
	if c.LogLevel != "" && c.LogLevel != "INFO" {
		args = append(args, "--log-level", c.LogLevel)
	}
	
	return args
}

// buildOllamaArgs builds Ollama serve arguments
func (c *Config) buildOllamaArgs() []string {
	return []string{"serve"}
}

// BuildEnv returns environment variables needed for the server process
func (c *Config) BuildEnv(modelDir string) []string {
	if c.Backend == backend.BackendOllama {
		env := []string{
			fmt.Sprintf("OLLAMA_MODELS=%s", filepath.Join(modelDir, "cache")),
			fmt.Sprintf("OLLAMA_HOST=%s:%d", c.Host, c.Port),
		}
		if c.Debug {
			env = append(env, "OLLAMA_DEBUG=1")
		}
		return env
	}
	return nil
}

// Executable returns the backend executable name
func (c *Config) Executable() string {
	if c.Backend == backend.BackendOllama {
		return "ollama"
	}
	return "mlx-openai-server"
}

// FromTemplate creates a config from a model template
func FromTemplate(t *model.Template, modelDir string, activeBackend backend.BackendType) Config {
	bt := activeBackend
	if t.Backend != "" {
		bt = backend.BackendType(t.Backend)
	}

	port := t.Port
	if port == 0 {
		if bt == backend.BackendOllama {
			port = 11434
		} else {
			port = 8000
		}
	}

	host := t.Host
	if host == "" {
		host = "0.0.0.0"
	}

	return Config{
		Backend:          bt,
		Model:            t.ModelName,
		ModelPath:        modelDir + "/" + t.ModelName,
		Type:             t.ModelType,
		Port:             port,
		Host:             host,
		ReasoningParser:  t.ReasoningParser,
		ToolCallParser:   t.ToolCallParser,
		MessageConverter: t.MessageConverter,
		TrustRemoteCode:  t.TrustRemoteCode,
		Debug:            t.Debug,
	}
}
