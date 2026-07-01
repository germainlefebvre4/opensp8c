package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
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
func extractGhostNamed(line []byte) string {
	s := string(line)
	if !strings.Contains(s, `"ghost_named"`) {
		return ""
	}
	return session.ExtractGhostNamed(line)
}

type ExploreHandler struct {
	ws        *WorkspaceHandler
	mgr       *session.Manager
	prefs     *preferences.Service
	watcher   *watcher.WatcherService
	convStore *conversation.Store
}

func NewExploreHandler(ws *WorkspaceHandler, mgr *session.Manager, prefs *preferences.Service, watcherSvc *watcher.WatcherService, convStore *conversation.Store) *ExploreHandler {
	return &ExploreHandler{ws: ws, mgr: mgr, prefs: prefs, watcher: watcherSvc, convStore: convStore}
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
					if anonymous && !ghostNamed {
						if name := extractGhostNamed(msg); name != "" {
							ghostNamed = true
							h.applyGhostName(workspaceID, sessionID, name)
						}
					}
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
					if anonymous && !ghostNamed {
						if name := extractGhostNamed(msg); name != "" {
							ghostNamed = true
							h.applyGhostName(workspaceID, sessionID, name)
						}
					}
					if err := conn.Write(wsCtx, websocket.MessageText, msg); err != nil {
						return
					}
				}
			}
		}
	}()

	// Incoming: WebSocket → subprocess stdin.
	firstSent := anonymous && len(snapshot) > 0
	ghostCreated := len(snapshot) > 0 // ghost already created if we're reconnecting
	for {
		_, msg, err := conn.Read(wsCtx)
		if err != nil {
			break
		}
		if anonymous && !firstSent {
			firstSent = true
			msg = prependExploreSkill(msg)
			if !ghostCreated {
				ghostCreated = true
				h.createGhostRecord(workspaceID, sessionID)
			}
		}
		if anonymous && h.prefs != nil {
			// Anchored on the user-message side only: an assistant turn never
			// happens without a preceding user message, so this is enough to
			// track recency without touching preferences.json on every streamed delta.
			_ = h.prefs.TouchExplorationActivity(sessionID)
		}
		sess.Log().WriteLine("in", msg)
		msg = append(msg, '\n')
		if _, err := io.WriteString(sess.Proc(), string(msg)); err != nil {
			break
		}
	}
}

// createGhostRecord creates a ghost record in preferences and broadcasts ghost_card_created.
func (h *ExploreHandler) createGhostRecord(workspaceID, sessionID string) {
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
	cfg, ok := agents.ByID("claude")
	if !ok {
		h.watcher.Broadcast(workspaceID, watcher.Event{Type: "ff_failed", Name: ghostName, Error: "agent not found"})
		return
	}

	systemPrompt := ""
	if explorationContext != "" {
		systemPrompt = "The user explored this topic in a conversation. Here is the exploration context:\n\n" + explorationContext
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proc, err := session.StartSubprocess(ctx, workspacePath, cfg, systemPrompt, "", false, nil)
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

	// FF succeeded: move the exploration's conversation logs under the new change
	// before clearing the ghost record, so they survive the promotion.
	if h.convStore != nil {
		if err := h.convStore.MoveExplorationLogs(workspaceID, ghostID, ghostName); err != nil {
			log.Printf("[explore] failed to move exploration logs for %s -> %s: %v", ghostID, ghostName, err)
		}
	}

	// FF succeeded: clean up ghost record before broadcasting done
	_ = h.prefs.DeleteExploration(ghostID)
	h.watcher.Broadcast(workspaceID, watcher.Event{Type: "exploration_deleted", Name: ghostID})
	h.watcher.Broadcast(workspaceID, watcher.Event{Type: "ff_done", Name: ghostName})
}
