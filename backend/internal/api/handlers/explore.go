package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/language"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
	"github.com/go-chi/chi/v5"
	"nhooyr.io/websocket"
)

// prependExploreSkill prefixes the user message content with "/opsx:explore "
// to trigger the explore skill on the first message of an anonymous session.
// Returns msg unchanged if parsing fails.
func prependExploreSkill(msg []byte) []byte {
	var payload map[string]interface{}
	if err := json.Unmarshal(msg, &payload); err != nil {
		return msg
	}
	message, ok := payload["message"].(map[string]interface{})
	if !ok {
		return msg
	}
	content, ok := message["content"].(string)
	if !ok {
		return msg
	}
	message["content"] = "/opsx:explore " + content
	payload["message"] = message
	result, err := json.Marshal(payload)
	if err != nil {
		return msg
	}
	return result
}

// extractGhostNamed parses a buffered session message for the ghost_named marker.
// The prefilter checks for the bare word rather than a quoted substring: when
// the marker is embedded inside a larger JSON string (e.g. a consolidated
// assistant message), its surrounding quotes are themselves escaped (\"), so
// a quoted prefilter would never match — the bare word matches either way.
func extractGhostNamed(line []byte) string {
	s := string(line)
	if !strings.Contains(s, `ghost_named`) {
		return ""
	}
	return session.ExtractGhostNamed(line)
}

// extractAllGhostQuestions parses a buffered session message for every
// ghost_question marker it may carry. See extractGhostNamed for why the
// prefilter checks the bare word.
func extractAllGhostQuestions(line []byte) []string {
	s := string(line)
	if !strings.Contains(s, `ghost_question`) {
		return nil
	}
	return session.ExtractAllGhostQuestions(line)
}

// ghostQuestionMarkerPattern matches the marker as it appears once JSON has
// been decoded (i.e. in a plain string value, not the escaped form found in
// a raw JSONL line).
var ghostQuestionMarkerPattern = regexp.MustCompile(`\{"event":\s*"ghost_question",\s*"question":\s*"(?:[^"\\]|\\.)*"\}\n?`)

// stripGhostQuestionMarker returns a copy of msg with any embedded
// ghost_question marker substring removed from its string values.
//
// The system prompt asks the agent to put the marker on its own line, but in
// practice a single stdout event can carry the marker mixed in with other,
// ordinary response text (e.g. the model's whole turn arrives as one
// consolidated message, or a translated bridge collapses a turn into a
// single event) — dropping the whole line, as detectGhostQuestion used to,
// would silently discard that surrounding text along with the marker. This
// walks the decoded JSON structure and strips the marker from every string
// field instead, so the rest of the message still reaches the frontend.
// Returns msg unchanged if it doesn't parse as JSON.
func stripGhostQuestionMarker(msg []byte) []byte {
	var data interface{}
	if err := json.Unmarshal(msg, &data); err != nil {
		return msg
	}
	cleaned := stripGhostQuestionMarkerValue(data)
	out, err := json.Marshal(cleaned)
	if err != nil {
		return msg
	}
	return out
}

func stripGhostQuestionMarkerValue(v interface{}) interface{} {
	switch val := v.(type) {
	case string:
		return ghostQuestionMarkerPattern.ReplaceAllString(val, "")
	case map[string]interface{}:
		for k, vv := range val {
			val[k] = stripGhostQuestionMarkerValue(vv)
		}
		return val
	case []interface{}:
		for i, vv := range val {
			val[i] = stripGhostQuestionMarkerValue(vv)
		}
		return val
	default:
		return v
	}
}

