package preferences

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type SessionEntry struct {
	Agent           string `json:"agent"`
	ClaudeSessionId string `json:"claudeSessionId,omitempty"`
}

type ExplorationRecord struct {
	ID             string `json:"id"`
	WorkspaceID    string `json:"workspaceId"`
	Name           string `json:"name"`
	SessionID      string `json:"sessionId"`
	CreatedAt      string `json:"createdAt"`
	LastActivityAt string `json:"lastActivityAt"`
}

type Preferences struct {
	DefaultAgent  string                  `json:"defaultAgent"`
	Sessions      map[string]SessionEntry `json:"sessions,omitempty"`
	SessionAgents map[string]string       `json:"sessionAgents,omitempty"` // legacy: migration source only
	Explorations  []ExplorationRecord     `json:"explorations,omitempty"`
}

type Service struct {
	mu   sync.Mutex
	path string
}

func NewService(path string) *Service {
	return &Service{path: path}
}

func (s *Service) Path() string {
	return s.path
}

func (s *Service) load() (*Preferences, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return &Preferences{DefaultAgent: "claude", Sessions: map[string]SessionEntry{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var p Preferences
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	if p.DefaultAgent == "" {
		p.DefaultAgent = "claude"
	}
	// Migrate from legacy sessionAgents format
	if len(p.Sessions) == 0 && len(p.SessionAgents) > 0 {
		p.Sessions = make(map[string]SessionEntry, len(p.SessionAgents))
		for k, v := range p.SessionAgents {
			p.Sessions[k] = SessionEntry{Agent: v}
		}
		p.SessionAgents = nil
		_ = s.save(&p) // best-effort: persist migrated data
	}
	if p.Sessions == nil {
		p.Sessions = map[string]SessionEntry{}
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

func (s *Service) GetSession(workspaceID, changeName string) SessionEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return SessionEntry{}
	}
	return p.Sessions[workspaceID+"/"+changeName]
}

func (s *Service) SetSession(workspaceID, changeName string, entry SessionEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	p.Sessions[workspaceID+"/"+changeName] = entry
	return s.save(p)
}

func (s *Service) AddExploration(record ExplorationRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	if record.CreatedAt == "" {
		record.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if record.LastActivityAt == "" {
		record.LastActivityAt = record.CreatedAt
	}
	p.Explorations = append(p.Explorations, record)
	return s.save(p)
}

// TouchExplorationActivity updates LastActivityAt to now for the given exploration.
// Used as the anchor for exploreLogRetentionDays; a no-op if the id is unknown.
func (s *Service) TouchExplorationActivity(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	for i, e := range p.Explorations {
		if e.ID == id {
			p.Explorations[i].LastActivityAt = time.Now().UTC().Format(time.RFC3339)
			return s.save(p)
		}
	}
	return nil
}

func (s *Service) UpdateExplorationName(id, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	for i, e := range p.Explorations {
		if e.ID == id {
			p.Explorations[i].Name = name
			return s.save(p)
		}
	}
	return nil
}

func (s *Service) GetExploration(id, workspaceID string) *ExplorationRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return nil
	}
	for _, e := range p.Explorations {
		if e.ID == id && e.WorkspaceID == workspaceID {
			r := e
			return &r
		}
	}
	return nil
}

func (s *Service) DeleteExploration(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	updated := p.Explorations[:0]
	for _, e := range p.Explorations {
		if e.ID != id {
			updated = append(updated, e)
		}
	}
	p.Explorations = updated
	return s.save(p)
}

func (s *Service) ListExplorations(workspaceID string) []ExplorationRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return nil
	}
	var result []ExplorationRecord
	for _, e := range p.Explorations {
		if e.WorkspaceID == workspaceID {
			result = append(result, e)
		}
	}
	return result
}
