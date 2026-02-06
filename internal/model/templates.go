package model

// Template represents a predefined model configuration
type Template struct {
	Name             string   `yaml:"name"`
	ModelName        string   `yaml:"model_name"`
	ModelType        ModelType `yaml:"model_type"`
	ReasoningParser  string   `yaml:"reasoning_parser,omitempty"`
	ToolCallParser   string   `yaml:"tool_call_parser,omitempty"`
	MessageConverter string   `yaml:"message_converter,omitempty"`
	TrustRemoteCode  bool     `yaml:"trust_remote_code,omitempty"`
	Debug            bool     `yaml:"debug,omitempty"`
	Port             int      `yaml:"port,omitempty"`
	Host             string   `yaml:"host,omitempty"`
	Description      string   `yaml:"description,omitempty"`
}

// DefaultTemplates returns an empty slice - all templates now come from YAML config
func DefaultTemplates() []Template {
	return []Template{}
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
