package preferences

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/language"
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

// AgentLanguages holds the raw language settings per level ("auto" or a code).
// Empty values mean the default.
type AgentLanguages struct {
	Chat          string `json:"chat,omitempty"`
	Documentation string `json:"documentation,omitempty"`
	Code          string `json:"code,omitempty"`
}

// AgentLanguagesUpdate is a partial update: nil fields are left untouched.
type AgentLanguagesUpdate struct {
	Chat          *string
	Documentation *string
	Code          *string
}

type Preferences struct {
	DefaultAgent               string                       `json:"defaultAgent"`
	Sessions                   map[string]SessionEntry      `json:"sessions,omitempty"`
	SessionAgents              map[string]string            `json:"sessionAgents,omitempty"` // legacy: migration source only
	Explorations               []ExplorationRecord          `json:"explorations,omitempty"`
	Env                        map[string]string            `json:"env,omitempty"`      // Custom hot-injected environment variables
	AgentEnv                   map[string]map[string]string `json:"agentEnv,omitempty"` // Per-agent environment variables, layered over Env
	NativeQuestionMode         bool                         `json:"nativeQuestionMode,omitempty"`
	CustomAgentSpecializations []string                     `json:"customAgentSpecializations,omitempty"`
	AgentLanguages             *AgentLanguages              `json:"agentLanguages,omitempty"`
	UILocale                   string                       `json:"uiLocale,omitempty"`
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
		p := &Preferences{DefaultAgent: "claude", Sessions: map[string]SessionEntry{}, Env: map[string]string{}}
		ensureAgentEnv(p)
		return p, nil
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
	if p.Env == nil {
		p.Env = map[string]string{}
	}
	ensureAgentEnv(&p)
	if migrateGeminiEnv(&p) {
		_ = s.save(&p) // best-effort: persist migrated data
	}
	return &p, nil
}

// ensureAgentEnv guarantees an (possibly empty) entry per supported agent.
func ensureAgentEnv(p *Preferences) {
	if p.AgentEnv == nil {
		p.AgentEnv = map[string]map[string]string{}
	}
	for _, a := range agents.SupportedAgents {
		if p.AgentEnv[a.ID] == nil {
			p.AgentEnv[a.ID] = map[string]string{}
		}
	}
}

// migrateGeminiEnv moves the historical Gemini keys from the global env to
// agentEnv["gemini"]. It is idempotent: once the keys left Env, nothing is found.
func migrateGeminiEnv(p *Preferences) bool {
	moved := false
	for _, k := range []string{"GOOGLE_CLOUD_PROJECT", "GEMINI_MODEL", "GEMINI_SANDBOX"} {
		if v, ok := p.Env[k]; ok {
			p.AgentEnv["gemini"][k] = v
			delete(p.Env, k)
			moved = true
		}
	}
	return moved
}

// EnvFor returns the global env overlaid with the agent-specific env.
func (p *Preferences) EnvFor(agentID string) map[string]string {
	out := make(map[string]string, len(p.Env))
	for k, v := range p.Env {
		out[k] = v
	}
	for k, v := range p.AgentEnv[agentID] {
		out[k] = v
	}
	return out
}

// LanguageLevels returns the raw stored settings; safe on a nil receiver.
func (p *Preferences) LanguageLevels() language.Levels {
	if p == nil || p.AgentLanguages == nil {
		return language.Levels{}
	}
	return language.Levels{Chat: p.AgentLanguages.Chat, Documentation: p.AgentLanguages.Documentation, Code: p.AgentLanguages.Code}
}

// ResolvedLanguages resolves the three levels; safe on a nil receiver
// (failed load), in which case defaults apply.
func (p *Preferences) ResolvedLanguages() language.Resolved {
	if p == nil {
		return language.Resolve(language.Levels{}, "")
	}
	return language.Resolve(p.LanguageLevels(), p.UILocale)
}

// LanguageDirective builds the language directive for a role; safe on a nil
// receiver, like EnvFor's callers tolerate a failed load.
func (p *Preferences) LanguageDirective(role language.Role) string {
	return language.Directive(role, p.ResolvedLanguages())
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

func (s *Service) SetEnv(env map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	p.Env = env
	return s.save(p)
}

// SetAgentEnv replaces the env dictionary of each agent present in updates.
func (s *Service) SetAgentEnv(updates map[string]map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	for id, env := range updates {
		if env == nil {
			env = map[string]string{}
		}
		p.AgentEnv[id] = env
	}
	return s.save(p)
}

// SetAgentLanguages applies a partial update of the language levels.
func (s *Service) SetAgentLanguages(u AgentLanguagesUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	if p.AgentLanguages == nil {
		p.AgentLanguages = &AgentLanguages{}
	}
	if u.Chat != nil {
		p.AgentLanguages.Chat = *u.Chat
	}
	if u.Documentation != nil {
		p.AgentLanguages.Documentation = *u.Documentation
	}
	if u.Code != nil {
		p.AgentLanguages.Code = *u.Code
	}
	return s.save(p)
}

func (s *Service) SetUILocale(locale string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	p.UILocale = locale
	return s.save(p)
}

func (s *Service) SetCustomAgentSpecializations(specializations []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	p.CustomAgentSpecializations = specializations
	return s.save(p)
}

// GetNativeQuestionMode returns whether the global native question mode
// (Claude's AskUserQuestion tool) is enabled. Disabled by default.
func (s *Service) GetNativeQuestionMode() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return false
	}
	return p.NativeQuestionMode
}

func (s *Service) SetNativeQuestionMode(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	p.NativeQuestionMode = enabled
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