// parseNativeQuestionResponse parses a client-sent WebSocket message as a
// reply to a native_question card. The frontend envelope is translated into
// the actual tool_result wire format expected by the Claude subprocess by
// session.BuildToolResultMessage.
func parseNativeQuestionResponse(msg []byte) (toolUseID, content string, ok bool) {
	var payload struct {
		Type      string `json:"type"`
		ToolUseID string `json:"toolUseId"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal(msg, &payload); err != nil {
		return "", "", false
	}
	if payload.Type != "native_question_response" || payload.ToolUseID == "" {
		return "", "", false
	}
	return payload.ToolUseID, payload.Content, true
}

type ExploreHandler struct {
	ws        *WorkspaceHandler
	mgr       *session.Manager
	prefs     *preferences.Service
	watcher   *watcher.WatcherService
	convStore *conversation.Store
	draftsDir string
}

func NewExploreHandler(ws *WorkspaceHandler, mgr *session.Manager, prefs *preferences.Service, watcherSvc *watcher.WatcherService, convStore *conversation.Store, draftsDir string) *ExploreHandler {
	return &ExploreHandler{
		ws:        ws,
		mgr:       mgr,
		prefs:     prefs,
		watcher:   watcherSvc,
		convStore: convStore,
		draftsDir: draftsDir,
	}
}

func (h *ExploreHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	changeName := chi.URLParam(r, "name")

	workspacePath, ok := h.ws.workspacePath(workspaceID)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	sess, err := h.mgr.Start(workspaceID, changeName, workspacePath)
	if err != nil {
		conn.Close(websocket.StatusInternalError, "failed to start session: "+err.Error())
		return
	}

	h.serveWS(r, conn, sess, func() { h.mgr.Stop(workspaceID, changeName) }, false, workspaceID, "")
}

func (h *ExploreHandler) StopSession(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	changeName := chi.URLParam(r, "name")
	h.mgr.Stop(workspaceID, changeName)
	w.WriteHeader(http.StatusNoContent)
}

// CreateAnonymousSession creates an anonymous explore session and returns its sessionId.
// If the request body carries a resumeGhostId matching an existing exploration
// for this workspace, the session reuses that id (reattaching to a still-live
// subprocess, or starting a fresh one under the same id) instead of minting a
// new, unrelated ghost.
func (h *ExploreHandler) CreateAnonymousSession(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")

	workspacePath, ok := h.ws.workspacePath(workspaceID)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	var body struct {
		ResumeGhostID string `json:"resumeGhostId,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	resumeID := ""
	if body.ResumeGhostID != "" && h.prefs != nil {
		if h.prefs.GetExploration(body.ResumeGhostID, workspaceID) != nil {
			resumeID = body.ResumeGhostID
		}
	}

	sessionID, _, err := h.mgr.StartAnonymous(workspaceID, workspacePath, resumeID)
	if err != nil {
		http.Error(w, "failed to start session: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if resumeID != "" {
		_ = h.prefs.TouchExplorationActivity(resumeID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"sessionId": sessionID})
}

// HandleAnonymousWS is the WebSocket handler for anonymous explore sessions.
func (h *ExploreHandler) HandleAnonymousWS(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	sessionID := chi.URLParam(r, "sessionId")

	sess := h.mgr.GetAnonymous(workspaceID, sessionID)
	if sess == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	h.serveWS(r, conn, sess, func() { h.mgr.StopAnonymous(workspaceID, sessionID) }, true, workspaceID, sessionID)
}

// StopAnonymousSession stops an anonymous explore session.
func (h *ExploreHandler) StopAnonymousSession(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	sessionID := chi.URLParam(r, "sessionId")
	h.mgr.StopAnonymous(workspaceID, sessionID)
	w.WriteHeader(http.StatusNoContent)
}

// serveWS handles the bidirectional WebSocket relay for a session.
// When anonymous=true, the first user message is prefixed with "/opsx:explore ",
// a ghost record is created, and ghost_named is detected in the outgoing stream.
func (h *ExploreHandler) serveWS(r *http.Request, conn *websocket.Conn, sess *session.Session, onExpire func(), anonymous bool, workspaceID, sessionID string) {
	wsCtx, wsCancel := context.WithCancel(r.Context())
	defer wsCancel()

	// Replay history: send buffered messages before going live.
	snapshot, cursor := sess.Snapshot()
	for _, msg := range snapshot {
		if err := conn.Write(wsCtx, websocket.MessageText, msg); err != nil {
			return
		}
	}

	// A (re)connecting client should immediately see a question still awaiting
	// a reply, even though marker detection on the historical buffer above
	// only runs on newly streamed messages, not on the replayed snapshot.
	if q := sess.PendingQuestion(); q != "" {
		evtMsg, _ := json.Marshal(map[string]string{
			"type":     "ghost_question",
			"question": q,
		})
		if err := conn.Write(wsCtx, websocket.MessageText, evtMsg); err != nil {
			return
		}
	}

	// detectGhostQuestion checks a buffered message for every ghost_question
	// marker it carries, regardless of session kind (named or anonymous). A
	// single agent turn can surface more than one clarification question at
	// once (e.g. several markers concatenated across a streamed chunk or a
	// consolidated end-of-turn message); each one is broadcast as its own
	// dedicated event and marks the session as having that question pending.
	// It returns msg with every marker substring stripped out of it — markers
	// can be mixed in with ordinary response text rather than alone on their
	// own line, and that surrounding text must still reach the frontend
	// normally instead of being discarded along with the markers.
	//
	// A real agent turn is observed to surface the same marker text more than
	// once (e.g. an incremental streaming chunk that happens to carry the full
	// marker substring, followed by the turn's final consolidated message) —
	// broadcasting a second identical card for what is one logical question
	// would just clutter the thread, so a repeat of an already-pending
	// question is deduplicated: still stripped from the text, but no second
	// event.
	detectGhostQuestion := func(msg []byte) []byte {
		questions := extractAllGhostQuestions(msg)
		for _, question := range questions {
			if question == "" {
				continue
			}
			if !sess.MarkQuestionPending(question) {
				continue
			}
			evtMsg, _ := json.Marshal(map[string]string{
				"type":     "ghost_question",
				"question": question,
			})
			_ = conn.Write(wsCtx, websocket.MessageText, evtMsg)
		}
		return stripGhostQuestionMarker(msg)
	}

	// Outgoing goroutine: consume notify channel, forward new messages to WS.
	ghostNamed := false
	go func() {
		for {
			select {
			case <-wsCtx.Done():
				return
			case <-sess.Done():
				remaining, _ := sess.MessagesSince(cursor)
				for _, msg := range remaining {
					if anonymous {
						if !ghostNamed {
							if name := extractGhostNamed(msg); name != "" {
								ghostNamed = true
								h.applyGhostName(workspaceID, sessionID, name)
								evtMsg, _ := json.Marshal(map[string]string{
									"type": "ghost_named",
									"name": name,
								})
								_ = conn.Write(wsCtx, websocket.MessageText, evtMsg)
							}
						}
						if draft := extractGhostDraft(msg); draft != nil {
							h.saveGhostDraft(workspaceID, sessionID, draft)
						}
					}
					msg = detectGhostQuestion(msg)
					conn.Write(wsCtx, websocket.MessageText, msg)
				}
				conn.Write(wsCtx, websocket.MessageText, []byte(`{"type":"session_expired"}`))
				onExpire()
				conn.Close(websocket.StatusNormalClosure, "")
				return
			case <-sess.Notify():
				msgs, newCursor := sess.MessagesSince(cursor)
				cursor = newCursor
				for _, msg := range msgs {
					if anonymous {
						if !ghostNamed {
							if name := extractGhostNamed(msg); name != "" {
								ghostNamed = true
								h.applyGhostName(workspaceID, sessionID, name)
								evtMsg, _ := json.Marshal(map[string]string{
									"type": "ghost_named",
									"name": name,
								})
								_ = conn.Write(wsCtx, websocket.MessageText, evtMsg)
							}
						}
						if draft := extractGhostDraft(msg); draft != nil {
							h.saveGhostDraft(workspaceID, sessionID, draft)
						}
					}
					msg = detectGhostQuestion(msg)
					if err := conn.Write(wsCtx, websocket.MessageText, msg); err != nil {
						return
					}
				}
			}
		}
	}()

	// Incoming: WebSocket → subprocess stdin.
	ghostCreated := len(snapshot) > 0
	if anonymous && h.prefs != nil {
		if rec := h.prefs.GetExploration(sessionID, workspaceID); rec != nil {
			ghostCreated = true
		} else {
			ghostCreated = false
		}
	}
	firstSent := ghostCreated
	for {
		_, msg, err := conn.Read(wsCtx)
		if err != nil {
			break
		}
		if toolUseID, content, ok := parseNativeQuestionResponse(msg); ok {
			sess.ClearPendingQuestion()
			if anonymous && h.prefs != nil {
				_ = h.prefs.TouchExplorationActivity(sessionID)
			}
			toolResultMsg := session.BuildToolResultMessage(toolUseID, content)
			sess.Log().WriteLine("in", toolResultMsg)
			if _, err := io.WriteString(sess.Proc(), string(toolResultMsg)+"\n"); err != nil {
				break
			}
			continue
		}
		if anonymous && !firstSent {
			firstSent = true
			msg = prependExploreSkill(msg)
			if !ghostCreated {
				ghostCreated = true
				h.createGhostRecord(workspaceID, sessionID, sess)
			}
		}
		if anonymous && h.prefs != nil {
			// Anchored on the user-message side only: an assistant turn never
			// happens without a preceding user message, so this is enough to
			// track recency without touching preferences.json on every streamed delta.
			_ = h.prefs.TouchExplorationActivity(sessionID)
		}
		sess.ClearPendingQuestion()
		sess.Log().WriteLine("in", msg)
		msg = append(msg, '\n')
		if _, err := io.WriteString(sess.Proc(), string(msg)); err != nil {
			break
		}
	}
}

// createGhostRecord creates a ghost record in preferences and broadcasts ghost_card_created.
func (h *ExploreHandler) createGhostRecord(workspaceID, sessionID string, sess *session.Session) {
	if h.prefs == nil {
		return
	}
	shortID := sessionID
	if len(shortID) > 6 {
		shortID = shortID[:6]
	}
	tempName := "explore-" + shortID
	record := preferences.ExplorationRecord{
		ID:          sessionID,
		WorkspaceID: workspaceID,
		Name:        tempName,
		SessionID:   sessionID,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	_ = h.prefs.AddExploration(record)
	if h.watcher != nil {
		h.watcher.Broadcast(workspaceID, watcher.Event{Type: "ghost_card_created", Name: tempName})
	}
	if sess != nil {
		evt, _ := json.Marshal(map[string]string{
			"type": "ghost_card_created",
			"name": tempName,
		})
		sess.InjectMessage(evt)
	}
}

// applyGhostName updates the ghost record name and broadcasts ghost_named.
func (h *ExploreHandler) applyGhostName(workspaceID, sessionID, name string) {
	if h.prefs == nil {
		return
	}
	finalName := h.ensureUniqueName(workspaceID, sessionID, name)
	_ = h.prefs.UpdateExplorationName(sessionID, finalName)
	if h.watcher != nil {
		h.watcher.Broadcast(workspaceID, watcher.Event{Type: "ghost_named", Name: finalName})
	}
}

// ensureUniqueName checks if name conflicts with an existing change or exploration, adds suffix if needed.
func (h *ExploreHandler) ensureUniqueName(workspaceID, sessionID, name string) string {
	workspacePath, ok := h.ws.workspacePath(workspaceID)
	if !ok {
		return name
	}
	changes, _ := openspec.ListChanges(workspacePath)
	explorations := h.prefs.ListExplorations(workspaceID)

	taken := make(map[string]bool)
	for _, c := range changes {
		taken[c.Name] = true
	}
	for _, e := range explorations {
		if e.ID != sessionID {
			taken[e.Name] = true
		}
	}

	candidate := name
	for i := 2; taken[candidate]; i++ {
		candidate = name + "-" + string(rune('0'+i))
		if i > 9 {
			candidate = name + "-" + strings.Repeat("x", i-9)
		}
	}
	return candidate
}

// DeleteGhost stops an exploration session and removes its ghost record.
func (h *ExploreHandler) DeleteGhost(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	ghostID := chi.URLParam(r, "ghostId")

	h.mgr.StopAnonymous(workspaceID, ghostID)
	_ = h.prefs.DeleteExploration(ghostID)
	_ = h.deleteDraftFile(ghostID)
	if h.convStore != nil {
		_ = h.convStore.DeleteExplorationLogs(workspaceID, ghostID)
	}
	if h.watcher != nil {
		h.watcher.Broadcast(workspaceID, watcher.Event{Type: "exploration_deleted", Name: ghostID})
	}
	w.WriteHeader(http.StatusNoContent)
}

// PromoteGhost triggers FF for a ghost card, using an injected context if the session is expired.
func (h *ExploreHandler) PromoteGhost(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	ghostID := chi.URLParam(r, "ghostId")

	record := h.prefs.GetExploration(ghostID, workspaceID)
	if record == nil {
		http.Error(w, "exploration not found", http.StatusNotFound)
		return
	}

	workspacePath, ok := h.ws.workspacePath(workspaceID)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	var body struct {
		Context string `json:"context,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	ghostName := record.Name
	h.watcher.Broadcast(workspaceID, watcher.Event{Type: "ff_started", Name: ghostName})

	// Always start a new subprocess for FF (simpler, context injected via system prompt).
	go h.runPromoteFF(workspaceID, ghostID, ghostName, workspacePath, body.Context)

	w.WriteHeader(http.StatusAccepted)
}

// runPromoteFF starts a fresh Claude subprocess for FF with the exploration context injected.
func (h *ExploreHandler) runPromoteFF(workspaceID, ghostID, ghostName, workspacePath, explorationContext string) {
	var cfg agents.AgentConfig
	if h.mgr != nil {
		cfg = h.mgr.ResolveAgentConfig(workspaceID, "")
	} else {
		var ok bool
		cfg, ok = agents.ByID("claude")
		if !ok {
			h.watcher.Broadcast(workspaceID, watcher.Event{Type: "ff_failed", Name: ghostName, Error: "agent not found"})
			return
		}
	}

	// Try to load any local draft file to inject tasks and description into the promotion process context.
	if h.draftsDir != "" {
		draftPath := filepath.Join(h.draftsDir, ghostID+".json")
		if data, err := os.ReadFile(draftPath); err == nil {
			var draft ExplorationDraft
			if err := json.Unmarshal(data, &draft); err == nil {
				draftContext := "\n\nBased on your previous exploration, you drafted the following plan. Make sure to generate the tasks exactly matching this draft:\n"
				if draft.Description != "" {
					draftContext += "Description: " + draft.Description + "\n"
				}
				if len(draft.Tasks) > 0 {
					draftContext += "Draft Tasks:\n"
					for _, task := range draft.Tasks {
						status := "[ ]"
						if task.Done {
							status = "[x]"
						}
						draftContext += fmt.Sprintf("- %s %s\n", status, task.Text)
					}
				}
				explorationContext += draftContext
			}
		}
	}

	systemPrompt := ""
	if explorationContext != "" {
		systemPrompt = "The user explored this topic in a conversation. Here is the exploration context:\n\n" + explorationContext
	}

	var customEnv map[string]string
	langDirective := language.Directive(language.Docs, language.Resolve(language.Levels{}, ""))
	if h.prefs != nil {
		if p, err := h.prefs.Load(); err == nil && p != nil {
			customEnv = p.EnvFor(cfg.ID)
			langDirective = p.LanguageDirective(language.Docs)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proc, err := session.StartSubprocess(ctx, workspacePath, cfg, systemPrompt, "", false, nil, customEnv, false, langDirective)
	if err != nil {
		h.watcher.Broadcast(workspaceID, watcher.Event{Type: "ff_failed", Name: ghostName, Error: err.Error()})
		return
	}

	initMsg := map[string]interface{}{
		"type": "user",
		"message": map[string]string{
			"role":    "user",
			"content": "/opsx:ff " + ghostName,
		},
	}
	initBytes, _ := json.Marshal(initMsg)
	proc.Write(append(initBytes, '\n'))

	scanner := bufio.NewScanner(proc.Stdout())
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		// stdout consumed but not forwarded (FF runs silently in background)
	}

	if err := proc.Wait(); err != nil {
		h.watcher.Broadcast(workspaceID, watcher.Event{Type: "ff_failed", Name: ghostName, Error: err.Error()})
		return
	}

	changeDir := filepath.Join(workspacePath, "openspec", "changes", ghostName)
	if err := openspec.SetLaunched(changeDir, false); err != nil {
		log.Printf("[explore] failed to set launched=false for %s: %v", ghostName, err)
	}

	// FF succeeded: copy the exploration's conversation logs under the new change
	// so the change has a copy of the design/specs discussion, while leaving the
	// original logs active for further exploration.
	if h.convStore != nil {
		if err := h.convStore.CopyExplorationLogs(workspaceID, ghostID, ghostName); err != nil {
			log.Printf("[explore] failed to copy exploration logs for %s -> %s: %v", ghostID, ghostName, err)
		}
	}

	// FF succeeded: keep ghost record and draft file active for coexistence/solidification.
	// We broadcast ff_done to let the frontend know the promotion finished and the new change is on disk.
	h.watcher.Broadcast(workspaceID, watcher.Event{Type: "ff_done", Name: ghostName})
}

type DraftTask struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type ExplorationDraft struct {
	GhostID     string      `json:"ghostId"`
	WorkspaceID string      `json:"workspaceId"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Tasks       []DraftTask `json:"tasks"`
	LastSavedAt string      `json:"lastSavedAt,omitempty"`
}

func (h *ExploreHandler) deleteDraftFile(ghostID string) error {
	if h.draftsDir == "" {
		return nil
	}
	path := filepath.Join(h.draftsDir, ghostID+".json")
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (h *ExploreHandler) GetGhostDraft(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	ghostID := chi.URLParam(r, "ghostId")

	w.Header().Set("Content-Type", "application/json")

	if h.draftsDir == "" {
		http.Error(w, "drafts directory not configured", http.StatusInternalServerError)
		return
	}

	path := filepath.Join(h.draftsDir, ghostID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return a default empty draft structure
			draft := ExplorationDraft{
				GhostID:     ghostID,
				WorkspaceID: workspaceID,
				Name:        "",
				Description: "",
				Tasks:       []DraftTask{},
			}
			json.NewEncoder(w).Encode(draft)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

func (h *ExploreHandler) UpdateGhostDraft(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	ghostID := chi.URLParam(r, "ghostId")

	if h.draftsDir == "" {
		http.Error(w, "drafts directory not configured", http.StatusInternalServerError)
		return
	}

	var draft ExplorationDraft
	if err := json.NewDecoder(r.Body).Decode(&draft); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	draft.GhostID = ghostID
	draft.WorkspaceID = workspaceID
	draft.LastSavedAt = time.Now().UTC().Format(time.RFC3339)

	if err := os.MkdirAll(h.draftsDir, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := json.MarshalIndent(draft, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	path := filepath.Join(h.draftsDir, ghostID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (h *ExploreHandler) DeleteGhostDraft(w http.ResponseWriter, r *http.Request) {
	ghostID := chi.URLParam(r, "ghostId")

	if err := h.deleteDraftFile(ghostID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func extractGhostDraft(line []byte) *ExplorationDraft {
	var data map[string]interface{}
	if err := json.Unmarshal(line, &data); err != nil {
		return nil
	}
	event, _ := data["event"].(string)
	if event != "ghost_draft_updated" {
		return nil
	}
	draftMap, ok := data["draft"].(map[string]interface{})
	if !ok {
		return nil
	}
	draftBytes, err := json.Marshal(draftMap)
	if err != nil {
		return nil
	}
	var draft ExplorationDraft
	if err := json.Unmarshal(draftBytes, &draft); err != nil {
		return nil
	}
	return &draft
}

func (h *ExploreHandler) saveGhostDraft(workspaceID, sessionID string, draft *ExplorationDraft) {
	if h.draftsDir == "" {
		return
	}
	draft.GhostID = sessionID
	draft.WorkspaceID = workspaceID
	draft.LastSavedAt = time.Now().UTC().Format(time.RFC3339)

	if err := os.MkdirAll(h.draftsDir, 0755); err != nil {
		log.Printf("[explore] failed to create drafts dir: %v", err)
		return
	}

	data, err := json.MarshalIndent(draft, "", "  ")
	if err != nil {
		log.Printf("[explore] failed to marshal draft: %v", err)
		return
	}

	path := filepath.Join(h.draftsDir, sessionID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("[explore] failed to write draft file: %v", err)
		return
	}

	if h.watcher != nil {
		h.watcher.Broadcast(workspaceID, watcher.Event{Type: "draft_updated", Name: sessionID})
	}
}
