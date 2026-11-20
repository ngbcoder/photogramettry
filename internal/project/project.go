package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Project represents a PhotoGen project.
type Project struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	InputDir  string            `json:"input_dir"`
	OutputDir string            `json:"output_dir"`
	Detector  string            `json:"detector"`
	Workers   int               `json:"workers"`
	MaxDim    int               `json:"max_image_dim"`
	RatioTest float64           `json:"ratio_test"`
	Meta      map[string]string `json:"meta,omitempty"`
}

// Store persists projects as JSON files.
type Store struct {
	root string
	mu   sync.Mutex
}

// NewStore creates a project store rooted at dir.
func NewStore(root string) *Store {
	if root == "" {
		root = "./output"
	}
	_ = os.MkdirAll(filepath.Join(root, "projects"), 0o755)
	return &Store{root: root}
}

func (s *Store) dir() string { return filepath.Join(s.root, "projects") }

// List returns all projects.
func (s *Store) List() ([]Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.dir())
	if err != nil {
		return nil, err
	}
	out := []Project{}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.dir(), e.Name()))
		if err != nil {
			continue
		}
		var p Project
		if err := json.Unmarshal(b, &p); err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// Create persists a new project, assigning an ID if missing.
func (s *Store) Create(p Project) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = fmt.Sprintf("proj-%d", timeNow())
	}
	if err := s.write(p); err != nil {
		return Project{}, err
	}
	return p, nil
}

// Get loads a project by ID.
func (s *Store) Get(id string) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(filepath.Join(s.dir(), id+".json"))
	if err != nil {
		return Project{}, err
	}
	var p Project
	if err := json.Unmarshal(b, &p); err != nil {
		return Project{}, err
	}
	return p, nil
}

// Update persists changes to an existing project.
func (s *Store) Update(p Project) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		return Project{}, fmt.Errorf("missing id")
	}
	if err := s.write(p); err != nil {
		return Project{}, err
	}
	return p, nil
}

func (s *Store) write(p Project) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir(), p.ID+".json"), b, 0o644)
}

func timeNow() int64 {
	return time.Now().UnixNano()
}
