package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/go-chi/chi/v5"
	"nhooyr.io/websocket"
)

func TestGhostDraftCRUD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "opensp8c-test-drafts-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	h := &ExploreHandler{
		draftsDir: tmpDir,
	}

	workspaceID := "ws123"
	ghostID := "ghost456"

	// 1. GET - Should return an empty default draft
	req := httptest.NewRequest("GET", "/workspaces/"+workspaceID+"/explorations/"+ghostID+"/draft", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	rctx.URLParams.Add("ghostId", ghostID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.GetGhostDraft(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected GET code 200, got %d", rec.Code)
	}

	var defaultDraft ExplorationDraft
	if err := json.NewDecoder(rec.Body).Decode(&defaultDraft); err != nil {
		t.Fatalf("failed to decode GET body: %v", err)
	}

	if defaultDraft.GhostID != ghostID || defaultDraft.WorkspaceID != workspaceID {
		t.Errorf("default draft mismatch: %+v", defaultDraft)
	}
	if len(defaultDraft.Tasks) != 0 {
		t.Errorf("expected no default tasks, got %d", len(defaultDraft.Tasks))
	}

	// 2. PUT - Update/Create draft
	payload := ExplorationDraft{
		Name:        "test-name",
		Description: "test desc",
		Tasks: []DraftTask{
			{ID: "t1", Text: "Task 1", Done: false},
			{ID: "t2", Text: "Task 2", Done: true},
		},
	}
	bodyBytes, _ := json.Marshal(payload)
	putReq := httptest.NewRequest("PUT", "/workspaces/"+workspaceID+"/explorations/"+ghostID+"/draft", bytes.NewReader(bodyBytes))
	putReq = putReq.WithContext(context.WithValue(putReq.Context(), chi.RouteCtxKey, rctx))

	putRec := httptest.NewRecorder()
	h.UpdateGhostDraft(putRec, putReq)

	if putRec.Code != http.StatusOK {
		t.Errorf("expected PUT code 200, got %d", putRec.Code)
	}

	var savedDraft ExplorationDraft
	if err := json.NewDecoder(putRec.Body).Decode(&savedDraft); err != nil {
		t.Fatalf("failed to decode PUT response: %v", err)
	}

	if savedDraft.Name != "test-name" || len(savedDraft.Tasks) != 2 {
		t.Errorf("saved draft incorrect: %+v", savedDraft)
	}

	// Verify file is actually on disk
	filePath := filepath.Join(tmpDir, ghostID+".json")
	if _, err := os.Stat(filePath); err != nil {
		t.Errorf("expected file to exist at %s, got err: %v", filePath, err)
	}

	// 3. GET - Retrieve again, should return the saved draft from disk
	getReq2 := httptest.NewRequest("GET", "/workspaces/"+workspaceID+"/explorations/"+ghostID+"/draft", nil)
	getReq2 = getReq2.WithContext(context.WithValue(getReq2.Context(), chi.RouteCtxKey, rctx))

	getRec2 := httptest.NewRecorder()
	h.GetGhostDraft(getRec2, getReq2)

	if getRec2.Code != http.StatusOK {
		t.Errorf("expected GET (2) code 200, got %d", getRec2.Code)
	}

	var retrievedDraft ExplorationDraft
	if err := json.NewDecoder(getRec2.Body).Decode(&retrievedDraft); err != nil {
		t.Fatalf("failed to decode GET (2) response: %v", err)
	}

	if retrievedDraft.Name != "test-name" || len(retrievedDraft.Tasks) != 2 || retrievedDraft.Tasks[1].Done != true {
		t.Errorf("retrieved draft incorrect: %+v", retrievedDraft)
	}

	// 4. DELETE - Delete the draft
	delReq := httptest.NewRequest("DELETE", "/workspaces/"+workspaceID+"/explorations/"+ghostID+"/draft", nil)
	delReq = delReq.WithContext(context.WithValue(delReq.Context(), chi.RouteCtxKey, rctx))

	delRec := httptest.NewRecorder()
	h.DeleteGhostDraft(delRec, delReq)

	if delRec.Code != http.StatusNoContent {
		t.Errorf("expected DELETE code 204, got %d", delRec.Code)
	}

	// Verify file is deleted from disk
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted, got err: %v", err)
	}

	// 5. GET - Retrieve after delete, should fall back to empty default
	getReq3 := httptest.NewRequest("GET", "/workspaces/"+workspaceID+"/explorations/"+ghostID+"/draft", nil)
	getReq3 = getReq3.WithContext(context.WithValue(getReq3.Context(), chi.RouteCtxKey, rctx))

	getRec3 := httptest.NewRecorder()
	h.GetGhostDraft(getRec3, getReq3)

	var finalDraft ExplorationDraft
	_ = json.NewDecoder(getRec3.Body).Decode(&finalDraft)
	if len(finalDraft.Tasks) != 0 {
		t.Errorf("expected empty tasks after delete, got %d", len(finalDraft.Tasks))
	}
}

