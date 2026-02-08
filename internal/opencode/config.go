package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// OpencodeConfigPath returns the path to the user's opencode config
func OpencodeConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "opencode", "opencode.json")
}

// LoadUserConfig loads the existing opencode config from ~/.config/opencode/opencode.json
// Returns an empty map if the file doesn't exist or is invalid
func LoadUserConfig() map[string]interface{} {
	configPath := OpencodeConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		// File doesn't exist or can't be read - return empty map
		return make(map[string]interface{})
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		// Invalid JSON - return empty map
		return make(map[string]interface{})
	}

	return config
}

// GenerateProviderConfig creates the mlx-community provider configuration
func GenerateProviderConfig(host string, port int, modelName string) map[string]interface{} {
	return map[string]interface{}{
		"mlx-community": map[string]interface{}{
			"npm":  "@ai-sdk/openai-compatible",
			"name": "local-mlx-model",
			"options": map[string]interface{}{
				"baseURL": "http://" + host + ":" + itoa(port) + "/v1",
			},
			"models": map[string]interface{}{
				modelName: map[string]interface{}{
					"name": "mlx-server/" + modelName,
				},
			},
		},
	}
}

// MergeConfigs merges the user config with the generated provider config
// The provider.mlx-community section is replaced with the generated config
func MergeConfigs(userConfig map[string]interface{}, providerConfig map[string]interface{}) map[string]interface{} {
	// If user config is empty, just wrap provider config
	if len(userConfig) == 0 {
		return map[string]interface{}{
			"provider": providerConfig,
		}
	}

	// Clone user config
	result := make(map[string]interface{})
	for k, v := range userConfig {
		result[k] = v
	}

	// Get or create provider section
	providerSection, ok := result["provider"].(map[string]interface{})
	if !ok {
		providerSection = make(map[string]interface{})
	}

	// Clone provider section to avoid modifying original
	newProviderSection := make(map[string]interface{})
	for k, v := range providerSection {
		newProviderSection[k] = v
	}

	// Merge in the mlx-community provider
	for k, v := range providerConfig {
		newProviderSection[k] = v
	}

	result["provider"] = newProviderSection
	return result
}

// ConfigToJSON serializes the config to a JSON string
func ConfigToJSON(config map[string]interface{}) (string, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// itoa converts int to string without importing strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
