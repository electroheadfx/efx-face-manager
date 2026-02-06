package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Message represents a single chat message
type Message struct {
	Role      string    `json:"role"` // "user", "assistant", "system"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Conversation represents a chat conversation
type Conversation struct {
	ID        string    `json:"id"`    // "chat-001"
	Title     string    `json:"title"` // User or AI generated
	Port      int       `json:"port"`  // Server port
	Model     string    `json:"model"` // Model name
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetChatDir returns the chat storage directory path
func GetChatDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "efx-face-manager", "chat")
}

// EnsureChatDir creates the chat directory if it doesn't exist
func EnsureChatDir() error {
	return os.MkdirAll(GetChatDir(), 0755)
}

// NewConversation creates a new conversation with auto-generated ID
func NewConversation(port int, model string) (*Conversation, error) {
	if err := EnsureChatDir(); err != nil {
		return nil, fmt.Errorf("failed to create chat directory: %w", err)
	}

	id, err := NextConversationID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	conv := &Conversation{
		ID:        id,
		Title:     "New Conversation",
		Port:      port,
		Model:     model,
		Messages:  []Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	return conv, nil
}

// NextConversationID generates the next available conversation ID
func NextConversationID() (string, error) {
	dir := GetChatDir()

	// Ensure directory exists
	if err := EnsureChatDir(); err != nil {
		return "", err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "chat-001", nil // First conversation
	}

	maxNum := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "chat-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		// Extract number from chat-XXX.json
		numStr := strings.TrimPrefix(name, "chat-")
		numStr = strings.TrimSuffix(numStr, ".json")
		if num, err := strconv.Atoi(numStr); err == nil && num > maxNum {
			maxNum = num
		}
	}

	return fmt.Sprintf("chat-%03d", maxNum+1), nil
}

// ConversationPath returns the file path for a conversation
func ConversationPath(id string) string {
	return filepath.Join(GetChatDir(), id+".json")
}

// SaveConversation saves a conversation to disk
func SaveConversation(conv *Conversation) error {
	if err := EnsureChatDir(); err != nil {
		return fmt.Errorf("failed to create chat directory: %w", err)
	}

	conv.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	path := ConversationPath(conv.ID)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write conversation file: %w", err)
	}

	return nil
}

// LoadConversation loads a conversation from disk
func LoadConversation(id string) (*Conversation, error) {
	path := ConversationPath(id)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read conversation file: %w", err)
	}

	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversation: %w", err)
	}

	return &conv, nil
}

// ListConversations lists all conversations, optionally filtered by port
func ListConversations(port int) ([]Conversation, error) {
	dir := GetChatDir()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Conversation{}, nil
		}
		return nil, fmt.Errorf("failed to read chat directory: %w", err)
	}

	var conversations []Conversation
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		conv, err := LoadConversation(id)
		if err != nil {
			continue // Skip invalid files
		}

		// Filter by port if specified
		if port > 0 && conv.Port != port {
			continue
		}

		conversations = append(conversations, *conv)
	}

	// Sort by updated time, newest first
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].UpdatedAt.After(conversations[j].UpdatedAt)
	})

	return conversations, nil
}

// ListAllConversations lists all conversations regardless of port
func ListAllConversations() ([]Conversation, error) {
	return ListConversations(0)
}

// DeleteConversation deletes a conversation from disk
func DeleteConversation(id string) error {
	path := ConversationPath(id)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}
	return nil
}

// GetLatestConversation returns the most recent conversation for a port
func GetLatestConversation(port int) (*Conversation, error) {
	conversations, err := ListConversations(port)
	if err != nil {
		return nil, err
	}

	if len(conversations) == 0 {
		return nil, nil
	}

	return &conversations[0], nil
}

// AddMessage adds a message to the conversation and saves
func (c *Conversation) AddMessage(role, content string) {
	c.Messages = append(c.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}

// ToAPIMessages converts conversation messages to API format
func (c *Conversation) ToAPIMessages() []map[string]string {
	messages := make([]map[string]string, len(c.Messages))
	for i, msg := range c.Messages {
		messages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}
	return messages
}
