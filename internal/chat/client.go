package chat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is an OpenAI-compatible API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	port       int
}

// ChatRequest represents the request body for chat completions
type ChatRequest struct {
	Model       string              `json:"model,omitempty"`
	Messages    []map[string]string `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

// ChatResponse represents a non-streaming response
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// StreamChunk represents a single streaming chunk
type StreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// NewClient creates a new API client for the given port
func NewClient(port int) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://localhost:%d/v1", port),
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Long timeout for streaming
		},
		port: port,
	}
}

// GetPort returns the port this client is connected to
func (c *Client) GetPort() int {
	return c.port
}

// StreamChatCompletion sends a streaming chat completion request
// Returns channels for content chunks and errors
func (c *Client) StreamChatCompletion(messages []map[string]string) (<-chan string, <-chan error, func()) {
	contentCh := make(chan string, 100)
	errCh := make(chan error, 1)
	done := make(chan struct{})

	cancel := func() {
		close(done)
	}

	go func() {
		defer close(contentCh)
		defer close(errCh)

		reqBody := ChatRequest{
			Messages: messages,
			Stream:   true,
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			errCh <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		req, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
		if err != nil {
			errCh <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			errCh <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
			return
		}

		reader := bufio.NewReader(resp.Body)
		for {
			select {
			case <-done:
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				// EOF or unexpected EOF means stream ended - this is normal
				if err == io.EOF || strings.Contains(err.Error(), "EOF") {
					return
				}
				errCh <- fmt.Errorf("failed to read stream: %w", err)
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// SSE format: "data: {json}" or "data: [DONE]"
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}

			var chunk StreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				// Skip malformed chunks
				continue
			}

			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				select {
				case contentCh <- chunk.Choices[0].Delta.Content:
				case <-done:
					return
				}
			}

			// Check for finish reason
			if len(chunk.Choices) > 0 && chunk.Choices[0].FinishReason != nil {
				return
			}
		}
	}()

	return contentCh, errCh, cancel
}

// ChatCompletion sends a non-streaming chat completion request
func (c *Client) ChatCompletion(messages []map[string]string) (*ChatResponse, error) {
	reqBody := ChatRequest{
		Messages: messages,
		Stream:   false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// GenerateTitle asks the LLM to generate a title for the conversation
func (c *Client) GenerateTitle(messages []map[string]string) (string, error) {
	if len(messages) == 0 {
		return "New Conversation", nil
	}

	// Build a summary request
	titlePrompt := []map[string]string{
		{
			"role":    "system",
			"content": "Generate a very short title (3-6 words max) for this conversation. Reply with ONLY the title, nothing else.",
		},
	}

	// Add context from the conversation (first few messages)
	maxMessages := 4
	if len(messages) < maxMessages {
		maxMessages = len(messages)
	}
	for i := 0; i < maxMessages; i++ {
		titlePrompt = append(titlePrompt, messages[i])
	}

	resp, err := c.ChatCompletion(titlePrompt)
	if err != nil {
		return "New Conversation", err
	}

	if len(resp.Choices) > 0 {
		title := strings.TrimSpace(resp.Choices[0].Message.Content)
		// Clean up common issues
		title = strings.Trim(title, "\"'")
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		if title != "" {
			return title, nil
		}
	}

	return "New Conversation", nil
}

// CheckConnection checks if the server is reachable
func (c *Client) CheckConnection() error {
	req, err := http.NewRequest("GET", c.baseURL+"/models", nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("server not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}
