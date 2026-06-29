package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/glefebvre/opensp8c/internal/session"
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

type ExploreHandler struct {
	ws  *WorkspaceHandler
	mgr *session.Manager
}

func NewExploreHandler(ws *WorkspaceHandler, mgr *session.Manager) *ExploreHandler {
	return &ExploreHandler{ws: ws, mgr: mgr}
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

	h.serveWS(w, r, conn, sess, func() { h.mgr.Stop(workspaceID, changeName) }, false)
}

func (h *ExploreHandler) StopSession(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	changeName := chi.URLParam(r, "name")
	h.mgr.Stop(workspaceID, changeName)
	w.WriteHeader(http.StatusNoContent)
}

// CreateAnonymousSession creates an anonymous explore session and returns its sessionId.
func (h *ExploreHandler) CreateAnonymousSession(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")

	workspacePath, ok := h.ws.workspacePath(workspaceID)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	sessionID, _, err := h.mgr.StartAnonymous(workspaceID, workspacePath)
	if err != nil {
		http.Error(w, "failed to start session: "+err.Error(), http.StatusInternalServerError)
		return
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

	h.serveWS(w, r, conn, sess, func() { h.mgr.StopAnonymous(workspaceID, sessionID) }, true)
}

// StopAnonymousSession stops an anonymous explore session.
func (h *ExploreHandler) StopAnonymousSession(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	sessionID := chi.URLParam(r, "sessionId")
	h.mgr.StopAnonymous(workspaceID, sessionID)
	w.WriteHeader(http.StatusNoContent)
}

// serveWS handles the bidirectional WebSocket relay for a session.
// When anonymous is true, the first user message is prefixed with "/opsx:explore "
// to trigger the explore skill with the project context.
func (h *ExploreHandler) serveWS(_ http.ResponseWriter, r *http.Request, conn *websocket.Conn, sess *session.Session, onExpire func(), anonymous bool) {
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
	go func() {
		for {
			select {
			case <-wsCtx.Done():
				return
			case <-sess.Done():
				remaining, _ := sess.MessagesSince(cursor)
				for _, msg := range remaining {
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
					if err := conn.Write(wsCtx, websocket.MessageText, msg); err != nil {
						return
					}
				}
			}
		}
	}()

	// Incoming: WebSocket → subprocess stdin.
	firstSent := anonymous && len(snapshot) > 0
	for {
		_, msg, err := conn.Read(wsCtx)
		if err != nil {
			break
		}
		if anonymous && !firstSent {
			firstSent = true
			msg = prependExploreSkill(msg)
		}
		msg = append(msg, '\n')
		if _, err := io.WriteString(sess.Proc(), string(msg)); err != nil {
			break
		}
	}
}
