package chat

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Runner executes opencode run commands for chat
type Runner struct {
	Host      string
	Port      int
	ModelName string
}

// NewRunner creates a new chat runner
func NewRunner(host string, port int, modelName string) *Runner {
	return &Runner{
		Host:      host,
		Port:      port,
		ModelName: modelName,
	}
}

// generateConfig creates the OPENCODE_CONFIG_CONTENT JSON
func (r *Runner) generateConfig() string {
	return fmt.Sprintf(`{"provider":{"mlx-community":{"npm":"@ai-sdk/openai-compatible","name":"local-mlx-model","options":{"baseURL":"http://%s:%d/v1"},"models":{"%s":{"name":"mlx-server/%s"}}}}}`,
		r.Host, r.Port, r.ModelName, r.ModelName)
}

// BuildPromptWithHistory builds a prompt that includes conversation history
func BuildPromptWithHistory(messages []Message, newMessage string) string {
	if len(messages) == 0 {
		return newMessage
	}

	var sb strings.Builder
	sb.WriteString("Previous conversation:\n")
	for _, msg := range messages {
		role := "User"
		if msg.Role == "assistant" {
			role = "Assistant"
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}
	sb.WriteString("\nCurrent message:\n")
	sb.WriteString(fmt.Sprintf("User: %s", newMessage))
	return sb.String()
}

// Run executes opencode run with the given prompt and returns the response
func (r *Runner) Run(prompt string) (string, error) {
	configJSON := r.generateConfig()

	cmd := exec.Command("opencode", "run", prompt, "--model", fmt.Sprintf("mlx-community/%s", r.ModelName))
	cmd.Env = append(os.Environ(), "OPENCODE_CONFIG_CONTENT="+configJSON)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("opencode error: %w\n%s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// RunAsync executes opencode run asynchronously and returns channels for response and error
func (r *Runner) RunAsync(prompt string) (<-chan string, <-chan error) {
	responseCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(responseCh)
		defer close(errCh)

		response, err := r.Run(prompt)
		if err != nil {
			errCh <- err
			return
		}
		responseCh <- response
	}()

	return responseCh, errCh
}
