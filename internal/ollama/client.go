package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// Model represents a locally installed Ollama model
type Model struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	ModifiedAt time.Time `json:"modified_at"`
}

// LibraryModel represents a model available in the Ollama library
type LibraryModel struct {
	Name        string
	Description string
	Pulls       int
}

// PullProgress represents progress during a model pull
type PullProgress struct {
	Status    string `json:"status"`
	Digest    string `json:"digest"`
	Total     int64  `json:"total"`
	Completed int64  `json:"completed"`
}

// ModelInfo represents detailed information about a model
type ModelInfo struct {
	ModelFile  string `json:"modelfile"`
	Parameters string `json:"parameters"`
	Template   string `json:"template"`
}

// Client wraps Ollama CLI and API operations
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Ollama client with default settings
func NewClient() *Client {
	return &Client{
		baseURL: "http://localhost:11434",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithHost creates an Ollama client pointing to a specific host:port
func NewClientWithHost(host string, port int) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// List returns all locally installed Ollama models via API
func (c *Client) List() ([]Model, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/tags")
	if err != nil {
		// Fallback to CLI if API is unavailable
		return c.listViaCLI()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.listViaCLI()
	}

	var result struct {
		Models []Model `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	return result.Models, nil
}

// listViaCLI lists models using the ollama CLI as fallback
func (c *Client) listViaCLI() ([]Model, error) {
	cmd := exec.Command("ollama", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Ollama models: %w", err)
	}

	var models []Model
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	// Skip header line
	for i := 1; i < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if len(fields) >= 1 {
			models = append(models, Model{
				Name: fields[0],
			})
		}
	}

	return models, nil
}

// Pull downloads a model using the Ollama CLI (blocking)
func (c *Client) Pull(name string) error {
	cmd := exec.Command("ollama", "pull", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ollama pull failed: %w: %s", err, string(output))
	}
	return nil
}

// PullWithProgress pulls a model via API with streaming progress
func (c *Client) PullWithProgress(name string) (chan PullProgress, chan error) {
	progressCh := make(chan PullProgress, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(progressCh)
		defer close(errCh)

		body, _ := json.Marshal(map[string]interface{}{
			"name":   name,
			"stream": true,
		})

		client := &http.Client{} // No timeout for long pulls
		resp, err := client.Post(c.baseURL+"/api/pull", "application/json", bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("failed to start pull: %w", err)
			return
		}
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			var progress PullProgress
			if err := decoder.Decode(&progress); err != nil {
				if err == io.EOF {
					return
				}
				errCh <- fmt.Errorf("failed to read progress: %w", err)
				return
			}
			progressCh <- progress
			if progress.Status == "success" {
				return
			}
		}
	}()

	return progressCh, errCh
}

// Delete removes a model from Ollama
func (c *Client) Delete(name string) error {
	body, _ := json.Marshal(map[string]string{"name": name})
	req, err := http.NewRequest("DELETE", c.baseURL+"/api/delete", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Fallback to CLI
		cmd := exec.Command("ollama", "rm", name)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ollama rm failed: %w: %s", err, string(output))
		}
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete model: HTTP %d", resp.StatusCode)
	}

	return nil
}

// Show returns detailed information about a model
func (c *Client) Show(name string) (*ModelInfo, error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	resp, err := c.httpClient.Post(c.baseURL+"/api/show", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to show model info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get model info: HTTP %d", resp.StatusCode)
	}

	var info ModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse model info: %w", err)
	}

	return &info, nil
}

// SearchLibrary searches the Ollama library for available models.
// Falls back to a curated list of popular models if the registry is unreachable.
func (c *Client) SearchLibrary(query string, limit int) ([]LibraryModel, error) {
	models, err := c.fetchLibrary(query)
	if err != nil {
		// Fallback to curated popular models list
		models = popularModels()
	}

	// Apply query filter if fetched from fallback
	if query != "" {
		query = strings.ToLower(query)
		var filtered []LibraryModel
		for _, m := range models {
			if strings.Contains(strings.ToLower(m.Name), query) ||
				strings.Contains(strings.ToLower(m.Description), query) {
				filtered = append(filtered, m)
			}
		}
		models = filtered
	}

	if limit > 0 && len(models) > limit {
		models = models[:limit]
	}

	return models, nil
}

// fetchLibrary attempts to fetch models from the Ollama registry
func (c *Client) fetchLibrary(query string) ([]LibraryModel, error) {
	url := "https://ollama.com/api/tags"
	if query != "" {
		url = fmt.Sprintf("https://ollama.com/api/tags?q=%s", query)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var models []LibraryModel
	for _, m := range result.Models {
		models = append(models, LibraryModel{
			Name:        m.Name,
			Description: m.Description,
		})
	}

	return models, nil
}

// popularModels returns a curated list of popular Ollama models as fallback
func popularModels() []LibraryModel {
	return []LibraryModel{
		{Name: "llama3.3", Description: "Meta's Llama 3.3 70B model"},
		{Name: "llama3.2", Description: "Meta's Llama 3.2 (1B/3B)"},
		{Name: "llama3.1", Description: "Meta's Llama 3.1 (8B/70B/405B)"},
		{Name: "qwen2.5", Description: "Alibaba's Qwen 2.5 (0.5B-72B)"},
		{Name: "qwen2.5-coder", Description: "Qwen 2.5 optimized for code"},
		{Name: "deepseek-r1", Description: "DeepSeek R1 reasoning model"},
		{Name: "phi4", Description: "Microsoft Phi-4 (14B)"},
		{Name: "mistral", Description: "Mistral 7B"},
		{Name: "mixtral", Description: "Mistral Mixtral 8x7B MoE"},
		{Name: "gemma2", Description: "Google Gemma 2 (2B/9B/27B)"},
		{Name: "codellama", Description: "Meta's Code Llama"},
		{Name: "llava", Description: "LLaVA multimodal model"},
		{Name: "llama3.2-vision", Description: "Llama 3.2 Vision (11B/90B)"},
		{Name: "nomic-embed-text", Description: "Nomic text embeddings"},
		{Name: "mxbai-embed-large", Description: "mxbai embedding model"},
		{Name: "starcoder2", Description: "StarCoder2 code model"},
		{Name: "command-r", Description: "Cohere Command R"},
		{Name: "dolphin-mixtral", Description: "Dolphin Mixtral uncensored"},
		{Name: "solar", Description: "Upstage Solar 10.7B"},
		{Name: "yi", Description: "01.AI Yi (6B/34B)"},
	}
}

// FormatSize formats a byte count into a human-readable string
func FormatSize(bytes int64) string {
	const (
		gb = 1024 * 1024 * 1024
		mb = 1024 * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.0fMB", float64(bytes)/float64(mb))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// IsServerRunning checks if the Ollama server is responding
func (c *Client) IsServerRunning() bool {
	resp, err := c.httpClient.Get(c.baseURL + "/api/tags")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
