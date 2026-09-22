package session

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/preferences"
)

const inactivityTimeout = 30 * time.Minute
const maxMessages = 500

// explorationFramingPrompt instructs the agent to dedicate its first turn to
// framing the exploration and to signal any clarification question via the
// ghost_question marker. Shared between named and anonymous explore sessions.
const explorationFramingPrompt = `Dedicate your very first response in this conversation to framing the exploration: make sure you understand the user's intent and surface the open questions you need answered, rather than giving a generic reply.
Whenever you need to ask the user a clarifying question — including during this first turn — signal it by outputting, on its own line, exactly (no other text on that line):
{"event":"ghost_question","question":"THE_QUESTION_TEXT"}
where THE_QUESTION_TEXT is your question, phrased naturally. You may write other text before or after that line.`

const anonSystemPrompt = `You are in free exploration mode. The user will describe what they want to build or explore. Navigate files and explore the codebase to help them think through their ideas.
At the very START of your FIRST response, output on its own line exactly (no other text on that line):
{"event":"ghost_named","name":"THE_KEBAB_CASE_NAME"}
where THE_KEBAB_CASE_NAME is a concise kebab-case name (3-5 words) derived from the user's intent.
Do NOT call /opsx:ff or /opsx:new autonomously. You are in exploration mode only. Wait for explicit instruction from the user to create a change.

` + explorationFramingPrompt

type Session struct {
	proc     *Subprocess
	cancel   context.CancelFunc
	lastUsed time.Time
	mu       sync.Mutex

	msgMu           sync.RWMutex
	messages        [][]byte
	notify          chan struct{} // buffered(1): signals new messages available
	done            chan struct{} // closed when subprocess stdout ends
	pendingQuestion string        // clarification question awaiting a user reply, if any

	// nativeQuestionModeActive is fixed at session construction time (mirrors
	// the system prompt choice for this subprocess): true only when the
	// resolved agent is Claude and the global native question mode preference
	// was active. Read-only after startFanOut is launched, so it is safe to
	// read without synchronization.
	nativeQuestionModeActive bool

	log *conversation.SessionLog
}

// SetPendingQuestion records a clarification question as awaiting a user reply.
func (s *Session) SetPendingQuestion(question string) {
	s.msgMu.Lock()
	s.pendingQuestion = question
	s.msgMu.Unlock()
}

// ClearPendingQuestion clears the pending question state, typically once the
// user has sent their next message.
func (s *Session) ClearPendingQuestion() {
	s.msgMu.Lock()
	s.pendingQuestion = ""
	s.msgMu.Unlock()
}

// PendingQuestion returns the clarification question currently awaiting a
// user reply, or "" if none.
func (s *Session) PendingQuestion() string {
	s.msgMu.RLock()
	defer s.msgMu.RUnlock()
	return s.pendingQuestion
}

// NewTestSession constructs a bare Session with initialized channels, for use
// in tests of packages that depend on *Session (e.g. WebSocket handlers).
// proc may be nil if the test never calls Proc() (e.g. NewTestSubprocess to
// capture what gets written to stdin).
func NewTestSession(proc *Subprocess) *Session {
	return &Session{
		proc:   proc,
		notify: make(chan struct{}, 1),
		done:   make(chan struct{}),
	}
}

// Log returns the session's conversation log (may be nil if logging is disabled
// or failed to open). All SessionLog methods are nil-safe.
func (s *Session) Log() *conversation.SessionLog { return s.log }

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
	s.log.Close()
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

// InjectMessage inserts a custom message into the session's message buffer
// and notifies any listeners that new messages are available.
func (s *Session) InjectMessage(msg []byte) {
	s.msgMu.Lock()
	if len(s.messages) >= maxMessages {
		s.messages = s.messages[1:]
	}
	s.messages = append(s.messages, msg)
	s.msgMu.Unlock()

	select {
	case s.notify <- struct{}{}:
	default:
	}
}

