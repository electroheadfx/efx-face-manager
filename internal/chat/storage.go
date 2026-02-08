package chat

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Message represents a single chat message
type Message struct {
	Role      string    `json:"role"`      // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Conversation represents a chat conversation
type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	Port      int       `json:"port"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Storage handles chat conversation persistence
type Storage struct {
	baseDir string
}

// NewStorage creates a new storage instance
func NewStorage() *Storage {
	home, _ := os.UserHomeDir()
	return &Storage{
		baseDir: filepath.Join(home, ".config", "efx-face-manager", "chats"),
	}
}

// ensureDir creates the storage directory if it doesn't exist
func (s *Storage) ensureDir() error {
	return os.MkdirAll(s.baseDir, 0755)
}

// generateID creates a unique conversation ID
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// NewConversation creates a new conversation
func NewConversation(model string, port int) *Conversation {
	now := time.Now()
	return &Conversation{
		ID:        generateID(),
		Title:     "New Chat",
		Model:     model,
		Port:      port,
		Messages:  []Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddMessage adds a message to the conversation
func (c *Conversation) AddMessage(role, content string) {
	c.Messages = append(c.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	c.UpdatedAt = time.Now()

	// Auto-generate title from first user message
	if c.Title == "New Chat" && role == "user" && len(content) > 0 {
		title := content
		if len(title) > 40 {
			title = title[:40] + "..."
		}
		c.Title = title
	}
}

// Save saves a conversation to disk
func (s *Storage) Save(conv *Conversation) error {
	if err := s.ensureDir(); err != nil {
		return err
	}

	filename := filepath.Join(s.baseDir, "chat-"+conv.ID+".json")
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// Load loads a conversation by ID
func (s *Storage) Load(id string) (*Conversation, error) {
	filename := filepath.Join(s.baseDir, "chat-"+id+".json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, err
	}

	return &conv, nil
}

// FindByModelAndPort finds the most recent conversation for a model/port
func (s *Storage) FindByModelAndPort(model string, port int) (*Conversation, error) {
	convs, err := s.List()
	if err != nil {
		return nil, err
	}

	// Find conversations matching model and port, return most recent
	var matching []*Conversation
	for _, c := range convs {
		if c.Model == model && c.Port == port {
			matching = append(matching, c)
		}
	}

	if len(matching) == 0 {
		return nil, nil // No matching conversation
	}

	// Sort by updated time, most recent first
	sort.Slice(matching, func(i, j int) bool {
		return matching[i].UpdatedAt.After(matching[j].UpdatedAt)
	})

	return matching[0], nil
}

// List returns all conversations
func (s *Storage) List() ([]*Conversation, error) {
	if err := s.ensureDir(); err != nil {
		return nil, err
	}

	files, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, err
	}

	var convs []*Conversation
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if filepath.Ext(f.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.baseDir, f.Name()))
		if err != nil {
			continue
		}

		var conv Conversation
		if err := json.Unmarshal(data, &conv); err != nil {
			continue
		}

		convs = append(convs, &conv)
	}

	// Sort by updated time, most recent first
	sort.Slice(convs, func(i, j int) bool {
		return convs[i].UpdatedAt.After(convs[j].UpdatedAt)
	})

	return convs, nil
}

// Delete deletes a conversation
func (s *Storage) Delete(id string) error {
	filename := filepath.Join(s.baseDir, "chat-"+id+".json")
	return os.Remove(filename)
}

// Rename renames a conversation
func (s *Storage) Rename(id, newTitle string) error {
	conv, err := s.Load(id)
	if err != nil {
		return err
	}
	conv.Title = newTitle
	return s.Save(conv)
}
