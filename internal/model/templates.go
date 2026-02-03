package model

// Template represents a predefined model configuration
type Template struct {
	Name             string
	ModelName        string
	ModelType        ModelType
	ReasoningParser  string
	ToolCallParser   string
	MessageConverter string
	TrustRemoteCode  bool
	Debug            bool
	Port             int
	Host             string
	Description      string
}

// DefaultTemplates returns the predefined model templates
func DefaultTemplates() []Template {
	return []Template{
		{
			Name:             "GLM-4.7-Flash-8bit",
			ModelName:        "GLM-4.7-Flash-8bit",
			ModelType:        TypeLM,
			ReasoningParser:  "glm47_flash",
			ToolCallParser:   "glm4_moe",
			MessageConverter: "glm4_moe",
			Debug:            true,
			Port:             8000,
			Host:             "0.0.0.0",
			Description:      "reasoning+tools",
		},
		{
			Name:             "Qwen3-Coder-30B-A3B-Instruct-8bit",
			ModelName:        "Qwen3-Coder-30B-A3B-Instruct-8bit",
			ModelType:        TypeLM,
			ToolCallParser:   "qwen3_coder",
			MessageConverter: "qwen3_coder",
			Port:             8000,
			Host:             "0.0.0.0",
			Description:      "code+tools",
		},
		{
			Name:             "NVIDIA-Nemotron-3-Nano-30B-A3B-MLX-8Bit",
			ModelName:        "NVIDIA-Nemotron-3-Nano-30B-A3B-MLX-8Bit",
			ModelType:        TypeLM,
			ToolCallParser:   "qwen3",
			MessageConverter: "nemotron3_nano",
			TrustRemoteCode:  true,
			Port:             8000,
			Host:             "0.0.0.0",
			Description:      "tools",
		},
		{
			Name:             "Qwen3-VL-8B-Thinking-8bit",
			ModelName:        "Qwen3-VL-8B-Thinking-8bit",
			ModelType:        TypeMultimodal,
			Port:             8000,
			Host:             "0.0.0.0",
			Description:      "vision+thinking",
		},
	}
}

// GetTemplate returns a template by name
func GetTemplate(name string) *Template {
	for _, t := range DefaultTemplates() {
		if t.Name == name {
			return &t
		}
	}
	return nil
}
