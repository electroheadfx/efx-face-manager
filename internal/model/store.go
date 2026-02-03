package model

import (
	"os"
	"path/filepath"
	"sort"
)

// Model represents an installed model
type Model struct {
	Name       string
	Path       string
	TargetPath string // The symlink target (actual cache location)
	IsSymlink  bool
}

// ModelType represents the type of model
type ModelType string

const (
	TypeLM              ModelType = "lm"
	TypeMultimodal      ModelType = "multimodal"
	TypeImageGeneration ModelType = "image-generation"
	TypeImageEdit       ModelType = "image-edit"
	TypeEmbeddings      ModelType = "embeddings"
	TypeWhisper         ModelType = "whisper"
)

// Store manages model storage
type Store struct {
	BaseDir string
}

// NewStore creates a new model store
func NewStore(baseDir string) *Store {
	return &Store{BaseDir: baseDir}
}

// List returns all installed models
func (s *Store) List() ([]Model, error) {
	entries, err := os.ReadDir(s.BaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Model{}, nil
		}
		return nil, err
	}

	var models []Model
	for _, entry := range entries {
		// Skip cache directory and hidden files
		name := entry.Name()
		if name == "cache" || name[0] == '.' {
			continue
		}

		fullPath := filepath.Join(s.BaseDir, name)
		
		// Use Lstat to properly detect symlinks
		info, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}

		isSymlink := info.Mode()&os.ModeSymlink != 0
		
		model := Model{
			Name:      name,
			Path:      fullPath,
			IsSymlink: isSymlink,
		}

		// Get symlink target
		if isSymlink {
			target, err := os.Readlink(fullPath)
			if err == nil {
				model.TargetPath = target
			}
		}

		// Include both symlinks and directories (for flexibility)
		if isSymlink || entry.IsDir() {
			models = append(models, model)
		}
	}

	// Sort by name
	sort.Slice(models, func(i, j int) bool {
		return models[i].Name < models[j].Name
	})

	return models, nil
}

// Get returns a specific model by name
func (s *Store) Get(name string) (*Model, error) {
	models, err := s.List()
	if err != nil {
		return nil, err
	}

	for _, m := range models {
		if m.Name == name {
			return &m, nil
		}
	}

	return nil, os.ErrNotExist
}

// Exists checks if a model exists
func (s *Store) Exists(name string) bool {
	_, err := s.Get(name)
	return err == nil
}

// Count returns the number of installed models
func (s *Store) Count() int {
	models, err := s.List()
	if err != nil {
		return 0
	}
	return len(models)
}

// Remove removes a model (symlink only, not cache)
func (s *Store) Remove(name string) error {
	model, err := s.Get(name)
	if err != nil {
		return err
	}

	return os.Remove(model.Path)
}

// RemoveWithCache removes a model and its cache
func (s *Store) RemoveWithCache(name string) error {
	model, err := s.Get(name)
	if err != nil {
		return err
	}

	// Remove symlink
	if err := os.Remove(model.Path); err != nil {
		return err
	}

	// Find and remove cache directory
	// Target path is like: .../cache/models--org--name/snapshots/hash
	if model.TargetPath != "" {
		// Go up from snapshots/hash to get the model cache folder
		cacheModelDir := filepath.Dir(filepath.Dir(model.TargetPath))
		if filepath.Base(filepath.Dir(cacheModelDir)) == "cache" {
			os.RemoveAll(cacheModelDir)
		}
	}

	return nil
}
