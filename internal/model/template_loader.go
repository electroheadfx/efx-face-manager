package model

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// TemplateConfig represents the structure of templates.yaml
type TemplateConfig struct {
	Templates []Template `yaml:"templates"`
}

// LoadTemplates loads templates from ~/.config/efx-face-manager/templates.yaml
// Falls back to default templates if file doesn't exist
func LoadTemplates() ([]Template, error) {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "efx-face-manager")
	templateFile := filepath.Join(configDir, "templates.yaml")
	
	// Check if file exists
	if _, err := os.Stat(templateFile); os.IsNotExist(err) {
		// Return default templates if no custom file
		return DefaultTemplates(), nil
	}
	
	// Read file
	data, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, err
	}
	
	// Parse YAML
	var config TemplateConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	// Merge with defaults - custom templates override defaults with same names
	defaultTemplates := DefaultTemplates()
	customTemplates := make(map[string]Template)
	
	// Index custom templates by name
	for _, tmpl := range config.Templates {
		customTemplates[tmpl.Name] = tmpl
	}
	
	// Build final list: custom templates first, then defaults not overridden
	var result []Template
	
	// Add all custom templates
	for _, tmpl := range config.Templates {
		result = append(result, tmpl)
	}
	
	// Add default templates that aren't overridden
	for _, defaultTmpl := range defaultTemplates {
		if _, exists := customTemplates[defaultTmpl.Name]; !exists {
			result = append(result, defaultTmpl)
		}
	}
	
	return result, nil
}

// SaveTemplates saves templates to ~/.config/efx-face-manager/templates.yaml
func SaveTemplates(templates []Template) error {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "efx-face-manager")
	templateFile := filepath.Join(configDir, "templates.yaml")
	
	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	
	config := TemplateConfig{
		Templates: templates,
	}
	
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	
	return os.WriteFile(templateFile, data, 0644)
}