package handlers

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/go-chi/chi/v5"
)

type ActivityHandler struct {
	ws            *WorkspaceHandler
	convStore     *conversation.Store
	activityStore *activity.Store
}

func NewActivityHandler(ws *WorkspaceHandler, convStore *conversation.Store, activityStore *activity.Store) *ActivityHandler {
	return &ActivityHandler{
		ws:            ws,
		convStore:     convStore,
		activityStore: activityStore,
	}
}

func (h *ActivityHandler) GetActivity(w http.ResponseWriter, r *http.Request) {
	wsID := chi.URLParam(r, "id")
	changeName := chi.URLParam(r, "name")

	if _, ok := h.ws.workspacePath(wsID); !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	var allEntries []activity.Entry

	// 1. Read persisted activity entries from activity.Store
	if h.activityStore != nil {
		persisted, err := h.activityStore.Read(wsID, changeName)
		if err != nil {
			http.Error(w, "failed to read activity: "+err.Error(), http.StatusInternalServerError)
			return
		}
		allEntries = append(allEntries, persisted...)
	}

	// 2. Read and parse conversation runs across all kinds
	if h.convStore != nil {
		kinds, err := h.convStore.ListKinds(wsID, changeName)
		if err != nil {
			http.Error(w, "failed to list conversation kinds: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for _, kind := range kinds {
			runs, err := h.convStore.List(wsID, changeName, kind)
			if err != nil {
				continue
			}
			for _, run := range runs {
				lines, err := h.convStore.Load(wsID, changeName, kind, run.Ts)
				if err != nil {
					continue
				}
				entries, err := activity.ParseConversationLines(lines)
				if err != nil {
					continue
				}
				allEntries = append(allEntries, entries...)
			}
		}
	}

	// 3. Sort chronologically by Ts ascending
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Ts < allEntries[j].Ts
	})

	if allEntries == nil {
		allEntries = []activity.Entry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allEntries)
}