// dialExploreWS starts an httptest server wired to serveWS for sess, dials it,
// and returns the client connection (caller must close it) plus a context
// bound to the test's lifetime.
func dialExploreWS(t *testing.T, sess *session.Session) (*websocket.Conn, context.Context) {
	t.Helper()
	return dialExploreWSMode(t, sess, false)
}

func dialExploreWSMode(t *testing.T, sess *session.Session, anonymous bool) (*websocket.Conn, context.Context) {
	t.Helper()
	h := &ExploreHandler{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		h.serveWS(r, conn, sess, func() {}, anonymous, "ws1", "sid1")
	}))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })

	return conn, ctx
}

// TestServeWSReplaysPendingQuestionOnConnect verifies that a client connecting
// (or reconnecting) to a session that already has a pending clarification
// question immediately receives a "ghost_question" event with its text,
// deterministically (no dependency on live stdout streaming timing).
func TestServeWSReplaysPendingQuestionOnConnect(t *testing.T) {
	sess := session.NewTestSession(nil)
	sess.SetPendingQuestion("Quel est le périmètre exact ?")

	conn, ctx := dialExploreWS(t, sess)

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	var evt map[string]string
	if err := json.Unmarshal(data, &evt); err != nil {
		t.Fatalf("failed to unmarshal event: %v (data: %s)", err, data)
	}
	if evt["type"] != "ghost_question" {
		t.Fatalf("expected first event to be ghost_question, got: %s", data)
	}
	if evt["question"] != "Quel est le périmètre exact ?" {
		t.Errorf("unexpected question text: %q", evt["question"])
	}
}

// TestServeWSBroadcastsGhostQuestion verifies that when the subprocess stdout
// buffer receives a ghost_question marker while a client is connected,
// serveWS emits a dedicated "ghost_question" WebSocket event carrying the
// question text, and records the pending state on the session.
func TestServeWSBroadcastsGhostQuestion(t *testing.T) {
	sess := session.NewTestSession(nil)
	// Seeded before connecting so it is deterministically part of the replayed
	// history, letting the test detect once the live goroutine has taken over.
	sess.InjectMessage([]byte(`{"type":"bootstrap"}`))

	conn, ctx := dialExploreWS(t, sess)

	// Drain the replayed bootstrap message first.
	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatalf("read (bootstrap) failed: %v", err)
	}

	marker := []byte(`{"event":"ghost_question","question":"Quel est le périmètre exact ?"}`)
	sess.InjectMessage(marker)

	found := false
	for i := 0; i < 5 && !found; i++ {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		var evt map[string]string
		if err := json.Unmarshal(data, &evt); err != nil {
			continue
		}
		if evt["type"] == "ghost_question" {
			if evt["question"] != "Quel est le périmètre exact ?" {
				t.Errorf("unexpected question text: %q", evt["question"])
			}
			found = true
		}
	}
	if !found {
		t.Fatal("expected a ghost_question event to be broadcast")
	}

	if got := sess.PendingQuestion(); got != "Quel est le périmètre exact ?" {
		t.Errorf("expected session pending question to be set, got %q", got)
	}
}

