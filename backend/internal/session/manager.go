package session

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/preferences"
)

const inactivityTimeout = 30 * time.Minute
const maxMessages = 500

const anonSystemPrompt = `You are in free exploration mode. The user will describe what they want to build or explore. Navigate files and explore the codebase to help them think through their ideas.
At the very START of your FIRST response, output on its own line exactly (no other text on that line):
{"event":"ghost_named","name":"THE_KEBAB_CASE_NAME"}
where THE_KEBAB_CASE_NAME is a concise kebab-case name (3-5 words) derived from the user's intent.
Do NOT call /opsx:ff or /opsx:new autonomously. You are in exploration mode only. Wait for explicit instruction from the user to create a change.`

type Session struct {
	proc     *Subprocess
	cancel   context.CancelFunc
	lastUsed time.Time
	mu       sync.Mutex

	msgMu    sync.RWMutex
	messages [][]byte
	notify   chan struct{} // buffered(1): signals new messages available
	done     chan struct{} // closed when subprocess stdout ends
}

func (s *Session) Proc() *Subprocess {
	s.mu.Lock()
	s.lastUsed = time.Now()
	s.mu.Unlock()
	return s.proc
}

func (s *Session) Stop() {
	s.cancel()
	s.proc.CloseStdin()
	s.proc.Wait()
}

// Snapshot returns a copy of the message buffer and the cursor (buffer length at snapshot time).
func (s *Session) Snapshot() ([][]byte, int) {
	s.msgMu.RLock()
	defer s.msgMu.RUnlock()
	snap := make([][]byte, len(s.messages))
	copy(snap, s.messages)
	return snap, len(s.messages)
}

// MessagesSince returns messages from cursor onward and the updated cursor.
// If cursor exceeds buffer length (sliding window moved), starts from 0.
func (s *Session) MessagesSince(cursor int) ([][]byte, int) {
	s.msgMu.RLock()
	defer s.msgMu.RUnlock()
	if cursor > len(s.messages) {
		cursor = 0
	}
	slice := s.messages[cursor:]
	msgs := make([][]byte, len(slice))
	copy(msgs, slice)
	return msgs, len(s.messages)
}

func (s *Session) Notify() <-chan struct{} { return s.notify }
func (s *Session) Done() <-chan struct{}   { return s.done }

type Manager struct {
	mu       sync.Mutex
	sessions map[string]*Session
	prefs    *preferences.Service
}

func NewManager(prefs *preferences.Service) *Manager {
	m := &Manager{
		sessions: make(map[string]*Session),
		prefs:    prefs,
	}
	go m.reapLoop()
	return m
}

func sessionKey(workspaceID, changeName string) string {
	return workspaceID + "/" + changeName
}

func anonKey(workspaceID, sessionID string) string {
	return workspaceID + "/__anon__/" + sessionID
}

