package preferences

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Preferences struct {
	DefaultAgent  string            `json:"defaultAgent"`
	SessionAgents map[string]string `json:"sessionAgents"`
}

type Service struct {
	mu   sync.Mutex
	path string
}

func NewService(path string) *Service {
	return &Service{path: path}
}

func (s *Service) load() (*Preferences, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return &Preferences{DefaultAgent: "claude", SessionAgents: map[string]string{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var p Preferences
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	if p.SessionAgents == nil {
		p.SessionAgents = map[string]string{}
	}
	if p.DefaultAgent == "" {
		p.DefaultAgent = "claude"
	}
	return &p, nil
}

func (s *Service) save(p *Preferences) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *Service) Load() (*Preferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *Service) GetDefaultAgent() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return "claude"
	}
	return p.DefaultAgent
}

func (s *Service) SetDefaultAgent(agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	p.DefaultAgent = agentID
	return s.save(p)
}

func (s *Service) GetSessionAgent(workspaceID, changeName string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return ""
	}
	return p.SessionAgents[workspaceID+"/"+changeName]
}

func (s *Service) SetSessionAgent(workspaceID, changeName, agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	p.SessionAgents[workspaceID+"/"+changeName] = agentID
	return s.save(p)
}