// TestServeWSStripsEmbeddedGhostQuestionMarkerWithoutDroppingSurroundingText
// verifies the fix for a real bug caught in manual end-to-end testing: Claude
// often emits the ghost_question marker mixed into the same event as
// substantial surrounding response text (not alone on its own line as the
// system prompt asks for), e.g. a consolidated end-of-turn message. serveWS
// must strip just the marker substring and still forward the rest of that
// text to the frontend — not silently discard the whole event, which would
// leave the user with no visible response at all.
func TestServeWSStripsEmbeddedGhostQuestionMarkerWithoutDroppingSurroundingText(t *testing.T) {
	sess := session.NewTestSession(nil)
	sess.InjectMessage([]byte(`{"type":"bootstrap"}`))

	conn, ctx := dialExploreWS(t, sess)

	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatalf("read (bootstrap) failed: %v", err)
	}

	// Realistic shape observed in production: the marker embedded at the end
	// of a longer assistant message, not alone on its own line.
	embedded := []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Voici mon analyse du code existant.\n\n{\"event\":\"ghost_question\",\"question\":\"Quel est le périmètre exact ?\"}"}]}}`)
	sess.InjectMessage(embedded)

	var gotEvent, gotStrippedText bool
	for i := 0; i < 5 && (!gotEvent || !gotStrippedText); i++ {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		var evt map[string]interface{}
		if err := json.Unmarshal(data, &evt); err != nil {
			continue
		}
		if evt["type"] == "ghost_question" {
			gotEvent = true
			if evt["question"] != "Quel est le périmètre exact ?" {
				t.Errorf("unexpected question text: %v", evt["question"])
			}
			continue
		}
		if strings.Contains(string(data), "Voici mon analyse") {
			gotStrippedText = true
			if strings.Contains(string(data), "ghost_question") {
				t.Errorf("expected the marker substring to be stripped from the forwarded message, got: %s", data)
			}
		}
	}

	if !gotEvent {
		t.Error("expected a ghost_question event to be broadcast")
	}
	if !gotStrippedText {
		t.Error("expected the surrounding response text to still be forwarded, with only the marker stripped out")
	}
}

// TestServeWSDeduplicatesRepeatedGhostQuestionMarker verifies a fix for a real
// duplication caught in manual end-to-end testing: a single agent turn can
// surface the same marker text more than once (e.g. an incremental streaming
// chunk that happens to carry the full marker substring, followed by the
// turn's final consolidated message). Only one ghost_question event — and one
// QuestionCard — should reach the frontend per logical question.
func TestServeWSDeduplicatesRepeatedGhostQuestionMarker(t *testing.T) {
	sess := session.NewTestSession(nil)
	sess.InjectMessage([]byte(`{"type":"bootstrap"}`))

	conn, ctx := dialExploreWS(t, sess)

	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatalf("read (bootstrap) failed: %v", err)
	}

	marker := []byte(`{"event":"ghost_question","question":"Quel est le périmètre exact ?"}`)
	// Same logical question observed twice, as a real agent turn does.
	sess.InjectMessage(marker)
	sess.InjectMessage(marker)
	sess.InjectMessage([]byte(`{"type":"sentinel"}`))

	eventCount := 0
	sawSentinel := false
	for i := 0; i < 5 && !sawSentinel; i++ {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		var evt map[string]string
		if err := json.Unmarshal(data, &evt); err == nil && evt["type"] == "ghost_question" {
			eventCount++
			continue
		}
		if strings.Contains(string(data), "sentinel") {
			sawSentinel = true
		}
	}

	if !sawSentinel {
		t.Fatal("expected to reach the sentinel message")
	}
	if eventCount != 1 {
		t.Errorf("expected exactly 1 ghost_question event for a repeated identical marker, got %d", eventCount)
	}
}

// TestServeWSBroadcastsMultipleGhostQuestionsFromSameTurn verifies that when a
// single agent turn surfaces more than one ghost_question marker at once
// (e.g. two markers concatenated in the same consolidated message), serveWS
// emits a dedicated ghost_question event for each one and strips every
// marker substring from the forwarded text, leaving no raw JSON behind.
func TestServeWSBroadcastsMultipleGhostQuestionsFromSameTurn(t *testing.T) {
	sess := session.NewTestSession(nil)
	sess.InjectMessage([]byte(`{"type":"bootstrap"}`))

	conn, ctx := dialExploreWS(t, sess)

	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatalf("read (bootstrap) failed: %v", err)
	}

	// Two distinct questions concatenated in the same consolidated turn text,
	// as observed when an agent asks several clarification questions at once.
	consolidated := []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Deux questions pour cadrer :\n\n{\"event\":\"ghost_question\",\"question\":\"Quelle stack utiliser ?\"}\n{\"event\":\"ghost_question\",\"question\":\"Quel est le budget ?\"}"}]}}`)
	sess.InjectMessage(consolidated)
	sess.InjectMessage([]byte(`{"type":"sentinel"}`))

	var gotQuestions []string
	var gotForwardedText string
	sawSentinel := false
	for i := 0; i < 8 && !sawSentinel; i++ {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		var evt map[string]interface{}
		if err := json.Unmarshal(data, &evt); err != nil {
			continue
		}
		if evt["type"] == "ghost_question" {
			if q, ok := evt["question"].(string); ok {
				gotQuestions = append(gotQuestions, q)
			}
			continue
		}
		if strings.Contains(string(data), "Deux questions") {
			gotForwardedText = string(data)
			continue
		}
		if evt["type"] == "sentinel" {
			sawSentinel = true
		}
	}

	if !sawSentinel {
		t.Fatal("expected to reach the sentinel message")
	}
	if len(gotQuestions) != 2 {
		t.Fatalf("expected 2 ghost_question events, got %d: %v", len(gotQuestions), gotQuestions)
	}
	if gotQuestions[0] != "Quelle stack utiliser ?" || gotQuestions[1] != "Quel est le budget ?" {
		t.Errorf("unexpected question texts: %v", gotQuestions)
	}
	if gotForwardedText == "" {
		t.Fatal("expected the surrounding response text to still be forwarded")
	}
	if strings.Contains(gotForwardedText, "ghost_question") {
		t.Errorf("expected no raw ghost_question JSON to leak into the forwarded text, got: %s", gotForwardedText)
	}
}

