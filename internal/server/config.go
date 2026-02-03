package server

import (
	"fmt"

	"github.com/lmarques/efx-face-manager/internal/model"
)

// Config holds server configuration
type Config struct {
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
	
	// Image generation/edit specific
	ConfigName  string
	Quantize    int
	LoraPaths   string
	LoraScales  string
	
	// Whisper/embeddings specific
	MaxConcurrency int
	QueueTimeout   int
	QueueSize      int
}

// NewConfig creates a new server config with defaults
func NewConfig() Config {
	return Config{
		Port:           8000,
		Host:           "0.0.0.0",
		Type:           model.TypeLM,
		MaxConcurrency: 1,
		QueueTimeout:   300,
		QueueSize:      100,
	}
}

// BuildArgs builds the command line arguments
func (c *Config) BuildArgs() []string {
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

// FromTemplate creates a config from a model template
func FromTemplate(t *model.Template, modelDir string) Config {
	return Config{
		Model:            t.ModelName,
		ModelPath:        modelDir + "/" + t.ModelName,
		Type:             t.ModelType,
		Port:             t.Port,
		Host:             t.Host,
		ReasoningParser:  t.ReasoningParser,
		ToolCallParser:   t.ToolCallParser,
		MessageConverter: t.MessageConverter,
		TrustRemoteCode:  t.TrustRemoteCode,
		Debug:            t.Debug,
	}
}
