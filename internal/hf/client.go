package hf

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

const (
	baseURL = "https://huggingface.co/api/models"
)

// Model represents a HuggingFace model
type Model struct {
	ID           string `json:"id"`
	Author       string `json:"author"`
	ModelID      string `json:"modelId"`
	Downloads    int    `json:"downloads"`
	Likes        int    `json:"likes"`
	PipelineTag  string `json:"pipeline_tag"`
	LibraryName  string `json:"library_name"`
	LastModified string `json:"lastModified"`
	Private      bool   `json:"private"`
}

// Client is the HuggingFace API client
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new HuggingFace API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Search searches for models on HuggingFace
func (c *Client) Search(query string, author string, limit int) ([]Model, error) {
	params := url.Values{}
	
	if query != "" {
		params.Set("search", query)
	}
	if author != "" {
		params.Set("author", author)
	}
	params.Set("sort", "downloads")
	params.Set("direction", "-1")
	params.Set("limit", fmt.Sprintf("%d", limit))
	// Filter for MLX models
	params.Set("library", "mlx")
	
	reqURL := baseURL + "?" + params.Encode()
	
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}
	
	var models []Model
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return models, nil
}

// GetModel gets a specific model by ID
func (c *Client) GetModel(modelID string) (*Model, error) {
	reqURL := baseURL + "/" + url.PathEscape(modelID)
	
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch model: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}
	
	var model Model
	if err := json.NewDecoder(resp.Body).Decode(&model); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &model, nil
}

// Download downloads a model using the hf CLI
func (c *Client) Download(modelID string, cacheDir string) error {
	// Check if hf CLI is available (preferred)
	hfCmd := "hf"
	if _, err := exec.LookPath("hf"); err != nil {
		// Try huggingface-cli as fallback
		if _, err := exec.LookPath("huggingface-cli"); err != nil {
			return fmt.Errorf("hf CLI not found. Install with: pip install huggingface_hub")
		}
		hfCmd = "huggingface-cli"
	}

	// Use hf download with cache-dir
	var cmd *exec.Cmd
	if hfCmd == "hf" {
		cmd = exec.Command(hfCmd, "download", modelID, "--cache-dir", cacheDir+"/cache", "--no-quiet")
	} else {
		cmd = exec.Command(hfCmd, "download", modelID, "--cache-dir", cacheDir+"/cache")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("download failed: %w\n%s", err, string(output))
	}

	return nil
}

// DownloadWithProgress downloads a model and returns progress updates via channel
func (c *Client) DownloadWithProgress(modelID string, cacheDir string) (<-chan string, <-chan error) {
	progressCh := make(chan string, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(progressCh)
		defer close(errCh)

		progressCh <- fmt.Sprintf("Starting download: %s", modelID)

		// Check if hf CLI is available (preferred)
		hfCmd := "hf"
		if _, err := exec.LookPath("hf"); err != nil {
			if _, err := exec.LookPath("huggingface-cli"); err != nil {
				errCh <- fmt.Errorf("hf CLI not found. Install with: pip install huggingface_hub")
				return
			}
			hfCmd = "huggingface-cli"
		}

		// Use hf download with cache-dir
		var cmd *exec.Cmd
		if hfCmd == "hf" {
			cmd = exec.Command(hfCmd, "download", modelID, "--cache-dir", cacheDir+"/cache", "--no-quiet")
		} else {
			cmd = exec.Command(hfCmd, "download", modelID, "--cache-dir", cacheDir+"/cache")
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errCh <- err
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			errCh <- err
			return
		}

		if err := cmd.Start(); err != nil {
			errCh <- err
			return
		}

		// Read output
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := stdout.Read(buf)
				if n > 0 {
					progressCh <- strings.TrimSpace(string(buf[:n]))
				}
				if err != nil {
					break
				}
			}
		}()

		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := stderr.Read(buf)
				if n > 0 {
					progressCh <- strings.TrimSpace(string(buf[:n]))
				}
				if err != nil {
					break
				}
			}
		}()

		if err := cmd.Wait(); err != nil {
			errCh <- fmt.Errorf("download failed: %w", err)
			return
		}

		progressCh <- "Download complete!"
	}()

	return progressCh, errCh
}

// FormatDownloads formats download count for display
func FormatDownloads(downloads int) string {
	if downloads >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(downloads)/1000000)
	}
	if downloads >= 1000 {
		return fmt.Sprintf("%.1fk", float64(downloads)/1000)
	}
	return fmt.Sprintf("%d", downloads)
}