type Manager struct {
	mu        sync.Mutex
	sessions  map[string]*Session
	prefs     *preferences.Service
	convStore *conversation.Store
}

func NewManager(prefs *preferences.Service, convStore *conversation.Store) *Manager {
	m := &Manager{
		sessions:  make(map[string]*Session),
		prefs:     prefs,
		convStore: convStore,
	}
	go m.reapLoop()
	return m
}

func (m *Manager) Prefs() *preferences.Service {
	return m.prefs
}

// openSessionLog opens a chat SessionLog via resolve, logging (not failing) on error.
// Returns nil if convStore is unset or the file could not be opened.
func (m *Manager) openSessionLog(resolve func(ts string) (*os.File, error), ctxLabel string) *conversation.SessionLog {
	if m.convStore == nil {
		return nil
	}
	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	f, err := resolve(ts)
	if err != nil {
		log.Printf("[session] failed to open chat log for %s: %v", ctxLabel, err)
		return nil
	}
	return conversation.NewSessionLog(f)
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

func (m *Manager) ResolveAgentConfig(workspaceID, changeName string) agents.AgentConfig {
	r := m.resolveAgent(workspaceID, changeName)
	return r.config
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

	// Capture the most recent prior run's timestamp (if any) before opening a
	// new one below, so a resumed session can replay it to detect a
	// clarification question left unanswered at the previous interruption.
	previousRunTs := ""
	if isResume && m.convStore != nil {
		if runs, err := m.convStore.List(workspaceID, changeName, "chat"); err == nil && len(runs) > 0 {
			previousRunTs = runs[0].Ts
		}
	}

	sessLog := m.openSessionLog(func(ts string) (*os.File, error) {
		return m.convStore.OpenRun(workspaceID, changeName, "chat", ts)
	}, key)

	var customEnv map[string]string
	nativeQuestionMode := false
	if p, err := m.prefs.Load(); err == nil && p != nil {
		customEnv = p.Env
		nativeQuestionMode = p.NativeQuestionMode
	}

	ctx, cancel := context.WithCancel(context.Background())
	proc, err := StartSubprocess(ctx, workspacePath, resolved.config, explorationFramingPrompt, claudeSessionID, isResume, sessLog, customEnv, nativeQuestionMode)
	if err != nil && isResume {
		// Fallback: --resume failed at process start, try without resume
		log.Printf("[session] --resume failed for %s/%s, starting fresh: %v", workspaceID, changeName, err)
		proc, err = StartSubprocess(ctx, workspacePath, resolved.config, explorationFramingPrompt, claudeSessionID, false, sessLog, customEnv, nativeQuestionMode)
	}
	if err != nil {
		cancel()
		sessLog.Close()
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
		proc:                     proc,
		cancel:                   cancel,
		lastUsed:                 time.Now(),
		notify:                   make(chan struct{}, 1),
		done:                     make(chan struct{}),
		log:                      sessLog,
		nativeQuestionModeActive: resolved.config.ID == "claude" && nativeQuestionMode,
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
		s.log.WriteLine("in", initMsg)
		proc.Write(append(initMsg, '\n'))
	} else if previousRunTs != "" {
		m.injectResumeFollowUpIfPending(s, func() ([]LogLine, error) {
			raw, err := m.convStore.Load(workspaceID, changeName, "chat", previousRunTs)
			if err != nil {
				return nil, err
			}
			return ParseLogLines(raw), nil
		})
	}

	return s, nil
}

// injectResumeFollowUpIfPending loads a resumed session's prior run via load,
// reconstructs whether a clarification question was left unanswered, and if
// so, injects an automatic follow-up asking the agent to rephrase and ask it
// again — since the interrupted subprocess's own tool_use/marker state cannot
// be restored.
func (m *Manager) injectResumeFollowUpIfPending(s *Session, load func() ([]LogLine, error)) {
	lines, err := load()
	if err != nil {
		log.Printf("[session] failed to load prior run for resume follow-up: %v", err)
		return
	}
	pending := ReconstructPendingQuestion(lines)
	if pending == "" {
		return
	}
	followUp := resumeFollowUpMessage(pending)
	if followUp == nil {
		return
	}
	s.log.WriteLine("in", followUp)
	s.proc.Write(append(followUp, '\n'))
}

// StartAnonymous creates a session without a known changeName, identified by
// sessionID if given, or a freshly generated one otherwise. The session will
// be promoted to a named session when the LLM emits the change_created marker.
//
// Passing an existing ghost id as sessionID resumes it: if a live session
// still exists under that id it is returned as-is (mirrors Start's
// reuse-if-active semantics), otherwise a new subprocess is started reusing
// the same id, so its conversation log lands in the same _explore/<id>/
// directory as the original session.
func (m *Manager) StartAnonymous(workspaceID, workspacePath, sessionID string) (string, *Session, error) {
	// A caller-provided sessionID that isn't alive below means this call is
	// restarting a subprocess for a previously interrupted exploration.
	isRestart := sessionID != ""
	if sessionID == "" {
		sessionID = newSessionID()
	}
	key := anonKey(workspaceID, sessionID)

	m.mu.Lock()
	if s, ok := m.sessions[key]; ok {
		m.mu.Unlock()
		return sessionID, s, nil
	}
	m.mu.Unlock()

	resolved := m.resolveAgent(workspaceID, "")

	// A subprocess restart reusing an existing ghost id is this function's
	// equivalent of Start's --resume: capture the most recent prior run's
	// timestamp (if any) before opening a new one below, so it can be
	// replayed to detect a clarification question left unanswered at the
	// previous interruption.
	previousRunTs := ""
	if isRestart && m.convStore != nil {
		if runs, err := m.convStore.ListExplore(workspaceID, sessionID, "chat"); err == nil && len(runs) > 0 {
			previousRunTs = runs[0].Ts
		}
	}

	sessLog := m.openSessionLog(func(ts string) (*os.File, error) {
		return m.convStore.OpenExploreRun(workspaceID, sessionID, "chat", ts)
	}, key)

	var customEnv map[string]string
	nativeQuestionMode := false
	if p, err := m.prefs.Load(); err == nil && p != nil {
		customEnv = p.Env
		nativeQuestionMode = p.NativeQuestionMode
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Anonymous sessions use no session flags (no persistence, no resume)
	proc, err := StartSubprocess(ctx, workspacePath, resolved.config, anonSystemPrompt, "", false, sessLog, customEnv, nativeQuestionMode)
	if err != nil {
		cancel()
		sessLog.Close()
		return "", nil, err
	}

	s := &Session{
		proc:                     proc,
		cancel:                   cancel,
		lastUsed:                 time.Now(),
		notify:                   make(chan struct{}, 1),
		done:                     make(chan struct{}),
		log:                      sessLog,
		nativeQuestionModeActive: resolved.config.ID == "claude" && nativeQuestionMode,
	}

	injectAgentInfo(s, resolved)

	m.mu.Lock()
	m.sessions[key] = s
	m.mu.Unlock()

	m.startFanOut(s, key, workspaceID, true)

	if isRestart && previousRunTs != "" {
		m.injectResumeFollowUpIfPending(s, func() ([]LogLine, error) {
			raw, err := m.convStore.LoadExplore(workspaceID, sessionID, "chat", previousRunTs)
			if err != nil {
				return nil, err
			}
			return ParseLogLines(raw), nil
		})
	}

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

// nativeQuestionBlock accumulates the streamed JSON input of an in-progress
// AskUserQuestion tool_use content block, identified by its tool_use_id.
type nativeQuestionBlock struct {
	id   string
	json strings.Builder
}

// unwrapStreamEvent returns the inner event object's raw bytes when line is a
// {"type":"stream_event","event":{...}} wrapper — the shape recent Claude CLI
// versions use for incremental content_block_* events — or line unchanged
// otherwise (e.g. the consolidated end-of-turn "assistant"/"result" events,
// or a Gemini-translated line, are never wrapped this way).
func unwrapStreamEvent(line []byte) []byte {
	var outer struct {
		Type  string          `json:"type"`
		Event json.RawMessage `json:"event"`
	}
	if err := json.Unmarshal(line, &outer); err != nil {
		return line
	}
	if outer.Type != "stream_event" || len(outer.Event) == 0 {
		return line
	}
	return outer.Event
}

// detectNativeQuestionBlock inspects a single stdout line for the parts of a
// Claude tool_use content block named "AskUserQuestion"
// (content_block_start / content_block_delta / content_block_stop), tracked
// by content block index in blocks (owned by the caller's goroutine, no
// locking needed).
//
// It returns suppress=true when the line is part of such a block and must
// never be forwarded to the frontend as raw text. event is non-nil once the
// block is complete (content_block_stop), ready to be appended to the
// session's message buffer in place of the raw line.
func detectNativeQuestionBlock(blocks map[int]*nativeQuestionBlock, line []byte) (suppress bool, event []byte) {
	line = unwrapStreamEvent(line)
	var data map[string]interface{}
	if err := json.Unmarshal(line, &data); err != nil {
		return false, nil
	}
	indexF, hasIndex := data["index"].(float64)
	if !hasIndex {
		return false, nil
	}
	index := int(indexF)
	typ, _ := data["type"].(string)

	switch typ {
	case "content_block_start":
		cb, _ := data["content_block"].(map[string]interface{})
		if cb == nil {
			return false, nil
		}
		if cbType, _ := cb["type"].(string); cbType != "tool_use" {
			return false, nil
		}
		if name, _ := cb["name"].(string); name != "AskUserQuestion" {
			return false, nil
		}
		id, _ := cb["id"].(string)
		blocks[index] = &nativeQuestionBlock{id: id}
		return true, nil

	case "content_block_delta":
		blk, ok := blocks[index]
		if !ok {
			return false, nil
		}
		if delta, ok := data["delta"].(map[string]interface{}); ok {
			if partial, ok := delta["partial_json"].(string); ok {
				blk.json.WriteString(partial)
			}
		}
		return true, nil

	case "content_block_stop":
		blk, ok := blocks[index]
		if !ok {
			return false, nil
		}
		delete(blocks, index)

		raw := blk.json.String()
		if raw == "" {
			raw = "{}"
		}
		var input map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &input); err != nil {
			// Malformed input: drop silently rather than leak raw JSON to the frontend.
			return true, nil
		}
		evt := map[string]interface{}{
			"type":        "native_question",
			"tool_use_id": blk.id,
		}
		for k, v := range input {
			evt[k] = v
		}
		out, err := json.Marshal(evt)
		if err != nil {
			return true, nil
		}
		return true, out
	}
	return false, nil
}

// nativeQuestionSummary extracts a human-readable summary text from a
// native_question event, used as the session's pending-question state.
func nativeQuestionSummary(event []byte) string {
	var evt map[string]interface{}
	if err := json.Unmarshal(event, &evt); err != nil {
		return ""
	}
	questions, ok := evt["questions"].([]interface{})
	if !ok || len(questions) == 0 {
		return "AskUserQuestion"
	}
	var parts []string
	for _, q := range questions {
		qm, ok := q.(map[string]interface{})
		if !ok {
			continue
		}
		if text, ok := qm["question"].(string); ok && text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return "AskUserQuestion"
	}
	return strings.Join(parts, " ")
}

// LogLine is a direction-tagged message, either replayed from a persisted
// JSONL conversation log ("in"/"out"/"err") or taken from an in-memory
// buffer, for use with ReconstructPendingQuestion.
type LogLine struct {
	Dir  string
	Data []byte
}

// persistedLogLine mirrors the {ts,dir,data} shape written by
// conversation.SessionLog, just enough to recover Dir/Data when replaying a
// JSONL log file.
type persistedLogLine struct {
	Dir  string          `json:"dir"`
	Data json.RawMessage `json:"data"`
}

// ParseLogLines decodes raw JSONL lines (as returned by conversation.Store's
// Load/LoadExplore) into LogLine, skipping any line that fails to parse.
func ParseLogLines(rawLines [][]byte) []LogLine {
	lines := make([]LogLine, 0, len(rawLines))
	for _, raw := range rawLines {
		var pl persistedLogLine
		if err := json.Unmarshal(raw, &pl); err != nil {
			continue
		}
		lines = append(lines, LogLine{Dir: pl.Dir, Data: pl.Data})
	}
	return lines
}

// ReconstructPendingQuestion replays a session's message history (in
// chronological order) and returns the clarification question still
// awaiting a reply at the end, or "" if none. It mirrors the live Session
// tracking (SetPendingQuestion/ClearPendingQuestion): an "out" line carrying
// a ghost_question marker or a complete AskUserQuestion tool_use block opens
// a pending question, and any subsequent "in" (user) line clears it. This
// lets a fresh Session rebuild the same state a still-alive one would have,
// from a persisted JSONL log replayed after a subprocess restart, or from an
// in-memory buffer for a session that never went away.
func ReconstructPendingQuestion(lines []LogLine) string {
	pending := ""
	nativeBlocks := map[int]*nativeQuestionBlock{}
	for _, l := range lines {
		switch l.Dir {
		case "out":
			if q := ExtractGhostQuestion(l.Data); q != "" {
				pending = q
				continue
			}
			if suppress, event := detectNativeQuestionBlock(nativeBlocks, l.Data); suppress && event != nil {
				if summary := nativeQuestionSummary(event); summary != "" {
					pending = summary
				}
			}
		case "in":
			pending = ""
		}
	}
	return pending
}

// resumeFollowUpMessage builds the stream-json stdin payload for the
// automatic follow-up sent on resume when a clarification question was left
// unanswered, asking the agent to rephrase and ask it again.
func resumeFollowUpMessage(pendingQuestion string) []byte {
	content := "Your previous session was interrupted while you had an unanswered clarification question pending (\"" + pendingQuestion + "\"). Please rephrase and ask it again now, using the ghost_question marker (or the AskUserQuestion tool if native question mode is active) as usual."
	payload := map[string]interface{}{
		"type": "user",
		"message": map[string]string{
			"role":    "user",
			"content": content,
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return data
}

// startFanOut launches the goroutine that reads subprocess stdout into the message buffer.
func (m *Manager) startFanOut(s *Session, key string, workspaceID string, anonymous bool) {
	go func() {
		defer close(s.done)
		nativeBlocks := map[int]*nativeQuestionBlock{}
		scanner := bufio.NewScanner(s.proc.Stdout())
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			b := make([]byte, len(scanner.Bytes()))
			copy(b, scanner.Bytes())
			s.log.WriteLine("out", b)

			if suppress, event := detectNativeQuestionBlock(nativeBlocks, b); suppress {
				if event == nil {
					// Mid-block chunk of an AskUserQuestion tool_use: never forwarded as raw text.
					continue
				}
				if !s.nativeQuestionModeActive {
					// Defense in depth: an AskUserQuestion tool_use block outside of an
					// active native session (unexpected agent, or a Gemini bridge line
					// that happens to match the shape) is dropped silently — never
					// surfaced as raw text, and no native_question card either, since
					// there is no active tool_use/tool_result round trip to answer it.
					continue
				}
				b = event
				if summary := nativeQuestionSummary(event); summary != "" {
					s.SetPendingQuestion(summary)
				}
			}

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
// Tries JSON first, falls back to substring search for tolerance, including Gemini-translated delta format.
func ExtractGhostNamed(line []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(line, &data); err == nil {
		if event, ok := data["event"].(string); ok && event == "ghost_named" {
			if name, ok := data["name"].(string); ok && name != "" {
				return name
			}
		}
		// Also handle translated Claude/Gemini content_block_delta format
		if delta, ok := data["delta"].(map[string]interface{}); ok {
			if text, ok := delta["text"].(string); ok && text != "" {
				var inner map[string]interface{}
				if err := json.Unmarshal([]byte(text), &inner); err == nil {
					if event, ok := inner["event"].(string); ok && event == "ghost_named" {
						if name, ok := inner["name"].(string); ok && name != "" {
							return name
						}
					}
				}
				// Substring search on unescaped text content
				if strings.Contains(text, `"event":"ghost_named"`) || strings.Contains(text, `"event": "ghost_named"`) {
					if idx := strings.Index(text, `"name":"`); idx != -1 {
						rest := text[idx+8:]
						if end := strings.Index(rest, `"`); end != -1 && end > 0 {
							return rest[:end]
						}
					}
				}
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
	// Fallback for escaped quotes (e.g. inside raw string representation of content_block_delta)
	if strings.Contains(s, `\"event\":\"ghost_named\"`) || strings.Contains(s, `\"event\": \"ghost_named\"`) {
		if idx := strings.Index(s, `\"name\":\"`); idx != -1 {
			rest := s[idx+11:]
			if end := strings.Index(rest, `\"`); end != -1 && end > 0 {
				return rest[:end]
			}
		}
	}
	return ""
}

// ExtractGhostQuestion parses a line for the ghost_question marker emitted by the LLM.
// Tries JSON first, falls back to substring search for tolerance, including Gemini-translated delta format.
func ExtractGhostQuestion(line []byte) string {
	const jsonPrefix = `"question":"`
	const escapedPrefix = `\"question\":\"`

	var data map[string]interface{}
	if err := json.Unmarshal(line, &data); err == nil {
		if event, ok := data["event"].(string); ok && event == "ghost_question" {
			if question, ok := data["question"].(string); ok && question != "" {
				return question
			}
		}
		// Also handle translated Claude/Gemini content_block_delta format
		if delta, ok := data["delta"].(map[string]interface{}); ok {
			if text, ok := delta["text"].(string); ok && text != "" {
				var inner map[string]interface{}
				if err := json.Unmarshal([]byte(text), &inner); err == nil {
					if event, ok := inner["event"].(string); ok && event == "ghost_question" {
						if question, ok := inner["question"].(string); ok && question != "" {
							return question
						}
					}
				}
				// Substring search on unescaped text content
				if strings.Contains(text, `"event":"ghost_question"`) || strings.Contains(text, `"event": "ghost_question"`) {
					if idx := strings.Index(text, jsonPrefix); idx != -1 {
						rest := text[idx+len(jsonPrefix):]
						if end := strings.Index(rest, `"`); end != -1 && end > 0 {
							return rest[:end]
						}
					}
				}
			}
		}
	}
	s := string(line)
	if strings.Contains(s, `"event":"ghost_question"`) || strings.Contains(s, `"event": "ghost_question"`) {
		if idx := strings.Index(s, jsonPrefix); idx != -1 {
			rest := s[idx+len(jsonPrefix):]
			if end := strings.Index(rest, `"`); end != -1 && end > 0 {
				return rest[:end]
			}
		}
	}
	// Fallback for escaped quotes (e.g. inside raw string representation of content_block_delta)
	if strings.Contains(s, `\"event\":\"ghost_question\"`) || strings.Contains(s, `\"event\": \"ghost_question\"`) {
		if idx := strings.Index(s, escapedPrefix); idx != -1 {
			rest := s[idx+len(escapedPrefix):]
			if end := strings.Index(rest, `\"`); end != -1 && end > 0 {
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