// captureWriteCloser is an io.WriteCloser that buffers writes for inspection.
type captureWriteCloser struct {
	bytes.Buffer
}

func (c *captureWriteCloser) Close() error { return nil }

// TestServeWSWritesToolResultOnNativeQuestionResponse verifies that when the
// client answers a native_question card, serveWS translates the frontend's
// native_question_response envelope into a well-formed tool_result stream-json
// payload referencing the original tool_use_id, and writes it to the
// subprocess stdin (rather than forwarding the client envelope verbatim).
func TestServeWSWritesToolResultOnNativeQuestionResponse(t *testing.T) {
	stdin := &captureWriteCloser{}
	proc := session.NewTestSubprocess(stdin, nil, "claude")
	sess := session.NewTestSession(proc)
	sess.SetPendingQuestion("Quel format préfères-tu ?")

	conn, ctx := dialExploreWS(t, sess)

	// Drain the pending-question replay emitted on connect.
	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatalf("read (pending question replay) failed: %v", err)
	}

	respMsg, _ := json.Marshal(map[string]string{
		"type":      "native_question_response",
		"toolUseId": "toolu_123",
		"content":   "Réponse de l'utilisateur",
	})
	if err := conn.Write(ctx, websocket.MessageText, respMsg); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for stdin.Len() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if stdin.Len() == 0 {
		t.Fatal("timed out waiting for tool_result to be written to stdin")
	}

	var got struct {
		Type    string `json:"type"`
		Message struct {
			Role    string `json:"role"`
			Content []struct {
				Type      string `json:"type"`
				ToolUseID string `json:"tool_use_id"`
				Content   string `json:"content"`
			} `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(stdin.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal stdin payload: %v (data: %s)", err, stdin.String())
	}
	if got.Type != "user" || got.Message.Role != "user" {
		t.Errorf("unexpected envelope: %+v", got)
	}
	if len(got.Message.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(got.Message.Content))
	}
	block := got.Message.Content[0]
	if block.Type != "tool_result" || block.ToolUseID != "toolu_123" || block.Content != "Réponse de l'utilisateur" {
		t.Errorf("unexpected tool_result block: %+v", block)
	}

	if got := sess.PendingQuestion(); got != "" {
		t.Errorf("expected pending question to be cleared after tool_result, got %q", got)
	}
}

// TestServeWSDeliversLiveMessagesPastBufferCapacity reproduces a long Explore
// session: a connected client must keep receiving every event, in order, well
// beyond the 500-entry buffer window (regression: the client used to stall
// once the buffer saturated, leaving the final result undelivered).
func TestServeWSDeliversLiveMessagesPastBufferCapacity(t *testing.T) {
	sess := session.NewTestSession(nil)
	conn, ctx := dialExploreWS(t, sess)

	const total = 1500
	go func() {
		for i := 0; i < total; i++ {
			sess.InjectMessage([]byte(fmt.Sprintf(`{"type":"assistant","n":%d}`, i)))
			if i%20 == 0 {
				time.Sleep(time.Millisecond)
			}
		}
		sess.InjectMessage([]byte(`{"type":"result","n":-1}`))
	}()

	next := 0
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("stalled after %d events: %v", next, err)
		}
		var evt struct {
			Type string `json:"type"`
			N    int    `json:"n"`
		}
		if err := json.Unmarshal(data, &evt); err != nil {
			t.Fatalf("bad event %s: %v", data, err)
		}
		if evt.Type == "result" {
			break
		}
		if evt.N != next {
			t.Fatalf("out of order or lost: got n=%d, want %d", evt.N, next)
		}
		next++
	}
	if next != total {
		t.Fatalf("received %d events, want %d", next, total)
	}
}

// TestAnonymousRelayNeverLeaksGhostNamedMarker replays a real anonymous
// exploration stream (ghost_named fragmented token by token, then repeated in
// the consolidated assistant message) through serveWS and checks that no
// relayed text carries any part of the marker, while the ghost_named event and
// the surrounding text still come through.
func TestAnonymousRelayNeverLeaksGhostNamedMarker(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "ghost_named_stream.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	sess := session.NewTestSession(nil)
	sess.InjectMessage([]byte(`{"type":"bootstrap"}`))
	conn, ctx := dialExploreWSMode(t, sess, true)
	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatalf("read (bootstrap) failed: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		sess.InjectMessage([]byte(line))
	}
	sess.InjectMessage([]byte(`{"type":"sentinel"}`))

	var streamed strings.Builder
	var consolidated string
	var namedEvent string
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		var msg map[string]interface{}
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("invalid relayed JSON: %v", err)
		}
		switch msg["type"] {
		case "sentinel":
			goto done
		case "ghost_named":
			namedEvent, _ = msg["name"].(string)
			continue
		case "assistant":
			consolidated = string(data)
		}
		if evt, ok := msg["event"].(map[string]interface{}); ok && evt["type"] == "content_block_delta" {
			if delta, ok := evt["delta"].(map[string]interface{}); ok {
				if text, ok := delta["text"].(string); ok {
					streamed.WriteString(text)
				}
			}
		}
	}
done:
	if namedEvent != "rethink-application-ergonomics" {
		t.Errorf("expected ghost_named event for rethink-application-ergonomics, got %q", namedEvent)
	}
	if got := streamed.String(); got != "Let me look at what the app is today before I ask anything." {
		t.Errorf("unexpected streamed text: %q", got)
	}
	if strings.Contains(consolidated, "ghost_named") || strings.Contains(consolidated, "event") {
		t.Errorf("consolidated message leaks the marker: %s", consolidated)
	}
	if !strings.Contains(consolidated, `"text":"Let me look at what the app is today before I ask anything."`) {
		t.Errorf("consolidated text should start without a blank line: %s", consolidated)
	}
}

func TestGhostNamedRelayFilterReleasesHeldTextOnBlockStop(t *testing.T) {
	f := newGhostNamedRelayFilter()
	mk := func(text string) []byte {
		return deltaMessage(1, text)
	}
	if out := f.Process(mk("{\"eve")); len(out) != 0 {
		t.Fatalf("partial prefix should be held, got %d messages", len(out))
	}
	out := f.Process([]byte(`{"type":"stream_event","event":{"type":"content_block_stop","index":1}}`))
	if len(out) != 2 || !strings.Contains(string(out[0]), `{\"eve`) {
		t.Fatalf("expected held text released before block stop, got %q", out)
	}
}