func newSessionID() string {
	// Used for anonymous session keys (not Claude session IDs).
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// newClaudeSessionID generates a random UUID v4 for Claude's --session-id flag.
func newClaudeSessionID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 1
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

func (m *Manager) Get(workspaceID, changeName string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[sessionKey(workspaceID, changeName)]
}

func (m *Manager) GetAnonymous(workspaceID, sessionID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[anonKey(workspaceID, sessionID)]
}

type resolvedAgent struct {
	config         agents.AgentConfig
	status         agents.AgentStatus
	usedFallback   bool
	requestedLabel string
}

func (m *Manager) resolveAgentFromID(agentID string) resolvedAgent {
	if agentID == "" {
		agentID = m.prefs.GetDefaultAgent()
	}
	cfg, ok := agents.ByID(agentID)
	if !ok {
		cfg, _ = agents.ByID("claude")
	}
	status := agents.Detect(cfg)
	if !status.Installed {
		log.Printf("[session] agent %q not installed, falling back to claude", cfg.ID)
		requestedLabel := cfg.Label
		cfg, _ = agents.ByID("claude")
		claudeStatus := agents.Detect(cfg)
		return resolvedAgent{config: cfg, status: claudeStatus, usedFallback: true, requestedLabel: requestedLabel}
	}
	return resolvedAgent{config: cfg, status: status}
}

func (m *Manager) resolveAgent(workspaceID, changeName string) resolvedAgent {
	agentID := ""
	if changeName != "" {
		agentID = m.prefs.GetSession(workspaceID, changeName).Agent
	}
	return m.resolveAgentFromID(agentID)
}

func injectAgentInfo(s *Session, r resolvedAgent) {
	msg := map[string]interface{}{
		"type":    "agent_info",
		"id":      r.config.ID,
		"label":   r.config.Label,
		"version": r.status.Version,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	s.msgMu.Lock()
	s.messages = append(s.messages, data)
	s.msgMu.Unlock()

	if r.usedFallback {
		warn := map[string]interface{}{
			"type": "session_warning",
			"text": fmt.Sprintf("Agent \"%s\" non installé — utilisation de Claude par défaut.", r.requestedLabel),
		}
		warnData, err := json.Marshal(warn)
		if err != nil {
			return
		}
		s.msgMu.Lock()
		s.messages = append(s.messages, warnData)
		s.msgMu.Unlock()
	}
}

func (m *Manager) Start(workspaceID, changeName, workspacePath string) (*Session, error) {
	key := sessionKey(workspaceID, changeName)

	m.mu.Lock()
	if s, ok := m.sessions[key]; ok {
		m.mu.Unlock()
		return s, nil
	}
	m.mu.Unlock()

	// Read persisted session entry (agent + claudeSessionId)
	entry := m.prefs.GetSession(workspaceID, changeName)
	resolved := m.resolveAgentFromID(entry.Agent)

	claudeSessionID := entry.ClaudeSessionId
	isResume := claudeSessionID != ""
	if !isResume {
		claudeSessionID = newClaudeSessionID()
	}

	ctx, cancel := context.WithCancel(context.Background())
	proc, err := StartSubprocess(ctx, workspacePath, resolved.config, "", claudeSessionID, isResume)
	if err != nil && isResume {
		// Fallback: --resume failed at process start, try without resume
		log.Printf("[session] --resume failed for %s/%s, starting fresh: %v", workspaceID, changeName, err)
		proc, err = StartSubprocess(ctx, workspacePath, resolved.config, "", claudeSessionID, false)
	}
	if err != nil {
		cancel()
		return nil, err
	}

	// Persist after subprocess starts successfully to avoid stale --resume on next attempt
	newEntry := preferences.SessionEntry{
		Agent:           resolved.config.ID,
		ClaudeSessionId: claudeSessionID,
	}
	if err := m.prefs.SetSession(workspaceID, changeName, newEntry); err != nil {
		log.Printf("[session] failed to persist session entry: %v", err)
	}

	s := &Session{
		proc:     proc,
		cancel:   cancel,
		lastUsed: time.Now(),
		notify:   make(chan struct{}, 1),
		done:     make(chan struct{}),
	}

	injectAgentInfo(s, resolved)

	m.mu.Lock()
	m.sessions[key] = s
	m.mu.Unlock()

	m.startFanOut(s, key, workspaceID, false)

	// Auto-inject /opsx:explore only on first session start, not on resume
	if !isResume {
		initPayload := map[string]interface{}{
			"type": "user",
			"message": map[string]string{
				"role":    "user",
				"content": fmt.Sprintf("/opsx:explore %s", changeName),
			},
		}
		initMsg, _ := json.Marshal(initPayload)
		proc.Write(append(initMsg, '\n'))
	}

	return s, nil
}

// StartAnonymous creates a session without a known changeName, identified by a UUID.
// The session will be promoted to a named session when the LLM emits the change_created marker.
func (m *Manager) StartAnonymous(workspaceID, workspacePath string) (string, *Session, error) {
	sessionID := newSessionID()
	key := anonKey(workspaceID, sessionID)

	resolved := m.resolveAgent(workspaceID, "")

	ctx, cancel := context.WithCancel(context.Background())
	// Anonymous sessions use no session flags (no persistence, no resume)
	proc, err := StartSubprocess(ctx, workspacePath, resolved.config, anonSystemPrompt, "", false)
	if err != nil {
		cancel()
		return "", nil, err
	}

	s := &Session{
		proc:     proc,
		cancel:   cancel,
		lastUsed: time.Now(),
		notify:   make(chan struct{}, 1),
		done:     make(chan struct{}),
	}

	injectAgentInfo(s, resolved)

	m.mu.Lock()
	m.sessions[key] = s
	m.mu.Unlock()

	m.startFanOut(s, key, workspaceID, true)

	return sessionID, s, nil
}

// Promote moves a session from the anonymous key to a named change key.
// Called by the fan-out goroutine when the change_created marker is detected.
func (m *Manager) Promote(oldKey, workspaceID, changeName string) {
	newKey := sessionKey(workspaceID, changeName)
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[oldKey]
	if !ok {
		return
	}
	delete(m.sessions, oldKey)
	if _, exists := m.sessions[newKey]; !exists {
		m.sessions[newKey] = s
	}
}

// startFanOut launches the goroutine that reads subprocess stdout into the message buffer.
func (m *Manager) startFanOut(s *Session, key string, workspaceID string, anonymous bool) {
	go func() {
		defer close(s.done)
		scanner := bufio.NewScanner(s.proc.Stdout())
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			b := make([]byte, len(scanner.Bytes()))
			copy(b, scanner.Bytes())

			s.msgMu.Lock()
			if len(s.messages) >= maxMessages {
				s.messages = s.messages[1:]
			}
			s.messages = append(s.messages, b)
			s.msgMu.Unlock()

			select {
			case s.notify <- struct{}{}:
			default:
			}
		}
	}()
}

// ExtractGhostNamed parses a line for the ghost_named marker emitted by the LLM.
// Tries JSON first, falls back to substring search for tolerance.
func ExtractGhostNamed(line []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(line, &data); err == nil {
		if event, ok := data["event"].(string); ok && event == "ghost_named" {
			if name, ok := data["name"].(string); ok && name != "" {
				return name
			}
		}
	}
	s := string(line)
	if strings.Contains(s, `"event":"ghost_named"`) || strings.Contains(s, `"event": "ghost_named"`) {
		if idx := strings.Index(s, `"name":"`); idx != -1 {
			rest := s[idx+8:]
			if end := strings.Index(rest, `"`); end != -1 && end > 0 {
				return rest[:end]
			}
		}
	}
	return ""
}

// extractChangeCreated parses a line for the change_created marker.
// Tries JSON first, falls back to substring search for tolerance.
func extractChangeCreated(line []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(line, &data); err == nil {
		if event, ok := data["event"].(string); ok && event == "change_created" {
			if name, ok := data["name"].(string); ok && name != "" {
				return name
			}
		}
	}
	s := string(line)
	if strings.Contains(s, `"event":"change_created"`) || strings.Contains(s, `"event": "change_created"`) {
		if idx := strings.Index(s, `"name":"`); idx != -1 {
			rest := s[idx+8:]
			if end := strings.Index(rest, `"`); end != -1 && end > 0 {
				return rest[:end]
			}
		}
	}
	return ""
}

func (m *Manager) Stop(workspaceID, changeName string) {
	m.stopByKey(sessionKey(workspaceID, changeName))
}

func (m *Manager) StopAnonymous(workspaceID, sessionID string) {
	m.stopByKey(anonKey(workspaceID, sessionID))
}

func (m *Manager) stopByKey(key string) {
	m.mu.Lock()
	s, ok := m.sessions[key]
	if ok {
		delete(m.sessions, key)
	}
	m.mu.Unlock()
	if ok {
		s.Stop()
	}
}

func (m *Manager) reapLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.mu.Lock()
		for key, s := range m.sessions {
			s.mu.Lock()
			idle := time.Since(s.lastUsed)
			s.mu.Unlock()
			if idle > inactivityTimeout {
				delete(m.sessions, key)
				go s.Stop()
			}
		}
		m.mu.Unlock()
	}
}
